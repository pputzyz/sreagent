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
	"strconv"
	"sync"
	"time"

	"github.com/sreagent/sreagent/internal/pkg/safehttp"
)

const (
	// DefaultLarkBaseURL is the default API base for the China (Feishu) region.
	DefaultLarkBaseURL = "https://open.feishu.cn/open-apis"
	// LarkSuiteBaseURL is the API base for the international (Lark) region.
	LarkSuiteBaseURL = "https://open.larksuite.com/open-apis"

	tokenEndpoint     = "/auth/v3/tenant_access_token/internal"
	sendMsgEndpoint   = "/im/v1/messages"
	patchMsgEndpoint  = "/im/v1/messages/%s"
	deleteMsgEndpoint = "/im/v1/messages/%s"
)

// BaseURLForDomain returns the API base URL for the given domain setting.
// "larksuite" → international, anything else → China (Feishu).
func BaseURLForDomain(domain string) string {
	if domain == "larksuite" {
		return LarkSuiteBaseURL
	}
	return DefaultLarkBaseURL
}

// LarkError represents an error returned by the Lark API.
type LarkError struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

func (e *LarkError) Error() string {
	return fmt.Sprintf("lark api error code=%d msg=%s", e.Code, e.Message)
}

// Error-code semantics (see docs/lark-assistant-plan.md §2.3):
//   - retryable: gateway/business rate limits, token expiry (after cache
//     invalidation), transient send conflicts, timeouts
//   - NOT retryable: missing scope (99991672), bot not in chat (230002),
//     bot unavailable to user (230013), user blocked bot (230053) — these
//     fail identically on retry and must surface to the caller for fallback.
const (
	codeGatewayRateLimit = 99991400 // HTTP 429 gateway throttle
	codeTokenInvalid     = 99991663 // tenant_access_token invalid/expired
	codeIMRateLimit      = 230020   // per-chat/per-user 5 QPS business throttle
	codeSendInFlight     = 230049   // message send in flight (transient conflict)
	codeCardExpired      = 230031   // >14 days, cannot update
	codeSeqOutOfOrder    = 300317   // CardKit sequence not increasing
)

// IsRetryable returns true if the Lark error is transient and the call can be retried.
func (e *LarkError) IsRetryable() bool {
	switch e.Code {
	case codeGatewayRateLimit,
		codeIMRateLimit,
		codeSendInFlight,
		codeTokenInvalid, // retried with a fresh token: apiCall invalidates the cache on this code
		99991668,         // app/user token expired
		11232, 11233,     // legacy throttling codes (global / per-chat)
		10012, // frequency limit
		10006: // request timeout
		return true
	}
	return false
}

// larkAPIResult is the common response envelope for Lark API calls.
// HTTPStatus and RetryAfterSec are transport-level metadata filled by apiCall
// (not part of the JSON body).
type larkAPIResult struct {
	Code          int    `json:"code"`
	Msg           string `json:"msg"`
	HTTPStatus    int    `json:"-"`
	RetryAfterSec int    `json:"-"`
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

		// Rate-limited responses carry an x-ogw-ratelimit-reset header telling
		// us how long to wait — honor it (capped) instead of guessing.
		if result.RetryAfterSec > 0 {
			suggested := time.Duration(result.RetryAfterSec) * time.Second
			if suggested > delay {
				delay = suggested
			}
			if delay > 15*time.Second {
				delay = 15 * time.Second
			}
		}

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
		}
	}
	return lastResult, lastErr
}

// TokenCache caches a tenant_access_token along with its expiry.
// Exported so that other packages (e.g. NotifyMediaService) can share the same cache
// instance via dependency injection, avoiding duplicate token fetches.
type TokenCache struct {
	mu      sync.Mutex
	token   string
	expires time.Time
}

// Get returns the cached token if it is still valid.
func (c *TokenCache) Get() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token == "" || time.Now().After(c.expires) {
		return "", false
	}
	return c.token, true
}

// Set stores a token with the given TTL in seconds.
// The cache refreshes 60 seconds before actual expiry, clamped to minimum 30s.
func (c *TokenCache) Set(token string, ttlSeconds int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
	effective := ttlSeconds - 60
	if effective < 30 {
		effective = 30
	}
	c.expires = time.Now().Add(time.Duration(effective) * time.Second)
}

// Invalidate clears the cached token so the next Get() misses and a fresh
// token is fetched. Called when the API reports 99991663 (token invalid).
func (c *TokenCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = ""
	c.expires = time.Time{}
}

// NewTokenCache creates a new empty TokenCache.
func NewTokenCache() *TokenCache {
	return &TokenCache{}
}

// BotClient wraps Lark Bot API calls (auth, send, patch messages).
type BotClient struct {
	httpClient *http.Client
	appID      string
	appSecret  string
	tokenCache *TokenCache
	baseURL    string // configurable: DefaultLarkBaseURL or LarkSuiteBaseURL
}

// NewBotClient creates a new BotClient with SSRF protection.
// baseURL selects the API region; pass empty string for DefaultLarkBaseURL.
func NewBotClient(appID, appSecret, baseURL string) *BotClient {
	if baseURL == "" {
		baseURL = DefaultLarkBaseURL
	}
	return &BotClient{
		httpClient: safehttp.NewSafeClient(10 * time.Second),
		appID:      appID,
		appSecret:  appSecret,
		tokenCache: NewTokenCache(),
		baseURL:    baseURL,
	}
}

// NewBotClientWithCache creates a new BotClient that shares an existing TokenCache.
// This allows multiple BotClient instances (e.g. LarkService and NotifyMediaService)
// to share a single token cache, avoiding redundant token fetches.
func NewBotClientWithCache(appID, appSecret string, cache *TokenCache, baseURL string) *BotClient {
	if baseURL == "" {
		baseURL = DefaultLarkBaseURL
	}
	return &BotClient{
		httpClient: safehttp.NewSafeClient(10 * time.Second),
		appID:      appID,
		appSecret:  appSecret,
		tokenCache: cache,
		baseURL:    baseURL,
	}
}

// getTenantAccessToken returns a valid tenant_access_token, fetching a new one if needed.
func (c *BotClient) getTenantAccessToken(ctx context.Context) (string, error) {
	if tok, ok := c.tokenCache.Get(); ok {
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
			c.baseURL+tokenEndpoint, bytes.NewReader(body))
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

	c.tokenCache.Set(tokenResult.TenantAccessToken, tokenResult.Expire)
	return tokenResult.TenantAccessToken, nil
}

// parseRateLimitReset extracts the suggested wait seconds from the
// x-ogw-ratelimit-reset response header (present on 429 responses).
func parseRateLimitReset(h http.Header) int {
	v := h.Get("x-ogw-ratelimit-reset")
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// apiCall executes one authenticated Lark API request with retry semantics:
//   - the tenant_access_token is fetched (from cache) on EVERY attempt, and the
//     cache is invalidated when the API reports 99991663, so the retry uses a
//     fresh token instead of replaying the dead one
//   - HTTP 429 is mapped to the gateway throttle code and the
//     x-ogw-ratelimit-reset header drives the retry delay
//   - on success (code 0), the response body is unmarshalled into out (if non-nil)
func (c *BotClient) apiCall(ctx context.Context, method, url string, body []byte, out interface{}) error {
	_, err := doWithRetry(ctx, func() (larkAPIResult, error) {
		token, err := c.getTenantAccessToken(ctx)
		if err != nil {
			return larkAPIResult{}, err
		}

		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reader)
		if err != nil {
			return larkAPIResult{}, err
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return larkAPIResult{}, fmt.Errorf("lark api request: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 10<<20))

		var envelope larkAPIResult
		// Tolerate non-JSON bodies (gateway 429 pages); the envelope stays zero.
		_ = json.Unmarshal(respBody, &envelope)
		envelope.HTTPStatus = resp.StatusCode
		envelope.RetryAfterSec = parseRateLimitReset(resp.Header)

		if resp.StatusCode == http.StatusTooManyRequests && envelope.Code == 0 {
			envelope.Code = codeGatewayRateLimit
			envelope.Msg = "gateway rate limited (HTTP 429)"
		}
		if envelope.Code == codeTokenInvalid {
			c.tokenCache.Invalidate()
		}
		if envelope.Code == 0 && out != nil {
			if err := json.Unmarshal(respBody, out); err != nil {
				return envelope, fmt.Errorf("parse lark api response: %w", err)
			}
		}
		return envelope, nil
	})
	return err
}

// SendMessage sends a card message to a Lark group chat via Bot API.
// Returns the message_id which can be used to update the card later.
func (c *BotClient) SendMessage(ctx context.Context, chatID string, card *CardMessage) (string, error) {
	return c.sendCard(ctx, "chat_id", chatID, card)
}

// SendInteractiveJSON sends a raw interactive message content (e.g. a Card 2.0
// JSON string, or a CardKit reference {"type":"card","data":{"card_id":...}}).
// Returns the message_id.
func (c *BotClient) SendInteractiveJSON(ctx context.Context, receiveIDType, receiveID, contentJSON string) (string, error) {
	return c.sendRaw(ctx, receiveIDType, receiveID, "interactive", contentJSON)
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

	url := c.baseURL + sendMsgEndpoint + "?receive_id_type=" + receiveIDType
	if err := c.apiCall(ctx, http.MethodPost, url, body, &sendResult); err != nil {
		return "", err
	}
	return sendResult.Data.MessageID, nil
}

// UpdateMessage patches the content of an existing card message.
func (c *BotClient) UpdateMessage(ctx context.Context, messageID string, card *CardMessage) error {
	cardJSON, err := json.Marshal(card.Card)
	if err != nil {
		return fmt.Errorf("marshal card: %w", err)
	}

	payload := map[string]string{
		"msg_type": "interactive",
		"content":  string(cardJSON),
	}
	body, _ := json.Marshal(payload)
	url := c.baseURL + fmt.Sprintf(patchMsgEndpoint, messageID)
	return c.apiCall(ctx, http.MethodPatch, url, body, nil)
}

// DeleteMessage deletes a message by ID.
func (c *BotClient) DeleteMessage(ctx context.Context, messageID string) error {
	url := c.baseURL + fmt.Sprintf(deleteMsgEndpoint, messageID)
	return c.apiCall(ctx, http.MethodDelete, url, nil, nil)
}
