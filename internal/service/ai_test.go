package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/pkg/crypto"
)

// Test_callLLMWithToolsCustom_bounded_rounds verifies that when the LLM keeps
// returning tool_calls, the loop terminates after maxRounds and does not loop
// indefinitely.
func Test_callLLMWithToolsCustom_bounded_rounds(t *testing.T) {
	var callCount int64

	// Mock LLM server that always returns a tool_call response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt64(&callCount, 1)
		resp := chatCompletionResponse{
			Choices: []struct {
				Message struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"message"`
			}{
				{
					Message: struct {
						Content   string     `json:"content"`
						ToolCalls []ToolCall `json:"tool_calls,omitempty"`
					}{
						ToolCalls: []ToolCall{
							{
								ID:   "call_" + string(rune('0'+n)),
								Type: "function",
								Function: struct {
									Name      string `json:"name"`
									Arguments string `json:"arguments"`
								}{
									Name:      "fake_tool",
									Arguments: `{}`,
								},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := &AIService{
		logger: zap.NewNop(),
		client: server.Client(),
	}

	cfg := AIConfig{
		Provider: "openai",
		Model:    "test-model",
		BaseURL:  server.URL,
	}

	maxRounds := 3
	_, records, err := svc.callLLMWithToolsCustom(
		context.Background(),
		cfg,
		"system prompt",
		"user prompt",
		nil, // tools — not needed for this test
		func(_ context.Context, _ string, _ map[string]interface{}) (string, error) {
			return "tool result", nil
		},
		maxRounds,
	)

	require.NoError(t, err)
	assert.Equal(t, int64(maxRounds), atomic.LoadInt64(&callCount),
		"LLM should be called exactly maxRounds times")
	assert.Len(t, records, maxRounds,
		"should have exactly maxRounds tool call records")
	assert.Equal(t, "fake_tool", records[0].ToolName,
		"tool call records should capture the tool name")
}

// Test_ai_apikey_encrypted_roundtrip verifies that SaveAIConfig encrypts the
// API key and GetAIConfig decrypts it back to the original value.
// This is a unit test of the crypto primitives used by the AI config layer.
func Test_ai_apikey_encrypted_roundtrip(t *testing.T) {
	// Ensure SREAGENT_SECRET_KEY is set for encryption.
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Setenv("SREAGENT_SECRET_KEY", key)

	// Reset the crypto key so it picks up the env var.
	// We test via the public crypto API directly.
	testKeys := []string{
		"sk-abcdefghijklmnop1234567890",
		"short",
		"",
		"key-with-special-chars-!@#$%^&*()",
	}

	for _, apiKey := range testKeys {
		t.Run("key_len_"+string(rune('0'+len(apiKey)/10)), func(t *testing.T) {
			if apiKey == "" {
				// Empty string round-trips to empty.
				enc, err := crypto.EncryptString(apiKey)
				require.NoError(t, err)
				assert.Equal(t, "", enc, "empty string should not be encrypted")
				return
			}

			enc, err := crypto.EncryptString(apiKey)
			require.NoError(t, err)
			assert.NotEmpty(t, enc)
			assert.True(t, crypto.IsEncrypted(enc), "encrypted value should have enc: prefix")
			assert.NotEqual(t, apiKey, enc, "encrypted value should differ from plaintext")

			dec, err := crypto.DecryptString(enc)
			require.NoError(t, err)
			assert.Equal(t, apiKey, dec, "decrypted value must match original")
		})
	}
}

// Test_query_datasource_tool_range_limit verifies that the query_datasource
// tool rejects time ranges exceeding 24 hours.
func Test_query_datasource_tool_range_limit(t *testing.T) {
	registry := NewAIToolRegistry(zap.NewNop())

	// Create a mock DataSourceQuerier that should NOT be called when range > 24h.
	mockDS := &mockDataSourceQuerier{}

	// Register only the query_datasource tool.
	registry.RegisterBuiltinTools(
		mockDS,
		nil, // AlertRuleOperator
		nil, // IncidentService
		nil, // AuditLogService
		nil, // AlertEventService
		nil, // KnowledgeBaseService
		func() (interface{}, bool) { return nil, false },
	)

	tool, ok := registry.Get("query_datasource")
	require.True(t, ok, "query_datasource tool should be registered")

	// Test: time range > 24h should be rejected.
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"datasource_id": float64(1),
		"query":         "up",
		"time_range":    "48h",
	})
	assert.Error(t, err, "48h time range should be rejected")
	assert.Contains(t, err.Error(), "exceeds", "error should mention the limit being exceeded")

	// Test: time range exactly 24h should NOT be rejected at the range check level
	// (it may fail later due to mock, but not due to range limit).
	_, err = tool.Execute(context.Background(), map[string]interface{}{
		"datasource_id": float64(1),
		"query":         "up",
		"time_range":    "24h",
	})
	// 24h is exactly at the limit — should pass the range check.
	// The mock returns empty data, so it succeeds.
	assert.NoError(t, err, "24h time range should be accepted (at the limit)")

	// Test: time range < 24h should be accepted.
	_, err = tool.Execute(context.Background(), map[string]interface{}{
		"datasource_id": float64(1),
		"query":         "up",
		"time_range":    "1h",
	})
	assert.NoError(t, err, "1h time range should be accepted")

	// Verify the mock was called for valid ranges.
	assert.GreaterOrEqual(t, mockDS.rangeCalls.Load(), int64(2),
		"QueryRange should have been called for valid ranges")
}

// mockDataSourceQuerier is a minimal mock for testing AI tools.
type mockDataSourceQuerier struct {
	rangeCalls atomic.Int64
}

func (m *mockDataSourceQuerier) QueryDatasource(_ context.Context, _ uint, _ string, _ time.Time) (*QueryResponse, error) {
	return &QueryResponse{Series: nil}, nil
}

func (m *mockDataSourceQuerier) QueryRange(_ context.Context, _ uint, _ string, _, _ time.Time, _ string) (*QueryResponse, error) {
	m.rangeCalls.Add(1)
	return &QueryResponse{Series: nil}, nil
}

func (m *mockDataSourceQuerier) QueryLogs(_ context.Context, _ uint, _ LogQueryParams) (*LogQueryResponse, error) {
	return &LogQueryResponse{}, nil
}

func (m *mockDataSourceQuerier) ProxyToDatasource(_ context.Context, _ uint, _ string, _ map[string]string) ([]byte, error) {
	return []byte(`{"status":"success","data":[]}`), nil
}
