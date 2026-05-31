package datasource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// ElasticsearchChecker checks Elasticsearch health.
type ElasticsearchChecker struct{}

// CheckHealth performs a two-phase probe:
//  1. GET / — connectivity + version extraction from root response
//  2. POST /_search with {"size":0} — verifies search API works
func (c *ElasticsearchChecker) CheckHealth(ctx context.Context, endpoint, authType, authConfig string) HealthResult {
	base := strings.TrimRight(endpoint, "/")

	// ── Phase 1: connectivity + version ─────────────────────────────────────
	body, status, err := httpGetBody(ctx, base+"/", authType, authConfig)
	if err != nil {
		return HealthResult{Healthy: false, LatencyMs: -1,
			Message: fmt.Sprintf("connectivity probe failed: %v", err)}
	}
	if status != http.StatusOK {
		return HealthResult{Healthy: false, LatencyMs: -1,
			Message: fmt.Sprintf("connectivity probe returned HTTP %d", status)}
	}

	var rootResp struct {
		Version struct {
			Number string `json:"number"`
		} `json:"version"`
	}
	_ = json.Unmarshal(body, &rootResp)
	version := rootResp.Version.Number

	// ── Phase 2: search API probe ───────────────────────────────────────────
	start := time.Now()
	searchBody := []byte(`{"size":0}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/_search", bytes.NewReader(searchBody))
	if err != nil {
		return HealthResult{Healthy: false, LatencyMs: time.Since(start).Milliseconds(),
			Message: fmt.Sprintf("failed to build search request: %v", err)}
	}
	req.Header.Set("Content-Type", "application/json")
	if err := applyAuth(req, authType, authConfig); err != nil {
		return HealthResult{Healthy: false, LatencyMs: -1,
			Message: fmt.Sprintf("auth config error: %v", err)}
	}

	resp, err := httpClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return HealthResult{Healthy: false, LatencyMs: latency,
			Message: fmt.Sprintf("search API unreachable: %v", err)}
	}
	defer func() { _ = resp.Body.Close() }()
	// ES returns 200 for _search even with 0 results; 404 means no indices but API works
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		return HealthResult{Healthy: false, LatencyMs: latency,
			Message: fmt.Sprintf("search API returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))}
	}

	msg := "Elasticsearch is healthy"
	if version != "" {
		msg = fmt.Sprintf("Elasticsearch %s is healthy", version)
	}

	// ── Phase 3 (optional): verify auth credentials ────────────────────────
	if authType == "basic" || authType == "bearer" {
		authReq, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/_security/_authenticate", nil)
		if err == nil {
			if authErr := applyAuth(authReq, authType, authConfig); authErr == nil {
				authResp, authErr := httpClient.Do(authReq)
				if authErr == nil {
					defer func() { _ = authResp.Body.Close() }()
					if authResp.StatusCode == http.StatusUnauthorized || authResp.StatusCode == http.StatusForbidden {
						return HealthResult{Healthy: false, LatencyMs: latency,
							Message: fmt.Sprintf("authentication failed: HTTP %d", authResp.StatusCode), Version: version}
					}
					msg += " (credentials verified)"
				}
			}
		}
	}

	return HealthResult{Healthy: true, LatencyMs: latency, Message: msg, Version: version}
}

// ElasticsearchQueryLogsParams holds parameters for an Elasticsearch log query.
type ElasticsearchQueryLogsParams struct {
	Index     string
	Query     string // Lucene query string
	DateField string // timestamp field name, default "@timestamp"
	Start     time.Time
	End       time.Time
	Limit     int
	From      int // pagination offset
}

// ElasticsearchQueryLogs executes a log query against Elasticsearch using _msearch.
func ElasticsearchQueryLogs(ctx context.Context, endpoint, authType, authConfig string, params ElasticsearchQueryLogsParams) (*LogQueryResponse, error) {
	base := strings.TrimRight(endpoint, "/")

	if params.Index == "" {
		return nil, fmt.Errorf("elasticsearch index is required")
	}
	if err := validateESIndexName(params.Index); err != nil {
		return nil, err
	}
	if params.DateField == "" {
		params.DateField = "@timestamp"
	}
	if params.Limit <= 0 {
		params.Limit = 100
	}
	if params.Limit > 10000 {
		params.Limit = 10000
	}

	// Build _msearch NDJSON body
	headerLine, _ := json.Marshal(map[string]interface{}{"index": params.Index})

	boolQuery := map[string]interface{}{
		"filter": []interface{}{
			map[string]interface{}{
				"range": map[string]interface{}{
					params.DateField: map[string]interface{}{
						"gte":    params.Start.UnixMilli(),
						"lte":    params.End.UnixMilli(),
						"format": "epoch_millis",
					},
				},
			},
		},
	}

	queryClause := map[string]interface{}{
		"bool": boolQuery,
	}

	if params.Query != "" {
		boolQuery["must"] = []interface{}{
			map[string]interface{}{
				"query_string": map[string]interface{}{
					"query": params.Query,
				},
			},
		}
	}

	bodyLine, _ := json.Marshal(map[string]interface{}{
		"query":            queryClause,
		"track_total_hits": true,
		"sort": []interface{}{
			map[string]interface{}{params.DateField: "desc"},
		},
		"size": params.Limit,
		"from": params.From,
	})

	ndjson := string(headerLine) + "\n" + string(bodyLine) + "\n"

	apiURL := base + "/_msearch"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(ndjson))
	if err != nil {
		return nil, fmt.Errorf("failed to create msearch request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	if err := applyAuth(req, authType, authConfig); err != nil {
		return nil, fmt.Errorf("msearch auth: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("msearch request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("msearch returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read msearch response: %w", err)
	}

	var msearchResp struct {
		Responses []struct {
			Hits struct {
				Total struct {
					Value int64 `json:"value"`
				} `json:"total"`
				Hits []struct {
					Source map[string]interface{} `json:"_source"`
					Fields map[string]interface{} `json:"fields"`
				} `json:"hits"`
			} `json:"hits"`
		} `json:"responses"`
	}
	if err := json.Unmarshal(respBody, &msearchResp); err != nil {
		return nil, fmt.Errorf("failed to parse msearch response: %w", err)
	}

	if len(msearchResp.Responses) == 0 {
		return &LogQueryResponse{Entries: []LogEntry{}, Total: 0, Truncated: false}, nil
	}

	resp0 := msearchResp.Responses[0]
	entries := make([]LogEntry, 0, len(resp0.Hits.Hits))

	// P2-10: Include ECS standard message fields
	messageFields := []string{"log", "message", "msg", "_msg", "body", "event.message", "log.message"}

	for _, hit := range resp0.Hits.Hits {
		entry := LogEntry{
			Labels: make(map[string]interface{}),
		}

		// Extract timestamp
		if ts, ok := hit.Source[params.DateField]; ok {
			switch v := ts.(type) {
			case string:
				if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
					entry.Timestamp = t
				} else if t, err := time.Parse("2006-01-02T15:04:05.000Z", v); err == nil {
					entry.Timestamp = t
				}
			case float64:
				entry.Timestamp = time.UnixMilli(int64(v))
			}
		}

		// Extract message from common message-like fields (P2-10: supports dot-path nested fields)
		found := false
		for _, mf := range messageFields {
			val := lookupNested(hit.Source, mf)
			if val != nil {
				if s, ok := val.(string); ok {
					entry.Message = s
					found = true
					break
				}
			}
		}

		// Store remaining fields as labels
		for k, v := range hit.Source {
			if k == params.DateField {
				continue
			}
			isMessageField := false
			for _, mf := range messageFields {
				if k == mf {
					isMessageField = true
					break
				}
			}
			if found && isMessageField {
				continue
			}
			entry.Labels[k] = v
		}

		// Fallback: if no message field found, use first string field (P2-13: sorted alphabetically for determinism)
		if !found {
			keys := make([]string, 0, len(hit.Source))
			for k := range hit.Source {
				if k == params.DateField {
					continue
				}
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				v := hit.Source[k]
				if s, ok := v.(string); ok && len(s) > 0 && len(s) < 4096 {
					entry.Message = s
					delete(entry.Labels, k)
					break
				}
			}
		}

		entries = append(entries, entry)
	}

	total := int(resp0.Hits.Total.Value)
	if total == 0 {
		total = len(entries)
	}

	return &LogQueryResponse{
		Entries:   entries,
		Total:     total,
		Truncated: len(entries) >= params.Limit,
	}, nil
}

// ElasticsearchQueryHistogramParams holds parameters for an ES histogram query.
type ElasticsearchQueryHistogramParams struct {
	Index     string
	Query     string
	DateField string
	Start     time.Time
	End       time.Time
	Step      string // e.g. "1m", "5m", "1h" — converted to fixed_interval
}

// ElasticsearchQueryHistogram fetches log hit counts over time buckets using
// Elasticsearch date_histogram aggregation via _msearch.
func ElasticsearchQueryHistogram(ctx context.Context, endpoint, authType, authConfig string, params ElasticsearchQueryHistogramParams) (*LogHistogramResponse, error) {
	base := strings.TrimRight(endpoint, "/")

	if params.Index == "" {
		return nil, fmt.Errorf("elasticsearch index is required")
	}
	if err := validateESIndexName(params.Index); err != nil {
		return nil, err
	}
	if params.DateField == "" {
		params.DateField = "@timestamp"
	}
	if params.Step == "" {
		params.Step = "1m"
	}

	headerLine, _ := json.Marshal(map[string]interface{}{"index": params.Index})

	boolQuery := map[string]interface{}{
		"filter": []interface{}{
			map[string]interface{}{
				"range": map[string]interface{}{
					params.DateField: map[string]interface{}{
						"gte":    params.Start.UnixMilli(),
						"lte":    params.End.UnixMilli(),
						"format": "epoch_millis",
					},
				},
			},
		},
	}

	if params.Query != "" {
		boolQuery["must"] = []interface{}{
			map[string]interface{}{
				"query_string": map[string]interface{}{
					"query": params.Query,
				},
			},
		}
	}

	bodyLine, _ := json.Marshal(map[string]interface{}{
		"query": map[string]interface{}{
			"bool": boolQuery,
		},
		"size": 0,
		"aggs": map[string]interface{}{
			"histogram": map[string]interface{}{
				"date_histogram": map[string]interface{}{
					"field":          params.DateField,
					"fixed_interval": params.Step,
				},
			},
		},
	})

	ndjson := string(headerLine) + "\n" + string(bodyLine) + "\n"

	apiURL := base + "/_msearch"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(ndjson))
	if err != nil {
		return nil, fmt.Errorf("failed to create histogram msearch request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	if err := applyAuth(req, authType, authConfig); err != nil {
		return nil, fmt.Errorf("histogram msearch auth: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("histogram msearch request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("histogram msearch returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read histogram response: %w", err)
	}

	var msearchResp struct {
		Responses []struct {
			Aggregations struct {
				Histogram struct {
					Buckets []struct {
						Key         int64  `json:"key"`
						KeyAsString string `json:"key_as_string"`
						DocCount    int64  `json:"doc_count"`
					} `json:"buckets"`
				} `json:"histogram"`
			} `json:"aggregations"`
		} `json:"responses"`
	}
	if err := json.Unmarshal(respBody, &msearchResp); err != nil {
		return nil, fmt.Errorf("failed to parse histogram response: %w", err)
	}

	if len(msearchResp.Responses) == 0 {
		return &LogHistogramResponse{Buckets: []LogHistogramBucket{}, Total: 0}, nil
	}

	resp0 := msearchResp.Responses[0]
	buckets := make([]LogHistogramBucket, 0, len(resp0.Aggregations.Histogram.Buckets))
	var total int64

	for _, b := range resp0.Aggregations.Histogram.Buckets {
		ts := time.UnixMilli(b.Key)
		buckets = append(buckets, LogHistogramBucket{Timestamp: ts, Count: b.DocCount})
		total += b.DocCount
	}

	return &LogHistogramResponse{Buckets: buckets, Total: total}, nil
}

// FieldInfo holds a field name and type from an index mapping.
type FieldInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ElasticsearchGetIndices lists all non-hidden indices via _cat/indices.
func ElasticsearchGetIndices(ctx context.Context, endpoint, authType, authConfig string) ([]string, error) {
	base := strings.TrimRight(endpoint, "/")
	body, status, err := httpGetBody(ctx, base+"/_cat/indices?format=json&s=index", authType, authConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to list indices: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("_cat/indices returned HTTP %d: %s", status, string(body))
	}

	var indices []struct {
		Index string `json:"index"`
	}
	if err := json.Unmarshal(body, &indices); err != nil {
		return nil, fmt.Errorf("failed to parse indices response: %w", err)
	}

	names := make([]string, 0, len(indices))
	for _, idx := range indices {
		if idx.Index != "" && !strings.HasPrefix(idx.Index, ".") {
			names = append(names, idx.Index)
		}
	}
	return names, nil
}

// ElasticsearchGetFields extracts field names and types from an index mapping.
func ElasticsearchGetFields(ctx context.Context, endpoint, authType, authConfig, index string) ([]FieldInfo, error) {
	base := strings.TrimRight(endpoint, "/")
	body, status, err := httpGetBody(ctx, base+"/"+url.PathEscape(index)+"/_mapping", authType, authConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get mapping: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("_mapping returned HTTP %d: %s", status, string(body))
	}

	// The mapping response is nested: { "index_name": { "mappings": { "properties": { ... } } } }
	var mappingResp map[string]interface{}
	if err := json.Unmarshal(body, &mappingResp); err != nil {
		return nil, fmt.Errorf("failed to parse mapping response: %w", err)
	}

	fields := make([]FieldInfo, 0)
	for _, indexVal := range mappingResp {
		indexMap, ok := indexVal.(map[string]interface{})
		if !ok {
			continue
		}
		mappings, ok := indexMap["mappings"].(map[string]interface{})
		if !ok {
			continue
		}
		properties, ok := mappings["properties"].(map[string]interface{})
		if !ok {
			continue
		}
		extractFields(properties, "", &fields)
	}
	return fields, nil
}

// extractFields recursively extracts field names and types from ES mapping properties.
func extractFields(properties map[string]interface{}, prefix string, fields *[]FieldInfo) {
	for name, val := range properties {
		prop, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		fullName := name
		if prefix != "" {
			fullName = prefix + "." + name
		}
		fieldType, _ := prop["type"].(string)
		*fields = append(*fields, FieldInfo{Name: fullName, Type: fieldType})

		// Recurse into nested properties
		if subProps, ok := prop["properties"].(map[string]interface{}); ok {
			extractFields(subProps, fullName, fields)
		}
	}
}

// validateESIndexName rejects Elasticsearch index names that could match
// unintended indices or be used for path traversal. Wildcards (* ?) and
// leading dashes (which ES interprets as negation in multi-index patterns)
// are disallowed.
func validateESIndexName(index string) error {
	if index == "" {
		return fmt.Errorf("elasticsearch index is required")
	}
	if strings.HasPrefix(index, "-") {
		return fmt.Errorf("elasticsearch index name must not start with '-': %s", index)
	}
	if strings.ContainsAny(index, "*?") {
		return fmt.Errorf("elasticsearch index name must not contain wildcards (* or ?): %s", index)
	}
	return nil
}

// lookupNested retrieves a value from a map using dot-separated path notation.
// For example, lookupNested(m, "event.message") returns m["event"]["message"] if it exists.
func lookupNested(m map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = m
	for _, part := range parts {
		cm, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current, ok = cm[part]
		if !ok {
			return nil
		}
	}
	return current
}
