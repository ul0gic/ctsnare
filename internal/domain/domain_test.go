package domain

import "testing"

// TestSeverityStringValues pins the wire/string values of the Severity
// constants. These strings are load-bearing: storage persists them verbatim and
// QueryFilter.Severity matches against them, so an accidental rename here would
// silently break querying and stored-row classification. This is a sanity guard
// on the frozen domain contract, not a behavior test.
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

// TestSeverityValuesDistinct guards against two severity constants collapsing to
// the same string, which would make classification ambiguous.
func TestSeverityValuesDistinct(t *testing.T) {
	seen := map[Severity]struct{}{}
	for _, s := range []Severity{SeverityHigh, SeverityMed, SeverityLow} {
		if _, dup := seen[s]; dup {
			t.Errorf("duplicate Severity value %q", string(s))
		}
		seen[s] = struct{}{}
	}
}

// stubScorer / stubStore / stubProfileLoader confirm the core interfaces remain
// implementable — a compile-time contract check that fails to build if a method
// signature on the frozen interfaces drifts.
type stubScorer struct{}

func (stubScorer) Score(_ string, _ *Profile) ScoredDomain { return ScoredDomain{} }

// TestInterfacesImplementable asserts the domain interfaces can be satisfied.
// The real value is at compile time; the runtime body just touches the stub so
// it isn't reported as unused.
func TestInterfacesImplementable(t *testing.T) {
	var s Scorer = stubScorer{}
	got := s.Score("example.com", &Profile{Name: "test"})
	if got.Domain != "" {
		t.Errorf("stub scorer should return zero ScoredDomain, got %+v", got)
	}
}
