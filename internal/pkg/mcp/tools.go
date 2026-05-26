package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

type toolsListResult struct {
	Tools []Tool `json:"tools"`
}

// ListTools discovers available tools from the MCP server.
// It automatically connects and initializes the client if needed.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	if err := c.Connect(ctx); err != nil {
		return nil, err
	}

	result, err := c.sendRPC(ctx, "tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("tools/list: %w", err)
	}

	if result == nil {
		return []Tool{}, nil
	}

	var listResult toolsListResult
	if err := json.Unmarshal(result, &listResult); err != nil {
		return nil, fmt.Errorf("parse tools list: %w", err)
	}

	return listResult.Tools, nil
}

// CallTool invokes a tool on the MCP server with the given name and arguments.
// It automatically connects and initializes the client if needed.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallResult, error) {
	if err := c.Connect(ctx); err != nil {
		return nil, err
	}

	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}

	result, err := c.sendRPC(ctx, "tools/call", params)
	if err != nil {
		return nil, fmt.Errorf("tools/call %q: %w", name, err)
	}

	if result == nil {
		return &CallResult{}, nil
	}

	var callResult CallResult
	if err := json.Unmarshal(result, &callResult); err != nil {
		return nil, fmt.Errorf("parse tool call result: %w", err)
	}

	return &callResult, nil
}
