package lark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CardKit API endpoints.
const (
	cardKitCreateEndpoint = "/cardkit/v1/cards"
	cardKitUpdateEndpoint = "/cardkit/v1/cards/%s"
)

// CardKitClient wraps CardKit API calls for card entity management.
type CardKitClient struct {
	botClient   *BotClient
	rateLimiter *RateLimiter
}

// NewCardKitClient creates a new CardKitClient backed by the given BotClient.
func NewCardKitClient(botClient *BotClient, rateLimiter *RateLimiter) *CardKitClient {
	return &CardKitClient{
		botClient:   botClient,
		rateLimiter: rateLimiter,
	}
}

// CardKitResponse is the common response envelope for CardKit API calls.
type CardKitResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		CardID string `json:"card_id"`
	} `json:"data"`
}

// CreateCardEntity creates a CardKit card entity from a Card 2.0 JSON string.
// Returns the card_id used for sending and updating.
func (c *CardKitClient) CreateCardEntity(ctx context.Context, cardJSON string) (string, error) {
	token, err := c.botClient.getTenantAccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("cardkit create: auth: %w", err)
	}

	payload := map[string]string{
		"type": "card_json",
		"data": cardJSON,
	}
	body, _ := json.Marshal(payload)

	var result CardKitResponse
	_, err = doWithRetry(ctx, func() (larkAPIResult, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			c.botClient.baseURL+cardKitCreateEndpoint, bytes.NewReader(body))
		if err != nil {
			return larkAPIResult{}, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := c.botClient.httpClient.Do(req)
		if err != nil {
			return larkAPIResult{}, fmt.Errorf("cardkit create request: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		respBody, _ := io.ReadAll(resp.Body)

		if err := json.Unmarshal(respBody, &result); err != nil {
			return larkAPIResult{}, fmt.Errorf("parse cardkit create response: %w", err)
		}
		return larkAPIResult{Code: result.Code, Msg: result.Msg}, nil
	})
	if err != nil {
		return "", err
	}
	if result.Data.CardID == "" {
		return "", fmt.Errorf("cardkit create: empty card_id in response")
	}
	return result.Data.CardID, nil
}

// UpdateCardEntity updates an existing card entity's content.
// sequence must be strictly increasing per card_id; uuid is an idempotency key.
func (c *CardKitClient) UpdateCardEntity(ctx context.Context, cardID, cardJSON string, sequence int64, uuid string) error {
	token, err := c.botClient.getTenantAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("cardkit update: auth: %w", err)
	}

	payload := map[string]interface{}{
		"type":     "card_json",
		"data":     cardJSON,
		"sequence": sequence,
	}
	if uuid != "" {
		payload["uuid"] = uuid
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf(c.botClient.baseURL+cardKitUpdateEndpoint, cardID)

	var result larkAPIResult
	_, err = doWithRetry(ctx, func() (larkAPIResult, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
		if err != nil {
			return larkAPIResult{}, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := c.botClient.httpClient.Do(req)
		if err != nil {
			return larkAPIResult{}, fmt.Errorf("cardkit update request: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		respBody, _ := io.ReadAll(resp.Body)

		if err := json.Unmarshal(respBody, &result); err != nil {
			return larkAPIResult{}, fmt.Errorf("parse cardkit update response: %w", err)
		}
		return result, nil
	})
	if err != nil {
		// Wrap specific error codes.
		if larkErr, ok := err.(*LarkError); ok {
			switch larkErr.Code {
			case 300317:
				return &CardKitSequenceError{CardID: cardID, Err: larkErr}
			case 230031:
				return &CardKitExpiredError{CardID: cardID, Err: larkErr}
			}
		}
		return err
	}
	return nil
}

// SendCardByID sends a card entity to a chat using the card_id.
// Returns the message_id.
func (c *CardKitClient) SendCardByID(ctx context.Context, chatID, cardID string) (string, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.WaitChat(ctx, chatID); err != nil {
			return "", fmt.Errorf("rate limit wait: %w", err)
		}
	}

	// CardKit send uses msg_type="interactive" with content={"type":"card","data":{"card_id":"..."}}
	content := fmt.Sprintf(`{"type":"card","data":{"card_id":"%s"}}`, cardID)
	return c.botClient.sendRaw(ctx, "chat_id", chatID, "interactive", content)
}

// --- Error types ---

// CardKitSequenceError indicates a sequence ordering issue (error 300317).
// The caller should re-sync the sequence and retry.
type CardKitSequenceError struct {
	CardID string
	Err    error
}

func (e *CardKitSequenceError) Error() string {
	return fmt.Sprintf("cardkit sequence error for card %s: %v", e.CardID, e.Err)
}

// CardKitExpiredError indicates the card entity has expired (error 230031, >14 days).
// The caller should create a new card entity.
type CardKitExpiredError struct {
	CardID string
	Err    error
}

func (e *CardKitExpiredError) Error() string {
	return fmt.Sprintf("cardkit expired: card %s is older than 14 days: %v", e.CardID, e.Err)
}
