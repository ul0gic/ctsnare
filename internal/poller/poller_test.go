package poller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ul0gic/ctsnare/internal/config"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// sentinelStart is the sentinel value used for the firstStart atomic to
// distinguish "not yet recorded" from an actual start index of 0.
const sentinelStart int64 = -1

// newBacktrackServer creates a mock CT log server that returns the given tree size
// and tracks the start index of the first get-entries request. Initialize
// firstStart to sentinelStart before passing it in.
func newBacktrackServer(t *testing.T, treeSize int64, firstStart *atomic.Int64) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/ct/v1/get-sth", func(w http.ResponseWriter, _ *http.Request) {
		sth := SignedTreeHead{TreeSize: treeSize, Timestamp: time.Now().UnixMilli()}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sth) //nolint:errcheck
	})
	mux.HandleFunc("/ct/v1/get-entries", func(w http.ResponseWriter, r *http.Request) {
		// Record the first start parameter we see.
		startParam := r.URL.Query().Get("start")
		if startParam != "" {
			val, err := json.Number(startParam).Int64()
			if err == nil {
				firstStart.CompareAndSwap(sentinelStart, val)
			}
		}
		resp := ctlogEntriesResponse{Entries: []ctlogEntry{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestPoller_Backtrack_StartsAtOffset(t *testing.T) {
	const treeSize = 5000
	const backtrack = 1000

	var firstStart atomic.Int64
	firstStart.Store(sentinelStart)
	server := newBacktrackServer(t, treeSize, &firstStart)

	p := NewPoller(
		server.URL, "test-log",
		&mockScorer{}, &mockStore{}, &domain.Profile{Name: "test"},
		256, 100*time.Millisecond,
		make(chan<- domain.Hit, 10), make(chan<- PollStats, 10),
		nil,
		backtrack,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go p.Run(ctx) //nolint:errcheck

	// Wait for the poller to make at least one get-entries request.
	deadline := time.After(3 * time.Second)
	for firstStart.Load() == sentinelStart {
		select {
		case <-deadline:
			t.Fatal("poller did not make a get-entries request within 3 seconds")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	// With tree_size=5000, backtrack=1000 -> start should be 4000.
	assert.Equal(t, int64(4000), firstStart.Load(),
		"poller should start at tree_size - backtrack")
}

func TestPoller_Backtrack_Zero_StartsAtTip(t *testing.T) {
	// With backtrack=0, the poller should start at tree_size (the tip),
	// meaning no get-entries requests until new entries appear.
	const treeSize = 5000

	var firstStart atomic.Int64
	firstStart.Store(sentinelStart)
	server := newBacktrackServer(t, treeSize, &firstStart)

	p := NewPoller(
		server.URL, "test-log",
		&mockScorer{}, &mockStore{}, &domain.Profile{Name: "test"},
		256, 50*time.Millisecond,
		make(chan<- domain.Hit, 10), make(chan<- PollStats, 10),
		nil,
		0, // no backtrack
	)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	p.Run(ctx) //nolint:errcheck

	// With backtrack=0 and no new entries, the poller should never call get-entries.
	assert.Equal(t, sentinelStart, firstStart.Load(),
		"with backtrack=0, poller should not fetch entries when at the tip")
}

func TestPoller_Backtrack_ExceedsTreeSize_ClampsToZero(t *testing.T) {
	const treeSize = 500
	const backtrack = 10000 // much larger than tree size

	var firstStart atomic.Int64
	firstStart.Store(sentinelStart)
	server := newBacktrackServer(t, treeSize, &firstStart)

	p := NewPoller(
		server.URL, "test-log",
		&mockScorer{}, &mockStore{}, &domain.Profile{Name: "test"},
		256, 100*time.Millisecond,
		make(chan<- domain.Hit, 10), make(chan<- PollStats, 10),
		nil,
		backtrack,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go p.Run(ctx) //nolint:errcheck

	// Wait for request.
	deadline := time.After(3 * time.Second)
	for firstStart.Load() == sentinelStart {
		select {
		case <-deadline:
			t.Fatal("poller did not make a get-entries request within 3 seconds")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	// With tree_size=500, backtrack=10000 -> clamped to 0.
	assert.Equal(t, int64(0), firstStart.Load(),
		"when backtrack > tree_size, poller should clamp to index 0")
}

func TestNewPoller_BacktrackFieldSet(t *testing.T) {
	p := NewPoller(
		"https://example.com", "test",
		&mockScorer{}, &mockStore{}, &domain.Profile{Name: "test"},
		256, time.Second,
		make(chan<- domain.Hit, 1), make(chan<- PollStats, 1),
		nil,
		42,
	)
	assert.Equal(t, int64(42), p.backtrack)
}

func TestNewPoller_BacktrackDefault(t *testing.T) {
	p := NewPoller(
		"https://example.com", "test",
		&mockScorer{}, &mockStore{}, &domain.Profile{Name: "test"},
		256, time.Second,
		make(chan<- domain.Hit, 1), make(chan<- PollStats, 1),
		nil,
		0,
	)
	assert.Equal(t, int64(0), p.backtrack)
}

func TestManager_Backtrack_PassedToPollers(t *testing.T) {
	// Use a mock server that records the first get-entries start index.
	const treeSize = 3000
	const backtrack int64 = 500

	var firstStart atomic.Int64
	firstStart.Store(sentinelStart)
	server := newBacktrackServer(t, treeSize, &firstStart)

	cfg := &config.Config{
		CTLogs: []config.CTLogConfig{
			{URL: server.URL, Name: "backtrack-test"},
		},
		BatchSize:    256,
		PollInterval: 100 * time.Millisecond,
	}

	mgr := NewManager(cfg, &mockScorer{}, &mockStore{}, &domain.Profile{Name: "test"}, backtrack)

	hitChan := make(chan domain.Hit, 10)
	statsChan := make(chan PollStats, 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := mgr.Start(ctx, hitChan, statsChan, nil)
	require.NoError(t, err)

	// Wait for poller to fetch entries.
	deadline := time.After(3 * time.Second)
	for firstStart.Load() == sentinelStart {
		select {
		case <-deadline:
			cancel()
			mgr.Stop()
			t.Fatal("poller did not make a get-entries request within 3 seconds")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	assert.Equal(t, int64(2500), firstStart.Load(),
		"manager should pass backtrack to poller: 3000 - 500 = 2500")

	cancel()
	mgr.Stop()
}
