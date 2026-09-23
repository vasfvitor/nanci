package httpclient

import "testing"

func TestTruncateForLog(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		limit int
		want  string
	}{
		{name: "under limit", body: "abc", limit: 10, want: "abc"},
		{name: "at limit", body: "abcde", limit: 5, want: "abcde"},
		{name: "over limit", body: "abcdefgh", limit: 5, want: "abcde... (truncated)"},
		// "ação" is a(1) ç(2) ã(2) o(1) bytes; a cut inside ç or ã backs up.
		{name: "does not split rune", body: "ação", limit: 2, want: "a... (truncated)"},
		{name: "cut lands on rune start", body: "ação", limit: 3, want: "aç... (truncated)"},
		{name: "cut inside second rune", body: "ação", limit: 4, want: "aç... (truncated)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TruncateForLog([]byte(tt.body), tt.limit); got != tt.want {
				t.Errorf("TruncateForLog(%q, %d) = %q, want %q", tt.body, tt.limit, got, tt.want)
			}
		})
	}
}

func TestMaskIdentifier(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "abc", want: "***"},
		{in: "abcd", want: "****"},
		{in: "abcde", want: "ab*de"},
		{in: "12345678000195", want: "12**********95"},
	}
	for _, tt := range tests {
		if got := MaskIdentifier(tt.in); got != tt.want {
			t.Errorf("MaskIdentifier(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
