package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ul0gic/ctsnare/internal/domain"
	"github.com/ul0gic/ctsnare/internal/domainutil"
)

// TestQueryHits_DomainFilter_ApexAndSubdomains verifies the QueryFilter.Domain
// predicate matches the apex and all subdomains but rejects sibling and
// suffix-trap domains.
func TestQueryHits_DomainFilter_ApexAndSubdomains(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	domains := []string{
		"openai.com",
		"api.openai.com",
		"a.b.c.openai.com",
		"notopenai.com",
		"openai.com.evil.com",
		"anthropic.com",
	}
	for _, d := range domains {
		require.NoError(t, db.UpsertHit(ctx, testHit(d, 0, domain.SeverityLow)))
	}

	hits, err := db.QueryHits(ctx, domain.QueryFilter{Domain: "openai.com", Limit: 100})
	require.NoError(t, err)

	got := make(map[string]bool, len(hits))
	for _, h := range hits {
		got[h.Domain] = true
	}

	assert.True(t, got["openai.com"], "apex must match")
	assert.True(t, got["api.openai.com"], "subdomain must match")
	assert.True(t, got["a.b.c.openai.com"], "deep subdomain must match")
	assert.False(t, got["notopenai.com"], "sibling must not match")
	assert.False(t, got["openai.com.evil.com"], "suffix trap must not match")
	assert.False(t, got["anthropic.com"], "unrelated must not match")
	assert.Len(t, hits, 3, "exactly apex + two subdomains")
}

// TestQueryHits_DomainFilter_Normalization verifies the filter normalizes its
// input (wildcard, leading/trailing dot, case) the same way the watch path does.
func TestQueryHits_DomainFilter_Normalization(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	require.NoError(t, db.UpsertHit(ctx, testHit("api.openai.com", 0, domain.SeverityLow)))

	for _, target := range []string{"*.openai.com", ".openai.com", "openai.com.", "OpenAI.COM"} {
		hits, err := db.QueryHits(ctx, domain.QueryFilter{Domain: target, Limit: 10})
		require.NoError(t, err)
		assert.Len(t, hits, 1, "target %q should normalize and match api.openai.com", target)
	}
}

// TestQueryHits_DomainFilter_ComposesWithSession verifies the domain predicate
// is combined with other filters via AND.
func TestQueryHits_DomainFilter_ComposesWithSession(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	h1 := testHit("api.openai.com", 0, domain.SeverityLow)
	h1.Session = "run-a"
	h2 := testHit("chat.openai.com", 0, domain.SeverityLow)
	h2.Session = "run-b"
	require.NoError(t, db.UpsertHit(ctx, h1))
	require.NoError(t, db.UpsertHit(ctx, h2))

	hits, err := db.QueryHits(ctx, domain.QueryFilter{Domain: "openai.com", Session: "run-a", Limit: 10})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.Equal(t, "api.openai.com", hits[0].Domain)
}

// TestDomainMatcher_GoSQLParity is the authoritative parity test: for every case
// in domainutil.TrackMatchCases, the SQL predicate must return the same result
// as the Go matcher domainutil.MatchesTrackTarget. The two implementations MUST
// agree; this test fails if they ever drift.
func TestDomainMatcher_GoSQLParity(t *testing.T) {
	for _, c := range domainutil.TrackMatchCases {
		t.Run(c.Name, func(t *testing.T) {
			goResult := domainutil.MatchesTrackTarget(c.Domain, c.Target)
			require.Equal(t, c.Want, goResult, "Go matcher disagrees with expected table value")

			// An empty target means "no filter" at the SQL layer (the predicate
			// is omitted), which is a no-op by design rather than a match
			// decision — so parity with the per-domain Go matcher does not apply.
			if c.Target == "" {
				return
			}

			// An empty domain can't be a stored hit (domain is the primary key
			// and must be non-empty); the SQL side trivially cannot match it.
			if c.Domain == "" {
				assert.False(t, goResult, "empty domain never matches")
				return
			}

			// Each case gets a fresh DB so the single inserted domain is the
			// only candidate the SQL predicate can return.
			db := newTestDB(t)
			ctx := context.Background()
			require.NoError(t, db.UpsertHit(ctx, testHit(c.Domain, 0, domain.SeverityLow)))

			hits, err := db.QueryHits(ctx, domain.QueryFilter{Domain: c.Target, Limit: 10})
			require.NoError(t, err)
			sqlMatched := len(hits) == 1

			assert.Equal(t, goResult, sqlMatched,
				"SQL predicate (%v) must agree with Go matcher (%v) for domain=%q target=%q",
				sqlMatched, goResult, c.Domain, c.Target)
		})
	}
}
