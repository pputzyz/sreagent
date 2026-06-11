package lark

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCardKitServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *CardKitClient) {
	t.Helper()
	srv := httptest.NewServer(handler)

	botClient := &BotClient{
		httpClient: srv.Client(),
		appID:      "test-app",
		appSecret:  "test-secret",
		tokenCache: NewTokenCache(),
		baseURL:    srv.URL,
	}
	// Pre-populate token cache so we don't hit the token endpoint.
	botClient.tokenCache.Set("test-token", 3600)

	client := NewCardKitClient(botClient, nil)
	return srv, client
}

func Test_CardKitClient_CreateCardEntity(t *testing.T) {
	srv, client := newTestCardKitServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/cardkit/v1/cards")

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "card_json", body["type"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"msg":  "success",
			"data": map[string]string{"card_id": "card-abc123"},
		})
	})
	defer srv.Close()

	cardID, err := client.CreateCardEntity(context.Background(), `{"schema":"2.0","body":{"elements":[]}}`)
	require.NoError(t, err)
	assert.Equal(t, "card-abc123", cardID)
}

func Test_CardKitClient_CreateCardEntity_Error(t *testing.T) {
	srv, client := newTestCardKitServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 10003,
			"msg":  "invalid params",
		})
	})
	defer srv.Close()

	_, err := client.CreateCardEntity(context.Background(), "bad")
	assert.Error(t, err)
}

func Test_CardKitClient_UpdateCardEntity(t *testing.T) {
	srv, client := newTestCardKitServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, float64(5), body["sequence"])
		assert.Equal(t, "idem-123", body["uuid"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"msg":  "success",
		})
	})
	defer srv.Close()

	err := client.UpdateCardEntity(context.Background(), "card-abc", `{"schema":"2.0"}`, 5, "idem-123")
	assert.NoError(t, err)
}

func Test_CardKitClient_UpdateCardEntity_SequenceError(t *testing.T) {
	srv, client := newTestCardKitServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 300317,
			"msg":  "sequence error",
		})
	})
	defer srv.Close()

	err := client.UpdateCardEntity(context.Background(), "card-abc", `{}`, 1, "")
	assert.Error(t, err)
	var seqErr *CardKitSequenceError
	assert.ErrorAs(t, err, &seqErr)
}

func Test_CardKitClient_UpdateCardEntity_ExpiredError(t *testing.T) {
	srv, client := newTestCardKitServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 230031,
			"msg":  "card expired",
		})
	})
	defer srv.Close()

	err := client.UpdateCardEntity(context.Background(), "card-old", `{}`, 1, "")
	assert.Error(t, err)
	var expErr *CardKitExpiredError
	assert.ErrorAs(t, err, &expErr)
}

func Test_CardKitClient_SendCardByID(t *testing.T) {
	srv, client := newTestCardKitServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/im/v1/messages")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"msg":  "success",
			"data": map[string]string{"message_id": "msg-xyz"},
		})
	})
	defer srv.Close()

	msgID, err := client.SendCardByID(context.Background(), "chat-1", "card-abc")
	require.NoError(t, err)
	assert.Equal(t, "msg-xyz", msgID)
}
