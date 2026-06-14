package service

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

// Test_resolveOutputLang_defaults verifies the per-recipient AI language resolver
// fails safe to the platform default (zh-CN) when the preference service is absent
// or the user is anonymous, and never panics. The DB-backed "en" path is covered by
// integration tests (requires SREAGENT_TEST_DSN).
func Test_resolveOutputLang_defaults(t *testing.T) {
	s := NewAgentService(nil, nil, nil, zap.NewNop())

	// No preference service injected → default zh-CN.
	if got := s.resolveOutputLang(context.Background(), 42); got != "zh-CN" {
		t.Fatalf("nil prefSvc should default to zh-CN, got %q", got)
	}
	// Anonymous user (id 0) → default zh-CN.
	if got := s.resolveOutputLang(context.Background(), 0); got != "zh-CN" {
		t.Fatalf("userID 0 should default to zh-CN, got %q", got)
	}
}
