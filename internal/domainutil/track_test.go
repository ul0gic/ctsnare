package domainutil

import (
	"reflect"
	"testing"
)

func TestNormalizeTrackTarget(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "openai.com", "openai.com"},
		{"uppercase", "OpenAI.COM", "openai.com"},
		{"wildcard prefix", "*.openai.com", "openai.com"},
		{"bare leading dot", ".openai.com", "openai.com"},
		{"trailing dot fqdn", "openai.com.", "openai.com"},
		{"wildcard and trailing dot", "*.openai.com.", "openai.com"},
		{"surrounding whitespace", "  openai.com  ", "openai.com"},
		{"empty", "", ""},
		{"only wildcard", "*.", ""},
		{"only dot", ".", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NormalizeTrackTarget(c.input); got != c.want {
				t.Fatalf("NormalizeTrackTarget(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

func TestNormalizeTrackTargets(t *testing.T) {
	cases := []struct {
		name  string
		input []string
		want  []string
	}{
		{"nil", nil, nil},
		{"empty slice", []string{}, nil},
		{"all empty normalize", []string{"", "*.", "."}, nil},
		{"mixed", []string{"*.OpenAI.com", "", "anthropic.com."}, []string{"openai.com", "anthropic.com"}},
		{"order preserved", []string{"b.com", "a.com"}, []string{"b.com", "a.com"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NormalizeTrackTargets(c.input); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("NormalizeTrackTargets(%v) = %v, want %v", c.input, got, c.want)
			}
		})
	}
}

func TestMatchesTrackTarget(t *testing.T) {
	for _, c := range TrackMatchCases {
		t.Run(c.Name, func(t *testing.T) {
			if got := MatchesTrackTarget(c.Domain, c.Target); got != c.Want {
				t.Fatalf("MatchesTrackTarget(%q, %q) = %v, want %v", c.Domain, c.Target, got, c.Want)
			}
		})
	}
}

func TestMatchesAnyTrackTarget(t *testing.T) {
	targets := []string{"openai.com", "anthropic.com"}
	cases := []struct {
		domain string
		want   bool
	}{
		{"api.openai.com", true},
		{"anthropic.com", true},
		{"chat.anthropic.com", true},
		{"notopenai.com", false},
		{"openai.com.evil.com", false},
		{"example.com", false},
	}
	for _, c := range cases {
		t.Run(c.domain, func(t *testing.T) {
			if got := MatchesAnyTrackTarget(c.domain, targets); got != c.want {
				t.Fatalf("MatchesAnyTrackTarget(%q) = %v, want %v", c.domain, got, c.want)
			}
		})
	}
	if MatchesAnyTrackTarget("openai.com", nil) {
		t.Fatal("empty target list must never match")
	}
}
