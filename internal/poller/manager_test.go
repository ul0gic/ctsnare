package poller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ul0gic/ctsnare/internal/config"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// mockScorer implements domain.Scorer for testing.
type mockScorer struct{}

func (m *mockScorer) Score(domainName string, profile *domain.Profile) domain.ScoredDomain {
	return domain.ScoredDomain{
		Domain:   domainName,
		Score:    0,
		Severity: "",
	}
}

// mockStore implements domain.Store for testing.
type mockStore struct {
	mu   sync.Mutex
	hits []domain.Hit
}

func (m *mockStore) InsertHit(_ context.Context, hit domain.Hit) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hits = append(m.hits, hit)
	return nil
}

func (m *mockStore) UpsertHit(_ context.Context, hit domain.Hit) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hits = append(m.hits, hit)
	return nil
}

func (m *mockStore) QueryHits(_ context.Context, _ domain.QueryFilter) ([]domain.Hit, error) {
	return nil, nil
}

func (m *mockStore) Stats(_ context.Context) (domain.DBStats, error) {
	return domain.DBStats{}, nil
}

func (m *mockStore) ClearAll(_ context.Context) error { return nil }

func (m *mockStore) ClearSession(_ context.Context, _ string) error { return nil }

func (m *mockStore) Close() error { return nil }

// newMockCTLogServer creates a test HTTP server that responds to CT log API endpoints.
// The STH response returns a tree_size of 0, so the poller sleeps on each cycle.
func newMockCTLogServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/ct/v1/get-sth", func(w http.ResponseWriter, _ *http.Request) {
		sth := SignedTreeHead{TreeSize: 0, Timestamp: time.Now().UnixMilli()}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(sth); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/ct/v1/get-entries", func(w http.ResponseWriter, _ *http.Request) {
		resp := ctlogEntriesResponse{Entries: []ctlogEntry{}}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestManager_StartAndStop(t *testing.T) {
	server := newMockCTLogServer(t)

	cfg := &config.Config{
		CTLogs: []config.CTLogConfig{
			{URL: server.URL, Name: "test-log-1"},
		},
		BatchSize:    10,
		PollInterval: 100 * time.Millisecond,
	}

	mgr := NewManager(cfg, &mockScorer{}, &mockStore{}, &domain.Profile{Name: "test"})

	hitChan := make(chan domain.Hit, 10)
	statsChan := make(chan PollStats, 10)

	ctx := context.Background()
	err := mgr.Start(ctx, hitChan, statsChan)
	require.NoError(t, err)

	// Give poller time to make at least one cycle.
	time.Sleep(300 * time.Millisecond)

	// Stop should return without hanging.
	done := make(chan struct{})
	go func() {
		mgr.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success -- stopped cleanly.
	case <-time.After(5 * time.Second):
		t.Fatal("manager.Stop() did not return within 5 seconds")
	}
}

func TestManager_ContextCancellationStopsAllPollers(t *testing.T) {
	server := newMockCTLogServer(t)

	cfg := &config.Config{
		CTLogs: []config.CTLogConfig{
			{URL: server.URL, Name: "log-1"},
			{URL: server.URL, Name: "log-2"},
			{URL: server.URL, Name: "log-3"},
		},
		BatchSize:    10,
		PollInterval: 100 * time.Millisecond,
	}

	mgr := NewManager(cfg, &mockScorer{}, &mockStore{}, &domain.Profile{Name: "test"})

	hitChan := make(chan domain.Hit, 10)
	statsChan := make(chan PollStats, 10)

	ctx, cancel := context.WithCancel(context.Background())
	err := mgr.Start(ctx, hitChan, statsChan)
	require.NoError(t, err)

	// Allow pollers to start.
	time.Sleep(200 * time.Millisecond)

	// Cancel the context, which should propagate to all pollers.
	cancel()

	done := make(chan struct{})
	go func() {
		mgr.Stop()
		close(done)
	}()

	select {
	case <-done:
		// All pollers stopped via context cancellation.
	case <-time.After(5 * time.Second):
		t.Fatal("pollers did not stop after context cancellation within 5 seconds")
	}
}

func TestManager_MultipleLogConfigs(t *testing.T) {
	// Create a server that returns entries so stats get sent.
	var sthMu sync.Mutex
	requestCounts := make(map[string]int)

	mux := http.NewServeMux()
	mux.HandleFunc("/ct/v1/get-sth", func(w http.ResponseWriter, _ *http.Request) {
		sth := SignedTreeHead{TreeSize: 0, Timestamp: time.Now().UnixMilli()}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(sth); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/ct/v1/get-entries", func(w http.ResponseWriter, _ *http.Request) {
		resp := ctlogEntriesResponse{Entries: []ctlogEntry{}}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sthMu.Lock()
		requestCounts[r.URL.Path]++
		sthMu.Unlock()
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	cfg := &config.Config{
		CTLogs: []config.CTLogConfig{
			{URL: server.URL, Name: "log-a"},
			{URL: server.URL, Name: "log-b"},
		},
		BatchSize:    10,
		PollInterval: 50 * time.Millisecond,
	}

	mgr := NewManager(cfg, &mockScorer{}, &mockStore{}, &domain.Profile{Name: "test"})

	hitChan := make(chan domain.Hit, 10)
	statsChan := make(chan PollStats, 100)

	ctx := context.Background()
	err := mgr.Start(ctx, hitChan, statsChan)
	require.NoError(t, err)

	// Wait for pollers to make requests.
	time.Sleep(300 * time.Millisecond)

	mgr.Stop()

	// Verify both pollers made STH requests (each poller hits /ct/v1/get-sth).
	sthMu.Lock()
	sthCount := requestCounts["/ct/v1/get-sth"]
	sthMu.Unlock()

	// With 2 pollers, each making at least the initial STH request, we expect >= 2 requests.
	assert.GreaterOrEqual(t, sthCount, 2, "expected at least 2 STH requests from 2 pollers")
}

func TestManager_EmptyLogConfig(t *testing.T) {
	cfg := &config.Config{
		CTLogs:       []config.CTLogConfig{},
		BatchSize:    10,
		PollInterval: 100 * time.Millisecond,
	}

	mgr := NewManager(cfg, &mockScorer{}, &mockStore{}, &domain.Profile{Name: "test"})

	hitChan := make(chan domain.Hit, 10)
	statsChan := make(chan PollStats, 10)

	ctx := context.Background()
	err := mgr.Start(ctx, hitChan, statsChan)
	require.NoError(t, err)

	// Stop immediately -- nothing should be running.
	done := make(chan struct{})
	go func() {
		mgr.Stop()
		close(done)
	}()

	select {
	case <-done:
		// No pollers, stop returns immediately.
	case <-time.After(2 * time.Second):
		t.Fatal("manager.Stop() hung on empty config")
	}
}

func TestManager_StopBeforeStart(t *testing.T) {
	cfg := &config.Config{
		CTLogs:       []config.CTLogConfig{},
		BatchSize:    10,
		PollInterval: 100 * time.Millisecond,
	}

	mgr := NewManager(cfg, &mockScorer{}, &mockStore{}, &domain.Profile{Name: "test"})

	// Stop without Start should not panic.
	done := make(chan struct{})
	go func() {
		mgr.Stop()
		close(done)
	}()

	select {
	case <-done:
		// No panic, clean return.
	case <-time.After(2 * time.Second):
		t.Fatal("manager.Stop() hung when called before Start()")
	}
}
