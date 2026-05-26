package processors

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sreagent/sreagent/internal/engine/pipeline"
	"github.com/sreagent/sreagent/internal/model"
)

func init() {
	pipeline.Register("callback", newCallback)
}

// callbackProcessor sends the event to an external URL via HTTP POST.
type callbackProcessor struct {
	URL           string            `json:"url"`
	Method        string            `json:"method"`
	Headers       map[string]string `json:"headers"`
	Timeout       int               `json:"timeout"` // seconds
	SkipSSLVerify bool              `json:"skip_ssl_verify"`
}

func newCallback(config map[string]interface{}) (pipeline.Processor, error) {
	p := &callbackProcessor{
		Method:  "POST",
		Timeout: 10,
	}
	if v, ok := config["url"].(string); ok {
		p.URL = v
	}
	if v, ok := config["method"].(string); ok {
		p.Method = v
	}
	if v, ok := config["headers"].(map[string]interface{}); ok {
		p.Headers = make(map[string]string)
		for k, val := range v {
			if sv, ok := val.(string); ok {
				p.Headers[k] = sv
			}
		}
	}
	if v, ok := config["timeout"].(float64); ok {
		p.Timeout = int(v)
	}
	if v, ok := config["skip_ssl_verify"].(bool); ok {
		p.SkipSSLVerify = v
	}
	if p.URL == "" {
		return nil, fmt.Errorf("callback: url is required")
	}
	return p, nil
}

func (p *callbackProcessor) Process(ctx context.Context, event *model.AlertEvent) (*model.AlertEvent, string, error) {
	body, err := json.Marshal(event)
	if err != nil {
		return event, "", fmt.Errorf("callback: failed to marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, p.Method, p.URL, bytes.NewReader(body))
	if err != nil {
		return event, "", fmt.Errorf("callback: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{
		Timeout: time.Duration(p.Timeout) * time.Second,
	}
	if p.SkipSSLVerify {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		// Fire-and-forget: log error but don't block pipeline
		return event, fmt.Sprintf("callback: request failed: %v", err), nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return event, fmt.Sprintf("callback: got HTTP %d from %s", resp.StatusCode, p.URL), nil
	}
	return event, fmt.Sprintf("callback: sent to %s (HTTP %d)", p.URL, resp.StatusCode), nil
}
