package service

import (
	"testing"

	"github.com/sreagent/sreagent/internal/model"
)

// Test_mediaLanguage verifies the channel-level language resolver used to pick the
// message-template variant and Lark card labels for group-broadcast channels.
func Test_mediaLanguage(t *testing.T) {
	cases := []struct {
		name   string
		config string
		want   string
	}{
		{"explicit en", `{"webhook_url":"x","language":"en"}`, "en"},
		{"explicit zh-CN", `{"webhook_url":"x","language":"zh-CN"}`, "zh-CN"},
		{"absent → default zh-CN", `{"webhook_url":"x"}`, "zh-CN"},
		{"unknown → default zh-CN", `{"language":"fr"}`, "zh-CN"},
		{"empty config → default", ``, "zh-CN"},
		{"malformed config → default", `not json`, "zh-CN"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mediaLanguage(&model.NotifyMedia{Config: tc.config})
			if got != tc.want {
				t.Fatalf("mediaLanguage(%q) = %q, want %q", tc.config, got, tc.want)
			}
		})
	}
	if got := mediaLanguage(nil); got != "zh-CN" {
		t.Fatalf("mediaLanguage(nil) = %q, want zh-CN", got)
	}
}
