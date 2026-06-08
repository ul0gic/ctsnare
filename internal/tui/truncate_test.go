package tui

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestTruncate_RuneAware(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		maxLen int
		want   string
	}{
		{"ascii under limit", "apple.com", 20, "apple.com"},
		{"ascii at limit", "apple.com", 9, "apple.com"},
		{"ascii over limit", "verylongdomain.example.com", 10, "verylon..."},
		{"empty maxLen", "anything", 0, ""},
		{"tiny maxLen no ellipsis", "abcdef", 2, "ab"},
		// Cyrillic homograph: every rune is 2 bytes. A byte-slice would split a
		// rune; a rune-slice keeps valid UTF-8.
		{"multibyte under limit", "аррӏе.com", 20, "аррӏе.com"},
		{"multibyte over limit", "аррӏеаррӏеаррӏе.com", 8, "аррӏе..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.in, tt.maxLen)
			assert.Equal(t, tt.want, got)
			assert.True(t, utf8.ValidString(got),
				"truncate must never produce invalid UTF-8, got %q", got)
			assert.LessOrEqual(t, utf8.RuneCountInString(got), max(tt.maxLen, 0),
				"truncated string must not exceed maxLen runes")
		})
	}
}

func TestTruncate_NeverSplitsMultibyteRune(t *testing.T) {
	// A Cyrillic string truncated at every byte-unsafe boundary must stay valid.
	s := "аррӏеаррӏеаррӏеаррӏе"
	for n := 1; n <= utf8.RuneCountInString(s)+2; n++ {
		out := truncate(s, n)
		assert.True(t, utf8.ValidString(out),
			"truncate(%q, %d) produced invalid UTF-8: %q", s, n, out)
	}
}
