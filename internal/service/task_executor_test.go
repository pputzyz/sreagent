package service

import "testing"

// Test_quoteSSHArgs verifies user-supplied SSH args are POSIX single-quoted per token
// so shell metacharacters cannot inject syntax (regression for the bypassable blocklist).
func Test_quoteSSHArgs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"plain tokens", "foo bar", "'foo' 'bar'"},
		{"collapses whitespace", "  foo   bar  ", "'foo' 'bar'"},
		{"semicolon is inert", "a; rm -rf /", `'a;' 'rm' '-rf' '/'`},
		{"command substitution inert", "$(reboot)", `'$(reboot)'`},
		{"backtick inert", "`id`", "'`id`'"},
		{"pipe inert", "a|b", `'a|b'`},
		{"glob and tilde inert (blocklist missed these)", "* ~/x", `'*' '~/x'`},
		{"embedded single quote escaped", "it's", `'it'\''s'`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quoteSSHArgs(tt.in); got != tt.want {
				t.Fatalf("quoteSSHArgs(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
