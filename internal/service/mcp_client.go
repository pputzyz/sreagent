package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// MCPTool represents a tool discovered from an MCP server.
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// mcpJSONRPCRequest is a JSON-RPC 2.0 request for MCP.
type mcpJSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// mcpJSONRPCResponse is a JSON-RPC 2.0 response from MCP.
type mcpJSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *mcpRPCError    `json:"error,omitempty"`
}

type mcpRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpToolsListResult struct {
	Tools []MCPTool `json:"tools"`
}

// MCPClient is a lightweight MCP SSE client for discovering tools.
type MCPClient struct {
	httpClient *http.Client
}

// NewMCPClient creates a new MCPClient with a 30s timeout.
func NewMCPClient() *MCPClient {
	return &MCPClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// TestConnection attempts to connect to the SSE endpoint and verify it responds.
func (c *MCPClient) TestConnection(ctx context.Context, url string, headers map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// ListTools connects to an MCP server via SSE and enumerates available tools.
func (c *MCPClient) ListTools(ctx context.Context, url string, headers map[string]string) ([]MCPTool, error) {
	// Step 1: Connect to SSE endpoint to get the message endpoint
	messageURL, err := c.connectSSE(ctx, url, headers)
	if err != nil {
		return nil, fmt.Errorf("SSE connect: %w", err)
	}

	// Step 2: Send initialize request
	if err := c.sendJSONRPC(ctx, messageURL, headers, "initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "sreagent",
			"version": "1.0.0",
		},
	}); err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}

	// Step 3: Send initialized notification
	_ = c.sendJSONRPCNotification(ctx, messageURL, headers, "notifications/initialized", nil)

	// Step 4: Send tools/list request
	resp, err := c.sendJSONRPCWithResponse(ctx, messageURL, headers, "tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("tools/list: %w", err)
	}

	var result mcpToolsListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse tools: %w", err)
	}

	return result.Tools, nil
}

// connectSSE connects to the SSE endpoint and extracts the message endpoint URL.
func (c *MCPClient) connectSSE(ctx context.Context, url string, headers map[string]string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("SSE endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	// Parse SSE events to find the "endpoint" event
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)

	var eventType string
	timeout := time.After(10 * time.Second)

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
				// Parse base URL and resolve relative path
				base := url
				if idx := strings.LastIndex(base, "/"); idx >= 0 {
					base = base[:idx+1]
				}
				endpoint = base + strings.TrimPrefix(endpoint, "/")
			}

			resp.Body.Close()
			return endpoint, nil
		}

		// Reset event type after data line
		if line == "" {
			eventType = ""
		}
	}
}

// sendJSONRPC sends a JSON-RPC request and reads the response.
func (c *MCPClient) sendJSONRPC(ctx context.Context, url string, headers map[string]string, method string, params interface{}) error {
	_, err := c.sendJSONRPCWithResponse(ctx, url, headers, method, params)
	return err
}

// sendJSONRPCNotification sends a JSON-RPC notification (no response expected).
func (c *MCPClient) sendJSONRPCNotification(ctx context.Context, url string, headers map[string]string, method string, params interface{}) error {
	reqBody := mcpJSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return nil
}

// sendJSONRPCWithResponse sends a JSON-RPC request and parses the response.
func (c *MCPClient) sendJSONRPCWithResponse(ctx context.Context, url string, headers map[string]string, method string, params interface{}) (*mcpJSONRPCResponse, error) {
	var rpcID int
	switch method {
	case "initialize":
		rpcID = 1
	case "tools/list":
		rpcID = 2
	default:
		rpcID = 99
	}

	reqBody := mcpJSONRPCRequest{
		JSONRPC: "2.0",
		ID:      rpcID,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Try JSON-RPC response first
	var rpcResp mcpJSONRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err == nil && rpcResp.JSONRPC == "2.0" {
		if rpcResp.Error != nil {
			return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
		}
		return &rpcResp, nil
	}

	// Try SSE response format
	scanner := bufio.NewScanner(bytes.NewReader(respBody))
	scanner.Buffer(make([]byte, 64*1024), 64*1024)
	var eventType string
	var mu sync.Mutex
	var result *mcpJSONRPCResponse

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") && eventType == "message" {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			var r mcpJSONRPCResponse
			if err := json.Unmarshal([]byte(data), &r); err == nil && r.ID == rpcID {
				mu.Lock()
				result = &r
				mu.Unlock()
			}
		}
		if line == "" {
			eventType = ""
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if result != nil {
		if result.Error != nil {
			return nil, fmt.Errorf("RPC error %d: %s", result.Error.Code, result.Error.Message)
		}
		return result, nil
	}

	return nil, fmt.Errorf("unexpected response format (status %d): %s", resp.StatusCode, string(respBody))
}
