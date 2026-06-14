package lark

import "testing"

// Test_cardLabelsFor verifies the group-broadcast webhook card label catalog:
// "en" yields English labels; anything else (empty / "zh-CN" / unknown) falls back
// to Simplified Chinese so existing channels keep their current rendering.
func Test_cardLabelsFor(t *testing.T) {
	en := cardLabelsFor("en")
	if en.FieldStatus != "Status" || en.StatusFiring != "Firing" || en.BtnDetail != "📊 View details" {
		t.Fatalf("en labels wrong: %+v", en)
	}

	for _, lang := range []string{"", "zh-CN", "fr", "zh"} {
		zh := cardLabelsFor(lang)
		if zh.FieldStatus != "状态" || zh.StatusFiring != "告警中" {
			t.Fatalf("lang %q should fall back to zh-CN, got %+v", lang, zh)
		}
	}
}
