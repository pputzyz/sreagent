package datasource

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// VictoriaLogsChecker checks VictoriaLogs health.
type VictoriaLogsChecker struct{}

// CheckHealth performs a two-phase probe:
//  1. GET /health — basic liveness
//  2. POST /select/logsql/query with `* | limit 0` — verifies the LogsQL engine responds
func (c *VictoriaLogsChecker) CheckHealth(ctx context.Context, endpoint, authType, authConfig string) HealthResult {
	base := strings.TrimRight(endpoint, "/")

	// ── Phase 1: liveness ────────────────────────────────────────────────────
	if err := httpGet(ctx, base+"/health", authType, authConfig); err != nil {
		return HealthResult{Healthy: false, LatencyMs: -1,
			Message: fmt.Sprintf("liveness probe failed: %v", err)}
	}

	// ── Phase 2: LogsQL engine probe ─────────────────────────────────────────
	start := time.Now()
	apiURL := base + "/select/logsql/query"
	now := time.Now()
	form := url.Values{}
	form.Set("query", "*")
	form.Set("start", now.Add(-1*time.Minute).Format(time.RFC3339))
	form.Set("end", now.Format(time.RFC3339))
	form.Set("limit", "0") // zero limit — just test API responds, no data transfer

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return HealthResult{Healthy: false, LatencyMs: time.Since(start).Milliseconds(),
			Message: fmt.Sprintf("failed to build query request: %v", err)}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := applyAuth(req, authType, authConfig); err != nil {
		return HealthResult{Healthy: false, LatencyMs: -1,
			Message: fmt.Sprintf("auth config error: %v", err)}
	}

	resp, err := httpClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return HealthResult{Healthy: false, LatencyMs: latency,
			Message: fmt.Sprintf("query API unreachable: %v", err)}
	}
	defer func() { _ = resp.Body.Close() }()
	// VictoriaLogs returns 200 OK even with 0 results; any non-200 is a problem.
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return HealthResult{Healthy: false, LatencyMs: latency,
			Message: fmt.Sprintf("query API returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))}
	}

	return HealthResult{Healthy: true, LatencyMs: latency, Message: "VictoriaLogs is healthy"}
}

// vlogsQueryLimit is the maximum number of log lines returned per query.
// If the result count equals this limit, the actual count may be higher.
const vlogsQueryLimit = 10000

// VictoriaLogsInstantQuery executes a LogsQL query against VictoriaLogs and returns
// the count of matching log entries as a single QueryResult.
//
// The expression is a LogsQL query string (e.g. `error level:error _time:5m`).
// The result value is the number of log lines returned by the query.
// The lookback parameter controls the time window; defaults to 5 minutes if <= 0.
//
// VictoriaLogs API: POST /select/logsql/query
// Response format: NDJSON — one JSON object per line.
func VictoriaLogsInstantQuery(ctx context.Context, endpoint, authType, authConfig, expression string, lookback time.Duration) ([]QueryResult, error) {
	if lookback <= 0 {
		lookback = 5 * time.Minute
	}

	apiURL := strings.TrimRight(endpoint, "/") + "/select/logsql/query"

	// Build form body with the LogsQL query and the configured time window
	now := time.Now()
	form := url.Values{}
	form.Set("query", expression)
	form.Set("start", now.Add(-lookback).Format(time.RFC3339))
	form.Set("end", now.Format(time.RFC3339))
	form.Set("limit", fmt.Sprintf("%d", vlogsQueryLimit))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create logsql query request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := applyAuth(req, authType, authConfig); err != nil {
		return nil, fmt.Errorf("logsql query auth: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("logsql query request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("logsql query returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse NDJSON response: each line is a JSON log entry.
	// We count the number of lines to get the log count value.
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB per line max

	var count float64
	// Collect distinct stream label values across ALL entries to enrich result labels.
	// Using maps to track unique values per field — multiple hosts/jobs may report.
	jobs := make(map[string]bool)
	instances := make(map[string]bool)
	hosts := make(map[string]bool)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		count++

		// Parse every entry to collect all unique stream label values
		var entry map[string]interface{}
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if v, ok := entry["job"].(string); ok && v != "" {
			jobs[v] = true
		}
		if v, ok := entry["instance"].(string); ok && v != "" {
			instances[v] = true
		}
		if v, ok := entry["host"].(string); ok && v != "" {
			hosts[v] = true
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read logsql response: %w", err)
	}

	// P1-5: Warn if count hit the query limit — actual count may be higher
	if count >= vlogsQueryLimit {
		log.Printf("WARNING: VictoriaLogs query hit limit=%d, actual count may be higher for rule expression=%s", vlogsQueryLimit, expression)
	}

	// Build labels: merge collected stream labels with query info.
	// Join multiple values with comma so alert annotations show all affected hosts.
	labels := make(map[string]string, 5)
	labels["__logsql__"] = "true"
	labels["query"] = expression
	if len(jobs) > 0 {
		labels["job"] = joinMapKeys(jobs)
	}
	if len(instances) > 0 {
		labels["instance"] = joinMapKeys(instances)
	}
	if len(hosts) > 0 {
		labels["host"] = joinMapKeys(hosts)
	}

	return []QueryResult{
		{
			MetricName: "logsql_match_count",
			Labels:     labels,
			Values: []DataPoint{
				{Timestamp: now, Value: count},
			},
		},
	}, nil
}

// joinMapKeys returns a sorted, comma-separated string of all keys in the map.
func joinMapKeys(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

// LogEntry represents a single log line returned by VictoriaLogs.
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Message   string                 `json:"message"`
	Labels    map[string]interface{} `json:"labels"`
}

// LogQueryResponse holds the parsed result of a VictoriaLogs log query.
type LogQueryResponse struct {
	Entries    []LogEntry `json:"entries"`
	Total      int        `json:"total"`
	Truncated  bool       `json:"truncated"`
}

// QueryLogsParams holds parameters for a log query.
type QueryLogsParams struct {
	Query string
	Start time.Time
	End   time.Time
	Limit int
}

// QueryLogs executes a LogsQL query and returns the actual log entries.
//
// VictoriaLogs API: POST /select/logsql/query
// Response format: NDJSON — one JSON object per line.
// Each line contains the log fields including _msg (log message) and _time (timestamp).
func QueryLogs(ctx context.Context, endpoint, authType, authConfig string, params QueryLogsParams) (*LogQueryResponse, error) {
	apiURL := strings.TrimRight(endpoint, "/") + "/select/logsql/query"

	if params.Limit <= 0 {
		params.Limit = 100
	}
	if params.Limit > 10000 {
		params.Limit = 10000
	}

	form := url.Values{}
	form.Set("query", params.Query)
	form.Set("start", params.Start.Format(time.RFC3339))
	form.Set("end", params.End.Format(time.RFC3339))
	form.Set("limit", fmt.Sprintf("%d", params.Limit))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create log query request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := applyAuth(req, authType, authConfig); err != nil {
		return nil, fmt.Errorf("log query auth: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("log query request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("log query returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse NDJSON response
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB per line max

	entries := make([]LogEntry, 0, params.Limit)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(line, &raw); err != nil {
			continue // skip malformed lines
		}

		entry := LogEntry{
			Labels: make(map[string]interface{}),
		}

		// Extract timestamp from _time field
		if t, ok := raw["_time"]; ok {
			switch v := t.(type) {
			case string:
				if ts, err := time.Parse(time.RFC3339Nano, v); err == nil {
					entry.Timestamp = ts
				}
			case float64:
				// Unix nanoseconds
				entry.Timestamp = time.Unix(0, int64(v))
			}
			delete(raw, "_time")
		}

		// Extract message from _msg field
		if msg, ok := raw["_msg"]; ok {
			if s, ok := msg.(string); ok {
				entry.Message = s
			}
			delete(raw, "_msg")
		}

		// Store all remaining fields as labels
		for k, v := range raw {
			entry.Labels[k] = v
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read log query response: %w", err)
	}

	return &LogQueryResponse{
		Entries:   entries,
		Total:     len(entries),
		Truncated: len(entries) >= params.Limit,
	}, nil
}

// LogHistogramBucket represents a single time bucket in the histogram.
type LogHistogramBucket struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int64     `json:"count"`
}

// LogHistogramResponse holds the parsed histogram data.
type LogHistogramResponse struct {
	Buckets []LogHistogramBucket `json:"buckets"`
	Total   int64                `json:"total"`
}

// QueryLogHistogram fetches log hit counts over time buckets using the
// VictoriaLogs /select/logsql/hits endpoint with a step parameter.
func QueryLogHistogram(ctx context.Context, endpoint, authType, authConfig, expression string, start, end time.Time, step string) (*LogHistogramResponse, error) {
	apiURL := strings.TrimRight(endpoint, "/") + "/select/logsql/hits"

	form := url.Values{}
	form.Set("query", expression)
	form.Set("start", start.Format(time.RFC3339))
	form.Set("end", end.Format(time.RFC3339))
	if step != "" {
		form.Set("step", step)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create histogram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := applyAuth(req, authType, authConfig); err != nil {
		return nil, fmt.Errorf("histogram auth: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("histogram request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("histogram returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read histogram response: %w", err)
	}

	// Try format: JSON with "timestamps" and "values" arrays
	var hitsResp struct {
		Timestamps []int64 `json:"timestamps"`
		Values     []int64 `json:"values"`
	}
	if err := json.Unmarshal(body, &hitsResp); err == nil && len(hitsResp.Timestamps) > 0 {
		buckets := make([]LogHistogramBucket, 0, len(hitsResp.Timestamps))
		var total int64
		for i, ts := range hitsResp.Timestamps {
			var count int64
			if i < len(hitsResp.Values) {
				count = hitsResp.Values[i]
			}
			buckets = append(buckets, LogHistogramBucket{Timestamp: time.Unix(ts, 0), Count: count})
			total += count
		}
		return &LogHistogramResponse{Buckets: buckets, Total: total}, nil
	}

	// Try format: array of {timestamp, count} objects
	var altResp []struct {
		Timestamp int64 `json:"timestamp"`
		Count     int64 `json:"count"`
	}
	if err := json.Unmarshal(body, &altResp); err == nil && len(altResp) > 0 {
		buckets := make([]LogHistogramBucket, len(altResp))
		var total int64
		for i, b := range altResp {
			buckets[i] = LogHistogramBucket{Timestamp: time.Unix(b.Timestamp, 0), Count: b.Count}
			total += b.Count
		}
		return &LogHistogramResponse{Buckets: buckets, Total: total}, nil
	}

	// Try NDJSON format (one JSON object per line)
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	var buckets []LogHistogramBucket
	var total int64
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		var ts time.Time
		var count int64
		if v, ok := entry["_time"]; ok {
			switch t := v.(type) {
			case string:
				ts, _ = time.Parse(time.RFC3339Nano, t)
			case float64:
				ts = time.Unix(0, int64(t))
			}
		}
		if v, ok := entry["_count"]; ok {
			if c, ok := v.(float64); ok {
				count = int64(c)
			}
		}
		buckets = append(buckets, LogHistogramBucket{Timestamp: ts, Count: count})
		total += count
	}
	if len(buckets) > 0 {
		return &LogHistogramResponse{Buckets: buckets, Total: total}, nil
	}

	return nil, fmt.Errorf("victorialogs histogram: failed to parse response in any known format (tried timestamps+values, array, ndjson): %s", string(body))
}
