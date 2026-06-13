package service

import "testing"

// Test_getBoolField_tristate verifies the tri-state parsing used to guard
// email-based SSO account linking: absent → nil (preserve legacy behavior),
// explicit false → blocks linking, true → allows.
func Test_getBoolField_tristate(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want *bool
	}{
		{"absent", map[string]interface{}{}, "email_verified", nil},
		{"bool true", map[string]interface{}{"email_verified": true}, "email_verified", boolPtr(true)},
		{"bool false", map[string]interface{}{"email_verified": false}, "email_verified", boolPtr(false)},
		{"string true", map[string]interface{}{"email_verified": "true"}, "email_verified", boolPtr(true)},
		{"string false", map[string]interface{}{"email_verified": "false"}, "email_verified", boolPtr(false)},
		{"unparseable string", map[string]interface{}{"email_verified": "yes"}, "email_verified", nil},
		{"empty key", map[string]interface{}{"email_verified": true}, "", nil},
		{"wrong type", map[string]interface{}{"email_verified": 1}, "email_verified", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBoolField(tt.m, tt.key)
			if (got == nil) != (tt.want == nil) {
				t.Fatalf("nil mismatch: got %v want %v", got, tt.want)
			}
			if got != nil && *got != *tt.want {
				t.Fatalf("value mismatch: got %v want %v", *got, *tt.want)
			}
		})
	}
}

func boolPtr(b bool) *bool { return &b }
