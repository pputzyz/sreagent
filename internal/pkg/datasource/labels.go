package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

// PromLabelResult is what Prometheus /api/v1/labels and /api/v1/label/{name}/values return.
type PromLabelResult struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

// FetchAllLabels queries a Prometheus-compatible datasource for all label key→values.
// Supported types: prometheus, victoriametrics. Others return nil, nil.
// It first fetches all label names, then for each label fetches all values.
// To keep it bounded, it limits to the first 200 label names and 500 values per label.
func FetchAllLabels(ctx context.Context, dsType, endpoint, authType, authConfig string) (map[string][]string, error) {
	switch dsType {
	case "prometheus", "victoriametrics":
		return fetchPromLabels(ctx, endpoint, authType, authConfig)
	default:
		return nil, nil
	}
}

func fetchPromLabels(ctx context.Context, endpoint, authType, authConfig string) (map[string][]string, error) {
	base := strings.TrimRight(endpoint, "/")

	// Step 1: get all label names
	body, status, err := httpGetBody(ctx, base+"/api/v1/labels", authType, authConfig)
	if err != nil {
		return nil, fmt.Errorf("fetch label names: %w", err)
	}
	if status != 200 {
		return nil, fmt.Errorf("fetch label names: HTTP %d", status)
	}
	var namesResp PromLabelResult
	if err := json.Unmarshal(body, &namesResp); err != nil || namesResp.Status != "success" {
		return nil, fmt.Errorf("parse label names response")
	}

	names := namesResp.Data
	if len(names) > 200 {
		names = names[:200]
	}

	result := make(map[string][]string, len(names))

	// Step 2: for each label name, fetch values concurrently (max 8 in-flight)
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(8)
	var mu sync.Mutex

	for _, name := range names {
		name := name
		eg.Go(func() error {
			url := fmt.Sprintf("%s/api/v1/label/%s/values", base, name)
			body, status, err := httpGetBody(ctx, url, authType, authConfig)
			if err != nil || status != 200 {
				return nil // skip failed labels, don't abort the whole batch
			}
			var valResp PromLabelResult
			if err := json.Unmarshal(body, &valResp); err != nil || valResp.Status != "success" {
				return nil
			}
			vals := valResp.Data
			if len(vals) > 500 {
				vals = vals[:500]
			}
			mu.Lock()
			result[name] = vals
			mu.Unlock()
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return result, nil
}
