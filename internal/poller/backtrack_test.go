package poller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// TestBacktrack_TreeSize10000_Backtrack5000 verifies that with tree_size=10000
// and backtrack=5000, the poller starts fetching at index 5000.
func TestBacktrack_TreeSize10000_Backtrack5000(t *testing.T) {
	const treeSize = 10000
	const backtrack = 5000

	var firstStart atomic.Int64
	firstStart.Store(sentinelStart)
	server := newBacktrackServer(t, treeSize, &firstStart)

	p := NewPoller(
		server.URL, "backtrack-test",
		&mockScorer{}, &mockStore{}, &domain.Profile{Name: "test"},
		256, 100*time.Millisecond,
		make(chan<- domain.Hit, 10), make(chan<- PollStats, 10),
		nil,
		backtrack,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go p.Run(ctx) //nolint:errcheck

	deadline := time.After(5 * time.Second)
	for firstStart.Load() == sentinelStart {
		select {
		case <-deadline:
			t.Fatal("poller did not make a get-entries request within 5 seconds")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	assert.Equal(t, int64(5000), firstStart.Load(),
		"with tree_size=10000 and backtrack=5000, poller should start at 5000")
}

// TestBacktrack_TreeSize10000_BacktrackZero verifies that with backtrack=0,
// the poller starts at the tip (tree_size) and does not fetch entries.
func TestBacktrack_TreeSize10000_BacktrackZero(t *testing.T) {
	const treeSize = 10000

	var firstStart atomic.Int64
	firstStart.Store(sentinelStart)
	server := newBacktrackServer(t, treeSize, &firstStart)

	p := NewPoller(
		server.URL, "backtrack-zero-test",
		&mockScorer{}, &mockStore{}, &domain.Profile{Name: "test"},
		256, 50*time.Millisecond,
		make(chan<- domain.Hit, 10), make(chan<- PollStats, 10),
		nil,
		0,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	p.Run(ctx) //nolint:errcheck

	// With backtrack=0 at the tip and no new entries, no get-entries should fire.
	assert.Equal(t, sentinelStart, firstStart.Load(),
		"with backtrack=0, poller should stay at tip and not fetch entries")
}

// TestBacktrack_TreeSize10000_Backtrack20000_ClampsToZero verifies that when
// backtrack exceeds tree_size, the starting index clamps to 0.
func TestBacktrack_TreeSize10000_Backtrack20000_ClampsToZero(t *testing.T) {
	const treeSize = 10000
	const backtrack = 20000

	var firstStart atomic.Int64
	firstStart.Store(sentinelStart)
	server := newBacktrackServer(t, treeSize, &firstStart)

	p := NewPoller(
		server.URL, "backtrack-clamp-test",
		&mockScorer{}, &mockStore{}, &domain.Profile{Name: "test"},
		256, 100*time.Millisecond,
		make(chan<- domain.Hit, 10), make(chan<- PollStats, 10),
		nil,
		backtrack,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go p.Run(ctx) //nolint:errcheck

	deadline := time.After(5 * time.Second)
	for firstStart.Load() == sentinelStart {
		select {
		case <-deadline:
			t.Fatal("poller did not make a get-entries request within 5 seconds")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	assert.Equal(t, int64(0), firstStart.Load(),
		"when backtrack > tree_size, poller should clamp to index 0")
}

// TestBacktrack_ChangingTreeSize verifies that a poller handles a growing
// CT log correctly. Initial STH returns tree_size=10000, backtrack=5000
// starts at 5000. After fetching entries 5000..9999, the next STH returns
// tree_size=12000. The poller should then process entries 10000..11999.
func TestBacktrack_ChangingTreeSize(t *testing.T) {
	var mu sync.Mutex
	treeSize := int64(10000)
	var requestedRanges []struct{ Start, End int64 }

	mux := http.NewServeMux()
	mux.HandleFunc("/ct/v1/get-sth", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		ts := treeSize
		mu.Unlock()
		sth := SignedTreeHead{TreeSize: ts, Timestamp: time.Now().UnixMilli()}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sth) //nolint:errcheck
	})
	mux.HandleFunc("/ct/v1/get-entries", func(w http.ResponseWriter, r *http.Request) {
		startParam := r.URL.Query().Get("start")
		endParam := r.URL.Query().Get("end")

		var start, end int64
		if v, err := json.Number(startParam).Int64(); err == nil {
			start = v
		}
		if v, err := json.Number(endParam).Int64(); err == nil {
			end = v
		}

		mu.Lock()
		requestedRanges = append(requestedRanges, struct{ Start, End int64 }{start, end})

		// After the first batch starting at 5000 is requested, grow the tree.
		if start >= 5000 {
			treeSize = 12000
		}
		mu.Unlock()

		// Return empty entries so the poller advances its index.
		resp := ctlogEntriesResponse{Entries: []ctlogEntry{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	p := NewPoller(
		server.URL, "growing-log",
		&mockScorer{}, &mockStore{}, &domain.Profile{Name: "test"},
		5000, 50*time.Millisecond,
		make(chan<- domain.Hit, 10), make(chan<- PollStats, 100),
		nil,
		5000, // backtrack 5000 from initial tree_size 10000 -> start at 5000
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go p.Run(ctx) //nolint:errcheck

	// Wait until the poller has processed entries beyond 10000
	// (i.e., it noticed the tree grew to 12000).
	deadline := time.After(6 * time.Second)
	for {
		mu.Lock()
		hasSecondRange := false
		for _, rng := range requestedRanges {
			if rng.Start >= 10000 {
				hasSecondRange = true
				break
			}
		}
		mu.Unlock()

		if hasSecondRange {
			break
		}

		select {
		case <-deadline:
			cancel()
			mu.Lock()
			t.Fatalf("poller did not process entries beyond 10000; ranges seen: %v", requestedRanges)
			mu.Unlock()
			return
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	cancel()

	mu.Lock()
	defer mu.Unlock()

	// Verify: first request should start at 5000, later request(s) should start >= 10000.
	assert.NotEmpty(t, requestedRanges, "should have requested entries")
	assert.Equal(t, int64(5000), requestedRanges[0].Start,
		"first request should start at tree_size - backtrack = 5000")

	foundBeyond := false
	for _, rng := range requestedRanges {
		if rng.Start >= 10000 {
			foundBeyond = true
			break
		}
	}
	assert.True(t, foundBeyond,
		"after tree grows to 12000, poller should request entries starting at >= 10000")
}

// TestBacktrack_DiscardChan_ReceivesZeroScoreDomains verifies that the
// discard channel receives domains that scored zero when backtrack causes
// the poller to process historical entries.
func TestBacktrack_DiscardChan_ReceivesZeroScoreDomains(t *testing.T) {
	// The mock scorer returns score=0, so all domains go to discardChan.
	var firstStart atomic.Int64
	firstStart.Store(sentinelStart)
	server := newBacktrackServer(t, 5000, &firstStart)

	discardChan := make(chan string, 100)

	p := NewPoller(
		server.URL, "discard-test",
		&mockScorer{}, &mockStore{}, &domain.Profile{Name: "test"},
		256, 100*time.Millisecond,
		make(chan<- domain.Hit, 10), make(chan<- PollStats, 10),
		discardChan,
		1000,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go p.Run(ctx) //nolint:errcheck

	// Wait for the poller to start (it will request entries starting at 4000).
	deadline := time.After(3 * time.Second)
	for firstStart.Load() == sentinelStart {
		select {
		case <-deadline:
			t.Fatal("poller did not make a get-entries request")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	// With tree_size=5000 and backtrack=1000, poller starts at 4000.
	assert.Equal(t, int64(4000), firstStart.Load())

	cancel()
}
