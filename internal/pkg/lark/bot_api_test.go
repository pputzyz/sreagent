package lark

import (
	"context"
	"testing"
	"time"
)

func TestLarkError_Error(t *testing.T) {
	e := &LarkError{Code: 99991663, Message: "rate limit exceeded"}
	want := "lark api error code=99991663 msg=rate limit exceeded"
	if got := e.Error(); got != want {
		t.Errorf("LarkError.Error() = %q, want %q", got, want)
	}
}

func TestLarkError_IsRetryable(t *testing.T) {
	// Transient conditions: throttling, token expiry (cache is invalidated by
	// apiCall so the retry uses a fresh token), in-flight send conflicts.
	retryableCodes := []int{99991400, 99991663, 99991668, 230020, 230049, 11232, 11233, 10012, 10006}
	// Permanent conditions — retrying fails identically and must surface to
	// the caller for fallback: missing scope (99991672, previously
	// misclassified as "server busy"), bot not in chat, bot unavailable,
	// user blocked bot.
	nonRetryableCodes := []int{0, 99991672, 230002, 230013, 230053, 10001, 10002, 10003, 20001}

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
		return larkAPIResult{Code: 230020, Msg: "rate limited"}, nil
	}

	_, err := doWithRetry(context.Background(), fn)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestDoWithRetry_PermissionDenied_NoRetry(t *testing.T) {
	// 99991672 (missing scope) fails identically on every attempt — it must
	// surface immediately so the caller can fall back to another channel.
	calls := 0
	fn := func() (larkAPIResult, error) {
		calls++
		return larkAPIResult{Code: 99991672, Msg: "permission denied"}, nil
	}

	_, err := doWithRetry(context.Background(), fn)
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (no retry on permanent errors)", calls)
	}
}

func TestDoWithRetry_429_HonorsResetHeader(t *testing.T) {
	calls := 0
	start := time.Now()
	fn := func() (larkAPIResult, error) {
		calls++
		if calls < 2 {
			return larkAPIResult{Code: 99991400, Msg: "throttled", HTTPStatus: 429, RetryAfterSec: 1}, nil
		}
		return larkAPIResult{Code: 0}, nil
	}

	result, err := doWithRetry(context.Background(), fn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code != 0 {
		t.Errorf("result.Code = %d, want 0", result.Code)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond {
		t.Errorf("retry waited %v, want ≥ ~1s (x-ogw-ratelimit-reset header)", elapsed)
	}
}

func TestTokenCache_GetSet(t *testing.T) {
	tc := NewTokenCache()

	// Empty cache should return false
	_, ok := tc.Get()
	if ok {
		t.Error("empty cache returned ok=true")
	}

	// Set and get
	tc.Set("test-token", 3600)
	tok, ok := tc.Get()
	if !ok {
		t.Fatal("cache returned ok=false after set")
	}
	if tok != "test-token" {
		t.Errorf("token = %q, want %q", tok, "test-token")
	}
}

func TestTokenCache_Expiry(t *testing.T) {
	tc := NewTokenCache()
	// Set with very short effective TTL (1s means effective is clamped to 30s minimum)
	tc.Set("short-lived", 31)

	// Should still be valid immediately
	_, ok := tc.Get()
	if !ok {
		t.Error("cache expired too early")
	}
}

func TestBotClient_TokenCacheIntegration(t *testing.T) {
	tc := NewTokenCache()
	tc.Set("valid-token", 3600)
	tok, ok := tc.Get()
	if !ok || tok != "valid-token" {
		t.Errorf("token cache failed: got=%q ok=%v", tok, ok)
	}

	// Overwrite with new token
	tc.Set("new-token", 7200)
	tok, ok = tc.Get()
	if !ok || tok != "new-token" {
		t.Errorf("token cache overwrite failed: got=%q ok=%v", tok, ok)
	}
}
