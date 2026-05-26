package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	mcppkg "github.com/sreagent/sreagent/internal/pkg/mcp"
)

// MCPTool represents a tool discovered from an MCP server.
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// MCPToolCallResult is the result of calling an MCP tool (service-layer type).
type MCPToolCallResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// MCPContent is a single content item in a tool call result.
type MCPContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

// MCPClient is a lightweight MCP client that delegates to the internal/pkg/mcp package.
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

// ListTools connects to an MCP server and enumerates available tools.
func (c *MCPClient) ListTools(ctx context.Context, url string, headers map[string]string) ([]MCPTool, error) {
	client := mcppkg.NewClientWithHTTPClient(url, headers, c.httpClient)
	mcpTools, err := client.ListTools(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]MCPTool, 0, len(mcpTools))
	for _, t := range mcpTools {
		result = append(result, MCPTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}
	return result, nil
}

// CallTool invokes a tool on the MCP server.
func (c *MCPClient) CallTool(ctx context.Context, url string, headers map[string]string, toolName string, args map[string]interface{}) (*MCPToolCallResult, error) {
	client := mcppkg.NewClientWithHTTPClient(url, headers, c.httpClient)
	result, err := client.CallTool(ctx, toolName, args)
	if err != nil {
		return nil, err
	}

	mcpResult := &MCPToolCallResult{
		IsError: result.IsError,
	}
	for _, content := range result.Content {
		mcpResult.Content = append(mcpResult.Content, MCPContent{
			Type:     content.Type,
			Text:     content.Text,
			Data:     content.Data,
			MimeType: content.MimeType,
		})
	}
	return mcpResult, nil
}
