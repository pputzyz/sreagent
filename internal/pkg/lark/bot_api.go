package lark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/sreagent/sreagent/internal/pkg/safehttp"
)

const (
	larkBaseURL       = "https://open.feishu.cn/open-apis"
	tokenEndpoint     = "/auth/v3/tenant_access_token/internal"
	sendMsgEndpoint   = "/im/v1/messages"
	patchMsgEndpoint  = "/im/v1/messages/%s"
	deleteMsgEndpoint = "/im/v1/messages/%s"
)

// LarkError represents an error returned by the Lark API.
type LarkError struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

func (e *LarkError) Error() string {
	return fmt.Sprintf("lark api error code=%d msg=%s", e.Code, e.Message)
}

// IsRetryable returns true if the Lark error is transient and the call can be retried.
func (e *LarkError) IsRetryable() bool {
	switch e.Code {
	case 99991663, // rate limit exceeded
		99991668, // token expired / needs refresh
		99991672, // server busy
		10012,    // frequency limit
		10006:    // request timeout
		return true
	}
	return false
}

// larkAPIResult is the common response envelope for Lark API calls.
type larkAPIResult struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// doWithRetry executes fn with exponential backoff for retryable Lark errors.
// Max 3 attempts, base delay 500ms, jittered exponential backoff.
func doWithRetry(ctx context.Context, fn func() (larkAPIResult, error)) (larkAPIResult, error) {
	const maxAttempts = 3
	const baseDelay = 500 * time.Millisecond

	var lastResult larkAPIResult
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		result, err := fn()
		if err != nil {
			return result, err // network/parse error, not retryable
		}
		if result.Code == 0 {
			return result, nil
		}

		lastResult = result
		lastErr = &LarkError{Code: result.Code, Message: result.Msg}
		larkErr := lastErr.(*LarkError)

		if !larkErr.IsRetryable() || attempt == maxAttempts-1 {
			return result, lastErr
		}

		// Jittered exponential backoff: base * 2^attempt + random jitter
		delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
		jitter := time.Duration(rand.Int63n(int64(delay / 2)))
		delay += jitter

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
		}
	}
	return lastResult, lastErr
}

// tokenCache caches a tenant_access_token along with its expiry.
type tokenCache struct {
	mu      sync.Mutex
	token   string
	expires time.Time
}

func (c *tokenCache) get() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token == "" || time.Now().After(c.expires) {
		return "", false
	}
	return c.token, true
}

func (c *tokenCache) set(token string, ttlSeconds int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
	// Refresh 60 seconds before actual expiry, clamped to minimum 30s.
	effective := ttlSeconds - 60
	if effective < 30 {
		effective = 30
	}
	c.expires = time.Now().Add(time.Duration(effective) * time.Second)
}

// BotClient wraps Lark Bot API calls (auth, send, patch messages).
type BotClient struct {
	httpClient *http.Client
	appID      string
	appSecret  string
	tokenCache tokenCache
}

// NewBotClient creates a new BotClient with SSRF protection.
func NewBotClient(appID, appSecret string) *BotClient {
	return &BotClient{
		httpClient: safehttp.NewSafeClient(10 * time.Second),
		appID:      appID,
		appSecret:  appSecret,
	}
}

// getTenantAccessToken returns a valid tenant_access_token, fetching a new one if needed.
func (c *BotClient) getTenantAccessToken(ctx context.Context) (string, error) {
	if tok, ok := c.tokenCache.get(); ok {
		return tok, nil
	}

	body, _ := json.Marshal(map[string]string{
		"app_id":     c.appID,
		"app_secret": c.appSecret,
	})

	var tokenResult struct {
		larkAPIResult
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}

	_, err := doWithRetry(ctx, func() (larkAPIResult, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			larkBaseURL+tokenEndpoint, bytes.NewReader(body))
		if err != nil {
			return larkAPIResult{}, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return larkAPIResult{}, fmt.Errorf("get tenant_access_token: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		respBody, _ := io.ReadAll(resp.Body)

		if err := json.Unmarshal(respBody, &tokenResult); err != nil {
			return larkAPIResult{}, fmt.Errorf("parse token response: %w", err)
		}
		return tokenResult.larkAPIResult, nil
	})
	if err != nil {
		return "", err
	}

	c.tokenCache.set(tokenResult.TenantAccessToken, tokenResult.Expire)
	return tokenResult.TenantAccessToken, nil
}

// SendMessage sends a card message to a Lark group chat via Bot API.
// Returns the message_id which can be used to update the card later.
func (c *BotClient) SendMessage(ctx context.Context, chatID string, card *CardMessage) (string, error) {
	return c.sendCard(ctx, "chat_id", chatID, card)
}

// SendDirectMessage sends a card message directly to a user via Bot API.
// receiveIDType should be one of: "user_id", "open_id", "union_id", "email".
// Returns the message_id.
func (c *BotClient) SendDirectMessage(ctx context.Context, receiveIDType, receiveID string, card *CardMessage) (string, error) {
	switch receiveIDType {
	case "user_id", "open_id", "union_id", "email":
	default:
		return "", fmt.Errorf("unsupported receive_id_type for DM: %s", receiveIDType)
	}
	return c.sendCard(ctx, receiveIDType, receiveID, card)
}

// SendText sends a plain-text message via the Bot API (used for command replies).
// receiveIDType is typically "chat_id" for group replies or "open_id"/"user_id" for DMs.
func (c *BotClient) SendText(ctx context.Context, receiveIDType, receiveID, text string) (string, error) {
	textJSON, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return "", fmt.Errorf("marshal text: %w", err)
	}
	return c.sendRaw(ctx, receiveIDType, receiveID, "text", string(textJSON))
}

// sendCard is the shared implementation backing SendMessage / SendDirectMessage.
func (c *BotClient) sendCard(ctx context.Context, receiveIDType, receiveID string, card *CardMessage) (string, error) {
	cardJSON, err := json.Marshal(card.Card)
	if err != nil {
		return "", fmt.Errorf("marshal card: %w", err)
	}
	return c.sendRaw(ctx, receiveIDType, receiveID, "interactive", string(cardJSON))
}

// sendRaw is the underlying message send helper.
func (c *BotClient) sendRaw(ctx context.Context, receiveIDType, receiveID, msgType, content string) (string, error) {
	token, err := c.getTenantAccessToken(ctx)
	if err != nil {
		return "", err
	}

	payload := map[string]string{
		"receive_id": receiveID,
		"msg_type":   msgType,
		"content":    content,
	}
	body, _ := json.Marshal(payload)

	var sendResult struct {
		larkAPIResult
		Data struct {
			MessageID string `json:"message_id"`
		} `json:"data"`
	}

	_, err = doWithRetry(ctx, func() (larkAPIResult, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			larkBaseURL+sendMsgEndpoint+"?receive_id_type="+receiveIDType,
			bytes.NewReader(body))
		if err != nil {
			return larkAPIResult{}, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return larkAPIResult{}, fmt.Errorf("send bot message: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		respBody, _ := io.ReadAll(resp.Body)

		if err := json.Unmarshal(respBody, &sendResult); err != nil {
			return larkAPIResult{}, fmt.Errorf("parse send response: %w", err)
		}
		return sendResult.larkAPIResult, nil
	})
	if err != nil {
		return "", err
	}
	return sendResult.Data.MessageID, nil
}

// UpdateMessage patches the content of an existing card message.
func (c *BotClient) UpdateMessage(ctx context.Context, messageID string, card *CardMessage) error {
	token, err := c.getTenantAccessToken(ctx)
	if err != nil {
		return err
	}

	cardJSON, err := json.Marshal(card.Card)
	if err != nil {
		return fmt.Errorf("marshal card: %w", err)
	}

	payload := map[string]string{
		"msg_type": "interactive",
		"content":  string(cardJSON),
	}
	body, _ := json.Marshal(payload)
	url := larkBaseURL + fmt.Sprintf(patchMsgEndpoint, messageID)

	_, err = doWithRetry(ctx, func() (larkAPIResult, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
		if err != nil {
			return larkAPIResult{}, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return larkAPIResult{}, fmt.Errorf("update bot message: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		respBody, _ := io.ReadAll(resp.Body)

		var result larkAPIResult
		if err := json.Unmarshal(respBody, &result); err != nil {
			return larkAPIResult{}, fmt.Errorf("parse update response: %w", err)
		}
		return result, nil
	})
	return err
}

// DeleteMessage deletes a message by ID.
func (c *BotClient) DeleteMessage(ctx context.Context, messageID string) error {
	token, err := c.getTenantAccessToken(ctx)
	if err != nil {
		return err
	}

	url := larkBaseURL + fmt.Sprintf(deleteMsgEndpoint, messageID)

	_, err = doWithRetry(ctx, func() (larkAPIResult, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
		if err != nil {
			return larkAPIResult{}, err
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return larkAPIResult{}, fmt.Errorf("delete bot message: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		respBody, _ := io.ReadAll(resp.Body)

		var result larkAPIResult
		if err := json.Unmarshal(respBody, &result); err != nil {
			return larkAPIResult{}, fmt.Errorf("parse delete response: %w", err)
		}
		return result, nil
	})
	return err
}
