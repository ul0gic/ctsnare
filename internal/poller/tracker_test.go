package poller

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// scoreStub returns a fixed score/severity for every domain, letting tracker
// tests assert that storage decisions ignore the score in tracker mode.
type scoreStub struct {
	score    int
	severity domain.Severity
	keywords []string
}

func (s scoreStub) Score(domainName string, _ *domain.Profile) domain.ScoredDomain {
	return domain.ScoredDomain{
		Domain:          domainName,
		Score:           s.score,
		Severity:        s.severity,
		MatchedKeywords: s.keywords,
	}
}

// fakeCert builds a minimal certificate with a known issuer for hit assembly.
func fakeCert() *x509.Certificate {
	return &x509.Certificate{
		NotBefore: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Issuer: pkix.Name{
			Organization: []string{"Let's Encrypt"},
			CommonName:   "R3",
		},
	}
}

// newTrackerPoller wires a poller in tracker mode with the given scorer, a
// recording store, and a non-blocking hit channel.
func newTrackerPoller(scorer domain.Scorer, store *mockStore, targets []string, session string) (*Poller, chan domain.Hit) {
	hitChan := make(chan domain.Hit, 16)
	p := NewPoller(
		"https://example.com", "test-log",
		scorer, store, &domain.Profile{Name: "ignored"},
		256, time.Second,
		hitChan, make(chan<- PollStats, 1),
		nil,
		0, 0,
		session, targets,
	)
	return p, hitChan
}

// TestTrackerMode_StoresScoreZeroMatch verifies a tracked domain that scores 0
// IS stored unconditionally (bypassing the score==0 short-circuit and minScore).
func TestTrackerMode_StoresScoreZeroMatch(t *testing.T) {
	store := &mockStore{}
	p, hitChan := newTrackerPoller(scoreStub{score: 0}, store, []string{"openai.com"}, "run-1")

	stored := p.processDomain(context.Background(), "api.openai.com", []string{"api.openai.com"}, fakeCert())
	require.True(t, stored, "a tracked match must report stored=true")

	store.mu.Lock()
	defer store.mu.Unlock()
	require.Len(t, store.hits, 1, "score-0 tracked match must be stored")
	h := store.hits[0]
	assert.Equal(t, "api.openai.com", h.Domain)
	assert.Equal(t, "domain-track", h.Profile, "tracker hits carry the domain-track marker")
	assert.Equal(t, "run-1", h.Session, "session flows from --session")
	assert.Equal(t, "Let's Encrypt", h.Issuer)

	// The hit is also surfaced on the live feed.
	select {
	case fed := <-hitChan:
		assert.Equal(t, "api.openai.com", fed.Domain)
	default:
		t.Fatal("tracked hit should be sent to the live feed")
	}
}

// TestTrackerMode_DoesNotStoreNonMatch verifies a high-scoring domain that does
// NOT match any tracked target is NOT stored in tracker mode (fast path).
func TestTrackerMode_DoesNotStoreNonMatch(t *testing.T) {
	store := &mockStore{}
	// High score that would normally be stored in keyword mode.
	p, hitChan := newTrackerPoller(scoreStub{score: 99, severity: domain.SeverityHigh}, store, []string{"openai.com"}, "")

	stored := p.processDomain(context.Background(), "casino-bitcoin.xyz", []string{"casino-bitcoin.xyz"}, fakeCert())
	assert.False(t, stored, "non-matching domain must not be stored in tracker mode")

	store.mu.Lock()
	defer store.mu.Unlock()
	assert.Empty(t, store.hits, "non-matching high-score domain must not be stored in tracker mode")

	select {
	case <-hitChan:
		t.Fatal("non-matching domain must not be fed to the live feed in tracker mode")
	default:
	}
}

// TestTrackerMode_SuffixTrapNotStored verifies the suffix-trap domain
// openai.com.evil.com does not match the openai.com target.
func TestTrackerMode_SuffixTrapNotStored(t *testing.T) {
	store := &mockStore{}
	p, _ := newTrackerPoller(scoreStub{score: 0}, store, []string{"openai.com"}, "")

	stored := p.processDomain(context.Background(), "openai.com.evil.com", []string{"openai.com.evil.com"}, fakeCert())
	assert.False(t, stored)

	store.mu.Lock()
	defer store.mu.Unlock()
	assert.Empty(t, store.hits, "suffix-trap domain must not match the tracked apex")
}

// TestTrackerMode_ApexMatch verifies the apex itself matches.
func TestTrackerMode_ApexMatch(t *testing.T) {
	store := &mockStore{}
	p, _ := newTrackerPoller(scoreStub{score: 0}, store, []string{"openai.com"}, "")

	stored := p.processDomain(context.Background(), "openai.com", []string{"openai.com"}, fakeCert())
	require.True(t, stored)

	store.mu.Lock()
	defer store.mu.Unlock()
	require.Len(t, store.hits, 1)
	assert.Equal(t, "openai.com", store.hits[0].Domain)
}

// TestKeywordMode_Unaffected verifies that with no tracked targets, the keyword
// path still applies its score==0 short-circuit (the domain is not stored).
func TestKeywordMode_Unaffected(t *testing.T) {
	store := &mockStore{}
	p, _ := newTrackerPoller(scoreStub{score: 0}, store, nil, "")

	stored := p.processDomain(context.Background(), "openai.com", []string{"openai.com"}, fakeCert())
	assert.False(t, stored, "keyword mode drops score-0 domains")

	store.mu.Lock()
	defer store.mu.Unlock()
	assert.Empty(t, store.hits, "score-0 domain not stored in keyword mode")
}

func TestNewPoller_TrackDomainsAndSessionSet(t *testing.T) {
	p, _ := newTrackerPoller(scoreStub{}, &mockStore{}, []string{"openai.com"}, "sess")
	assert.Equal(t, []string{"openai.com"}, p.trackDomains)
	assert.Equal(t, "sess", p.session)
}
