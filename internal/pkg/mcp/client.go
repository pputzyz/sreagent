package mcp

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sreagent/sreagent/internal/pkg/safehttp"
)

const (
	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 30 * time.Second
	// DefaultSSETimeout is the timeout for waiting for the SSE endpoint event.
	DefaultSSETimeout = 10 * time.Second
)

// Tool represents an MCP tool definition returned by tools/list.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

// CallResult is the result of calling an MCP tool.
type CallResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content is a single content item in a tool call result.
type Content struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

// Client connects to an MCP server via SSE transport and discovers/calls tools.
type Client struct {
	serverURL string
	headers   map[string]string
	client    *http.Client

	mu          sync.Mutex
	connected   bool
	messageURL  string // resolved message endpoint from SSE
	initialized bool   // true after successful initialize handshake
	nextID      int    // JSON-RPC request ID counter
}

// NewClient creates a new MCP client for the given server URL and optional headers.
func NewClient(serverURL string, headers map[string]string) *Client {
	return &Client{
		serverURL: serverURL,
		headers:   headers,
		client:    safehttp.NewInternalClient(DefaultTimeout),
		nextID:    1,
	}
}

// NewClientWithHTTPClient creates a new MCP client with a custom HTTP client.
func NewClientWithHTTPClient(serverURL string, headers map[string]string, httpClient *http.Client) *Client {
	return &Client{
		serverURL: serverURL,
		headers:   headers,
		client:    httpClient,
		nextID:    1,
	}
}

// Connect establishes the SSE connection and performs the MCP initialize handshake.
// It is safe to call multiple times; subsequent calls are no-ops.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return nil
	}

	if !c.connected {
		messageURL, err := c.connectSSE(ctx)
		if err != nil {
			return fmt.Errorf("MCP SSE connect: %w", err)
		}
		c.messageURL = messageURL
		c.connected = true
	}

	// Perform initialize handshake
	if err := c.initialize(ctx); err != nil {
		return fmt.Errorf("MCP initialize: %w", err)
	}

	c.initialized = true
	return nil
}

// initialize sends the JSON-RPC initialize request and the initialized notification.
func (c *Client) initialize(ctx context.Context) error {
	// Send initialize request
	_, err := c.sendRPC(ctx, "initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "sreagent",
			"version": "1.0.0",
		},
	})
	if err != nil {
		return fmt.Errorf("initialize request: %w", err)
	}

	// Send initialized notification (no response expected)
	_ = c.sendNotification(ctx, "notifications/initialized", nil)

	return nil
}

// nextReqID returns the next JSON-RPC request ID.
func (c *Client) nextReqID() int {
	id := c.nextID
	c.nextID++
	return id
}
