package i18n

import "testing"

func Test_Negotiate(t *testing.T) {
	tests := []struct {
		name     string
		explicit string
		accept   string
		want     string
	}{
		{"explicit zh wins", "zh-CN", "en-US,en;q=0.9", ZhCN},
		{"explicit en wins", "en", "zh-CN", En},
		{"accept zh", "", "zh-CN,zh;q=0.9", ZhCN},
		{"accept en", "", "en-US,en;q=0.9", En},
		{"unknown falls back to en", "", "fr-FR", En},
		{"empty falls back to en", "", "", En},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Negotiate(tt.explicit, tt.accept); got != tt.want {
				t.Fatalf("Negotiate(%q,%q)=%q want %q", tt.explicit, tt.accept, got, tt.want)
			}
		})
	}
}

func Test_LocalizeMessage(t *testing.T) {
	// Known canonical message is translated for zh-CN.
	if got := LocalizeMessage(ZhCN, "invalid credentials"); got != "用户名或密码错误" {
		t.Fatalf("zh translate failed: %q", got)
	}
	// English locale returns the source unchanged.
	if got := LocalizeMessage(En, "invalid credentials"); got != "invalid credentials" {
		t.Fatalf("en should pass through: %q", got)
	}
	// Custom / unknown message passes through unchanged even for zh.
	custom := "snooze time must be in the future"
	if got := LocalizeMessage(ZhCN, custom); got != custom {
		t.Fatalf("unknown message should pass through: %q", got)
	}
	// Empty locale passes through.
	if got := LocalizeMessage("", "internal server error"); got != "internal server error" {
		t.Fatalf("empty locale should pass through: %q", got)
	}

	// New translations from service layer i18n migration.
	newTests := []struct{ en, zh string }{
		{"AI feature is not enabled", "AI 功能未启用，请在系统设置中配置并启用 AI"},
		{"AI is not enabled", "AI 未启用"},
		{"failed to load AI config", "加载 AI 配置失败"},
		{"expression is empty", "表达式为空"},
		{"PromQL syntax error", "PromQL 语法错误"},
		{"event_id is required", "event_id 不能为空"},
		{"task_id is required", "task_id 不能为空"},
		{"invalid cron expression", "无效的 cron 表达式"},
		{"query failed", "查询失败"},
		{"failed to create inspection run", "创建巡检运行记录失败"},
		{"failed to create report run", "创建报告运行记录失败"},
	}
	for _, tt := range newTests {
		if got := LocalizeMessage(ZhCN, tt.en); got != tt.zh {
			t.Errorf("LocalizeMessage(zh, %q) = %q, want %q", tt.en, got, tt.zh)
		}
		// English passthrough
		if got := LocalizeMessage(En, tt.en); got != tt.en {
			t.Errorf("LocalizeMessage(en, %q) = %q, want %q", tt.en, got, tt.en)
		}
	}
}
