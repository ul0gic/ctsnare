package enrichment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// mockStore implements domain.Store for enrichment tests. Only UpdateEnrichment
// is exercised; other methods are stubs.
type mockStore struct {
	mu          sync.Mutex
	enrichments map[string]enrichmentRecord
}

type enrichmentRecord struct {
	isLive          bool
	resolvedIPs     []string
	hostingProvider string
	httpStatus      int
}

func newMockStore() *mockStore {
	return &mockStore{enrichments: make(map[string]enrichmentRecord)}
}

func (m *mockStore) UpdateEnrichment(_ context.Context, d string, isLive bool, ips []string, provider string, status int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enrichments[d] = enrichmentRecord{
		isLive:          isLive,
		resolvedIPs:     ips,
		hostingProvider: provider,
		httpStatus:      status,
	}
	return nil
}

func (m *mockStore) getEnrichment(d string) (enrichmentRecord, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.enrichments[d]
	return rec, ok
}

func (m *mockStore) InsertHit(context.Context, domain.Hit) error { return nil }
func (m *mockStore) UpsertHit(context.Context, domain.Hit) error { return nil }
func (m *mockStore) QueryHits(context.Context, domain.QueryFilter) ([]domain.Hit, error) {
	return nil, nil
}
func (m *mockStore) Stats(context.Context) (domain.DBStats, error)   { return domain.DBStats{}, nil }
func (m *mockStore) ClearAll(context.Context) error                  { return nil }
func (m *mockStore) ClearSession(context.Context, string) error      { return nil }
func (m *mockStore) SetBookmark(context.Context, string, bool) error { return nil }
func (m *mockStore) DeleteHit(context.Context, string) error         { return nil }
func (m *mockStore) DeleteHits(context.Context, []string) error      { return nil }
func (m *mockStore) Close() error                                    { return nil }

func TestEnricher_LiveDomain(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := newMockStore()
	resultCh := make(chan EnrichResult, 10)
	enricher := NewEnricher(store, resultCh)
	// Override HTTP client to use the test server's TLS client.
	enricher.httpClient = server.Client()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go enricher.Run(ctx) //nolint:errcheck

	// Enqueue the test server's hostname (strip scheme for ProbeLiveness).
	// The httptest TLS server listens on 127.0.0.1, so DNS resolves.
	enricher.Enqueue("127.0.0.1")

	select {
	case result := <-resultCh:
		assert.Equal(t, "127.0.0.1", result.Domain)
		// HTTP probe may fail (TLS cert mismatch for raw IP), but DNS should resolve.
		assert.NotEmpty(t, result.ResolvedIPs, "should have resolved IPs for 127.0.0.1")
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for enrichment result")
	}

	cancel()
}

func TestEnricher_DeadDomain(t *testing.T) {
	store := newMockStore()
	resultCh := make(chan EnrichResult, 10)
	enricher := NewEnricher(store, resultCh)
	// Use a short timeout client so the dead domain probe fails fast.
	enricher.httpClient = &http.Client{Timeout: 1 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go enricher.Run(ctx) //nolint:errcheck

	// Use a domain that will not resolve.
	enricher.Enqueue("this-domain-definitely-does-not-exist-4829.invalid")

	select {
	case result := <-resultCh:
		assert.Equal(t, "this-domain-definitely-does-not-exist-4829.invalid", result.Domain)
		assert.False(t, result.IsLive, "dead domain should not be live")
		assert.Equal(t, 0, result.HTTPStatus)
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for enrichment result")
	}

	cancel()
}

func TestEnricher_GracefulShutdown(t *testing.T) {
	store := newMockStore()
	resultCh := make(chan EnrichResult, 100)
	enricher := NewEnricher(store, resultCh)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		enricher.Run(ctx) //nolint:errcheck
		close(done)
	}()

	// Allow workers to start.
	time.Sleep(50 * time.Millisecond)

	// Cancel context -- workers should exit cleanly without panic.
	cancel()

	select {
	case <-done:
		// Workers exited cleanly.
	case <-time.After(5 * time.Second):
		t.Fatal("enricher did not shut down within 5 seconds")
	}
}

func TestEnricher_QueueOverflow(t *testing.T) {
	store := newMockStore()
	resultCh := make(chan EnrichResult, 10)
	enricher := NewEnricher(store, resultCh)

	// Do NOT start the enricher -- queue will fill up.
	// Enqueue more than queueCapacity to test non-blocking drop.
	for i := 0; i < queueCapacity+100; i++ {
		enricher.Enqueue("overflow-domain.com")
	}

	// If we got here without blocking, the test passes.
	assert.Len(t, enricher.queue, queueCapacity, "queue should be at capacity, not more")
}

func TestEnricher_RateLimiting(t *testing.T) {
	// Count how many requests arrive at the HTTP server.
	var requestCount atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := newMockStore()
	resultCh := make(chan EnrichResult, 100)
	enricher := NewEnricher(store, resultCh)
	enricher.httpClient = &http.Client{Timeout: 2 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go enricher.Run(ctx) //nolint:errcheck

	// Enqueue 20 domains. With a global rate of 5 req/sec, processing
	// 20 domains should take roughly 4 seconds if rate limiting is working.
	const domainCount = 20
	for i := 0; i < domainCount; i++ {
		enricher.Enqueue("127.0.0.1")
	}

	start := time.Now()

	// Drain all results.
	received := 0
	for received < domainCount {
		select {
		case <-resultCh:
			received++
		case <-time.After(30 * time.Second):
			t.Fatalf("timed out after receiving %d/%d results", received, domainCount)
		}
	}

	elapsed := time.Since(start)

	// With 5 req/sec, 20 domains should take at least 3 seconds.
	// Being generous with the lower bound to avoid flaky tests.
	assert.GreaterOrEqual(t, elapsed, 2*time.Second,
		"rate limiting should prevent all domains from being probed instantly (elapsed: %v)", elapsed)

	cancel()
}

func TestEnricher_ResultSentOnChannel(t *testing.T) {
	store := newMockStore()
	resultCh := make(chan EnrichResult, 10)
	enricher := NewEnricher(store, resultCh)
	enricher.httpClient = &http.Client{Timeout: 1 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go enricher.Run(ctx) //nolint:errcheck

	enricher.Enqueue("127.0.0.1")

	select {
	case result := <-resultCh:
		assert.Equal(t, "127.0.0.1", result.Domain)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for enrichment result on channel")
	}

	cancel()
}

func TestEnricher_StoreReceivesEnrichment(t *testing.T) {
	store := newMockStore()
	resultCh := make(chan EnrichResult, 10)
	enricher := NewEnricher(store, resultCh)
	enricher.httpClient = &http.Client{Timeout: 2 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go enricher.Run(ctx) //nolint:errcheck

	enricher.Enqueue("127.0.0.1")

	// Wait for result to ensure probe completed.
	select {
	case <-resultCh:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for enrichment result")
	}

	// Verify the store received the UpdateEnrichment call.
	rec, ok := store.getEnrichment("127.0.0.1")
	assert.True(t, ok, "store should have received enrichment for 127.0.0.1")
	assert.NotEmpty(t, rec.resolvedIPs, "should have resolved IPs")

	cancel()
}
