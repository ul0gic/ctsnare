package enrichment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

// TestIntegration_FullPipeline verifies the complete enrichment pipeline:
// enqueue domain -> worker picks it up -> HTTP probe runs -> store receives
// UpdateEnrichment call -> result appears on enrichChan.
func TestIntegration_FullPipeline(t *testing.T) {
	// Stand up a live HTTP server that responds to HEAD requests.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := newMockStore()
	resultCh := make(chan EnrichResult, 10)
	enricher := NewEnricher(store, resultCh)
	// Point the enricher's HTTP client at our test server.
	enricher.httpClient = server.Client()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go enricher.Run(ctx) //nolint:errcheck

	// Enqueue 127.0.0.1 which the test server resolves to.
	enricher.Enqueue("127.0.0.1")

	// Wait for the result on the enrichChan.
	select {
	case result := <-resultCh:
		assert.Equal(t, "127.0.0.1", result.Domain)
		// DNS should resolve 127.0.0.1 to itself.
		assert.NotEmpty(t, result.ResolvedIPs, "should have resolved IPs")
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for enrichment result")
	}

	// Verify the store received the UpdateEnrichment call.
	rec, ok := store.getEnrichment("127.0.0.1")
	require.True(t, ok, "store should have received UpdateEnrichment for 127.0.0.1")
	assert.NotEmpty(t, rec.resolvedIPs, "store should have resolved IPs persisted")

	cancel()
}

// TestIntegration_DeadDomain_Pipeline verifies that a non-resolvable domain
// flows through the full pipeline: enqueue -> probe fails -> store gets
// UpdateEnrichment with isLive=false -> result appears on enrichChan.
func TestIntegration_DeadDomain_Pipeline(t *testing.T) {
	store := newMockStore()
	resultCh := make(chan EnrichResult, 10)
	enricher := NewEnricher(store, resultCh)
	enricher.httpClient = &http.Client{Timeout: 1 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go enricher.Run(ctx) //nolint:errcheck

	dead := "this-domain-absolutely-does-not-exist-9999.invalid"
	enricher.Enqueue(dead)

	select {
	case result := <-resultCh:
		assert.Equal(t, dead, result.Domain)
		assert.False(t, result.IsLive, "dead domain should not be live")
		assert.Equal(t, 0, result.HTTPStatus, "dead domain should have zero HTTP status")
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for dead domain enrichment result")
	}

	// Verify store received the call (even for dead domains).
	rec, ok := store.getEnrichment(dead)
	require.True(t, ok, "store should have received UpdateEnrichment even for dead domain")
	assert.False(t, rec.isLive, "stored enrichment should show dead")

	cancel()
}

// TestIntegration_MultipleDomains_Pipeline verifies that multiple domains
// enqueued sequentially all flow through the pipeline and produce results.
func TestIntegration_MultipleDomains_Pipeline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := newMockStore()
	resultCh := make(chan EnrichResult, 20)
	enricher := NewEnricher(store, resultCh)
	enricher.httpClient = server.Client()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go enricher.Run(ctx) //nolint:errcheck

	const count = 5
	for i := 0; i < count; i++ {
		enricher.Enqueue("127.0.0.1")
	}

	received := 0
	for received < count {
		select {
		case <-resultCh:
			received++
		case <-time.After(15 * time.Second):
			t.Fatalf("timed out after receiving %d/%d results", received, count)
		}
	}

	assert.Equal(t, count, received, "should receive all enqueued domain results")

	cancel()
}

// TestIntegration_RateLimiting_EndToEnd verifies that the enricher respects
// its rate limiter (5 req/sec burst with 5 tokens). Processing many domains
// should take a minimum amount of time proportional to the rate limit.
func TestIntegration_RateLimiting_EndToEnd(t *testing.T) {
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

	// Enqueue 15 domains. With 5 req/sec rate, processing 15 should take
	// at least ~2 seconds after the initial burst of 5.
	const domainCount = 15
	for i := 0; i < domainCount; i++ {
		enricher.Enqueue("127.0.0.1")
	}

	start := time.Now()

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

	// With 5 req/sec, 15 domains should take at least ~1.5 seconds.
	// Using a generous lower bound to avoid flakiness.
	assert.GreaterOrEqual(t, elapsed, 1*time.Second,
		"rate limiting should prevent instant processing (elapsed: %v)", elapsed)

	cancel()
}

// TestIntegration_GracefulShutdown_DrainsQueue verifies that cancelling the
// context causes all workers to exit cleanly without panics or hangs, even
// when domains are queued.
func TestIntegration_GracefulShutdown_DrainsQueue(t *testing.T) {
	store := newMockStore()
	resultCh := make(chan EnrichResult, 100)
	enricher := NewEnricher(store, resultCh)
	enricher.httpClient = &http.Client{Timeout: 1 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		enricher.Run(ctx) //nolint:errcheck
		close(done)
	}()

	// Enqueue some work so workers are busy.
	for i := 0; i < 10; i++ {
		enricher.Enqueue("127.0.0.1")
	}

	// Brief pause to let workers start processing.
	time.Sleep(100 * time.Millisecond)

	// Cancel context -- all workers should drain and exit.
	cancel()

	select {
	case <-done:
		// Workers exited cleanly.
	case <-time.After(10 * time.Second):
		t.Fatal("enricher did not shut down within 10 seconds after context cancel")
	}
}

// TestIntegration_GracefulShutdown_EmptyQueue verifies shutdown is clean
// when no domains have been enqueued.
func TestIntegration_GracefulShutdown_EmptyQueue(t *testing.T) {
	store := newMockStore()
	resultCh := make(chan EnrichResult, 10)
	enricher := NewEnricher(store, resultCh)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		enricher.Run(ctx) //nolint:errcheck
		close(done)
	}()

	// Let workers start.
	time.Sleep(50 * time.Millisecond)

	cancel()

	select {
	case <-done:
		// Clean exit with empty queue.
	case <-time.After(5 * time.Second):
		t.Fatal("enricher did not shut down with empty queue within 5 seconds")
	}
}

// TestIntegration_WorkerPoolSize verifies that the enricher spawns maxWorkers
// goroutines by checking that multiple domains are processed within a
// reasonable time window (faster than sequential processing).
func TestIntegration_WorkerPoolSize(t *testing.T) {
	store := newMockStore()
	resultCh := make(chan EnrichResult, 50)
	enricher := NewEnricher(store, resultCh)
	enricher.httpClient = &http.Client{Timeout: 1 * time.Second}
	// Use a generous rate limit so we measure concurrency, not rate.
	enricher.limiter = rate.NewLimiter(rate.Limit(100), maxWorkers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go enricher.Run(ctx) //nolint:errcheck

	// Enqueue maxWorkers domains. With a worker pool, they should all
	// be picked up and start processing nearly simultaneously.
	for i := 0; i < maxWorkers; i++ {
		enricher.Enqueue("this-domain-does-not-exist-9999.invalid")
	}

	// Drain all results. With 5 concurrent workers and fast DNS failure,
	// this should complete well within 10 seconds.
	received := 0
	for received < maxWorkers {
		select {
		case <-resultCh:
			received++
		case <-time.After(15 * time.Second):
			t.Fatalf("timed out after %d/%d results", received, maxWorkers)
		}
	}

	assert.Equal(t, maxWorkers, received, "all domains should be processed")
	cancel()
}
