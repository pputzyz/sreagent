package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// jsonRPCRequest is a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// jsonRPCResponse is a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// connectSSE connects to the SSE endpoint and extracts the message endpoint URL.
// The caller must hold c.mu.
func (c *Client) connectSSE(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.serverURL, nil)
	if err != nil {
		return "", fmt.Errorf("create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Use a longer timeout for the SSE connection itself
	httpClient := &http.Client{Timeout: 60 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("SSE connect: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("SSE endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	// Parse SSE events to find the "endpoint" event
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)

	var eventType string
	timeout := time.After(DefaultSSETimeout)

	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for endpoint event from SSE")
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return "", fmt.Errorf("SSE scan: %w", err)
			}
			return "", fmt.Errorf("SSE connection closed before endpoint event")
		}

		line := scanner.Text()

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}

		if strings.HasPrefix(line, "data:") && eventType == "endpoint" {
			endpoint := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if endpoint == "" {
				return "", fmt.Errorf("empty endpoint in SSE event")
			}

			// Resolve relative URL
			if !strings.HasPrefix(endpoint, "http") {
				base := c.serverURL
				if idx := strings.LastIndex(base, "/"); idx >= 0 {
					base = base[:idx+1]
				}
				endpoint = base + strings.TrimPrefix(endpoint, "/")
			}

			return endpoint, nil
		}

		// Reset event type after blank line
		if line == "" {
			eventType = ""
		}
	}
}

// sendRPC sends a JSON-RPC request and parses the response.
func (c *Client) sendRPC(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := c.nextReqID()

	reqBody := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.messageURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Try JSON-RPC response first
	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err == nil && rpcResp.JSONRPC == "2.0" {
		if rpcResp.Error != nil {
			return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
		}
		return rpcResp.Result, nil
	}

	// Try SSE response format
	result, err := c.parseSSEResponse(respBody, id)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// sendNotification sends a JSON-RPC notification (no response expected).
func (c *Client) sendNotification(ctx context.Context, method string, params interface{}) error {
	reqBody := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.messageURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	return nil
}

// parseSSEResponse parses an SSE-formatted response and extracts the JSON-RPC result
// matching the given request ID.
func (c *Client) parseSSEResponse(data []byte, requestID int) (json.RawMessage, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), 64*1024)

	var eventType string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") && eventType == "message" {
			d := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			var r jsonRPCResponse
			if err := json.Unmarshal([]byte(d), &r); err == nil && r.ID == requestID {
				if r.Error != nil {
					return nil, fmt.Errorf("RPC error %d: %s", r.Error.Code, r.Error.Message)
				}
				return r.Result, nil
			}
		}
		if line == "" {
			eventType = ""
		}
	}

	return nil, fmt.Errorf("no valid JSON-RPC response found in SSE stream")
}
