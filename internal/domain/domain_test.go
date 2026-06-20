package domain

import "testing"

// Pins Severity wire strings: storage persists them verbatim and QueryFilter
// matches against them, so a rename would silently break querying.
func TestSeverityStringValues(t *testing.T) {
	cases := []struct {
		sev  Severity
		want string
	}{
		{SeverityHigh, "HIGH"},
		{SeverityMed, "MED"},
		{SeverityLow, "LOW"},
	}
	for _, c := range cases {
		if string(c.sev) != c.want {
			t.Errorf("Severity %v = %q, want %q", c.sev, string(c.sev), c.want)
		}
	}
}

// Guards against two severity constants collapsing to the same string.
func TestSeverityValuesDistinct(t *testing.T) {
	seen := map[Severity]struct{}{}
	for _, s := range []Severity{SeverityHigh, SeverityMed, SeverityLow} {
		if _, dup := seen[s]; dup {
			t.Errorf("duplicate Severity value %q", string(s))
		}
		seen[s] = struct{}{}
	}
}

// Compile-time check that the frozen interfaces stay implementable.
type stubScorer struct{}

func (stubScorer) Score(_ string, _ *Profile) ScoredDomain { return ScoredDomain{} }

func (stubScorer) ScoreWithCert(_ string, _ *Profile, _ CertMeta) ScoredDomain {
	return ScoredDomain{}
}

// The real check is at compile time; the body just touches the stub.
func TestInterfacesImplementable(t *testing.T) {
	var s Scorer = stubScorer{}
	got := s.Score("example.com", &Profile{Name: "test"})
	if got.Domain != "" {
		t.Errorf("stub scorer should return zero ScoredDomain, got %+v", got)
	}
}
