package lark

import (
	"context"
	"testing"
)

func TestLarkError_Error(t *testing.T) {
	e := &LarkError{Code: 99991663, Message: "rate limit exceeded"}
	want := "lark api error code=99991663 msg=rate limit exceeded"
	if got := e.Error(); got != want {
		t.Errorf("LarkError.Error() = %q, want %q", got, want)
	}
}

func TestLarkError_IsRetryable(t *testing.T) {
	retryableCodes := []int{99991663, 99991668, 99991672, 10012, 10006}
	nonRetryableCodes := []int{0, 10001, 10002, 10003, 20001}

	for _, code := range retryableCodes {
		e := &LarkError{Code: code}
		if !e.IsRetryable() {
			t.Errorf("LarkError{Code: %d}.IsRetryable() = false, want true", code)
		}
	}
	for _, code := range nonRetryableCodes {
		e := &LarkError{Code: code}
		if e.IsRetryable() {
			t.Errorf("LarkError{Code: %d}.IsRetryable() = true, want false", code)
		}
	}
}

func TestDoWithRetry_SuccessFirstAttempt(t *testing.T) {
	calls := 0
	fn := func() (larkAPIResult, error) {
		calls++
		return larkAPIResult{Code: 0, Msg: "ok"}, nil
	}

	result, err := doWithRetry(context.Background(), fn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code != 0 {
		t.Errorf("result.Code = %d, want 0", result.Code)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestDoWithRetry_RetryableThenSuccess(t *testing.T) {
	calls := 0
	fn := func() (larkAPIResult, error) {
		calls++
		if calls < 3 {
			return larkAPIResult{Code: 99991663, Msg: "rate limit"}, nil
		}
		return larkAPIResult{Code: 0, Msg: "ok"}, nil
	}

	result, err := doWithRetry(context.Background(), fn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code != 0 {
		t.Errorf("result.Code = %d, want 0", result.Code)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestDoWithRetry_NonRetryableError(t *testing.T) {
	calls := 0
	fn := func() (larkAPIResult, error) {
		calls++
		return larkAPIResult{Code: 10001, Msg: "param error"}, nil
	}

	_, err := doWithRetry(context.Background(), fn)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (non-retryable should not retry)", calls)
	}
}

func TestDoWithRetry_NetworkError(t *testing.T) {
	calls := 0
	fn := func() (larkAPIResult, error) {
		calls++
		return larkAPIResult{}, &LarkError{Code: -1, Message: "network error"}
	}

	// Network errors (returned as error, not as code) should not retry
	_, err := doWithRetry(context.Background(), fn)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDoWithRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	fn := func() (larkAPIResult, error) {
		return larkAPIResult{Code: 99991663, Msg: "rate limit"}, nil
	}

	_, err := doWithRetry(ctx, fn)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestDoWithRetry_ExhaustsRetries(t *testing.T) {
	calls := 0
	fn := func() (larkAPIResult, error) {
		calls++
		return larkAPIResult{Code: 99991672, Msg: "server busy"}, nil
	}

	_, err := doWithRetry(context.Background(), fn)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestTokenCache_GetSet(t *testing.T) {
	var tc tokenCache

	// Empty cache should return false
	_, ok := tc.get()
	if ok {
		t.Error("empty cache returned ok=true")
	}

	// Set and get
	tc.set("test-token", 3600)
	tok, ok := tc.get()
	if !ok {
		t.Fatal("cache returned ok=false after set")
	}
	if tok != "test-token" {
		t.Errorf("token = %q, want %q", tok, "test-token")
	}
}

func TestTokenCache_Expiry(t *testing.T) {
	var tc tokenCache
	// Set with very short effective TTL (1s means effective is clamped to 30s minimum)
	tc.set("short-lived", 31)

	// Should still be valid immediately
	_, ok := tc.get()
	if !ok {
		t.Error("cache expired too early")
	}
}

func TestBotClient_TokenCacheIntegration(t *testing.T) {
	var tc tokenCache
	tc.set("valid-token", 3600)
	tok, ok := tc.get()
	if !ok || tok != "valid-token" {
		t.Errorf("token cache failed: got=%q ok=%v", tok, ok)
	}

	// Overwrite with new token
	tc.set("new-token", 7200)
	tok, ok = tc.get()
	if !ok || tok != "new-token" {
		t.Errorf("token cache overwrite failed: got=%q ok=%v", tok, ok)
	}
}
