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
}
