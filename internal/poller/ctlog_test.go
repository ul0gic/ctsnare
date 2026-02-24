package poller

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSTH_ParsesCorrectly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ct/v1/get-sth", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"tree_size": 12345678, "timestamp": 1700000000000}`)
	}))
	defer server.Close()

	client := NewCTLogClient(server.URL)
	sth, err := client.GetSTH(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int64(12345678), sth.TreeSize)
	assert.Equal(t, int64(1700000000000), sth.Timestamp)
}

func TestGetEntries_ReturnsEntries(t *testing.T) {
	// Create minimal base64-encoded leaf data (at least 15 bytes).
	leafData := make([]byte, 20)
	leafB64 := base64.StdEncoding.EncodeToString(leafData)
	extraB64 := base64.StdEncoding.EncodeToString([]byte("extra"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ct/v1/get-entries", r.URL.Path)
		assert.Equal(t, "0", r.URL.Query().Get("start"))
		assert.Equal(t, "1", r.URL.Query().Get("end"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"entries": [{"leaf_input": "%s", "extra_data": "%s"}]}`, leafB64, extraB64)
	}))
	defer server.Close()

	client := NewCTLogClient(server.URL)
	entries, err := client.GetEntries(context.Background(), 0, 1)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, int64(0), entries[0].Index)
	assert.Equal(t, server.URL, entries[0].LogURL)
}

func TestGetSTH_RateLimitingTriggersBackoff(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)
		if attempt <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"tree_size": 100, "timestamp": 1700000000000}`)
	}))
	defer server.Close()

	client := NewCTLogClient(server.URL)
	sth, err := client.GetSTH(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(100), sth.TreeSize)
	assert.GreaterOrEqual(t, int(attempts.Load()), 3)
}

func TestGetSTH_Non200ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewCTLogClient(server.URL)
	_, err := client.GetSTH(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestGetSTH_InvalidJSONReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{invalid json}`)
	}))
	defer server.Close()

	client := NewCTLogClient(server.URL)
	_, err := client.GetSTH(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decoding STH")
}

func TestGetSTH_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response -- context should cancel first.
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	client := NewCTLogClient(server.URL)
	_, err := client.GetSTH(ctx)
	assert.Error(t, err)
}

func TestGetEntries_InvalidBase64Skipped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"entries": [{"leaf_input": "not-valid-base64!!!", "extra_data": "AAAA"}]}`)
	}))
	defer server.Close()

	client := NewCTLogClient(server.URL)
	entries, err := client.GetEntries(context.Background(), 0, 0)
	require.NoError(t, err)
	// Entry with invalid base64 should be skipped, not cause an error.
	assert.Empty(t, entries)
}
