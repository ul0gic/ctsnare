package enrichment

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbeLiveness_LiveHTTPSServer(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = 5 * time.Second

	// Use the raw server listener address (host:port).
	addr := server.Listener.Addr().String()
	statusCode, isLive, err := ProbeLiveness(client, addr)

	require.NoError(t, err)
	assert.True(t, isLive, "TLS test server should be detected as live")
	assert.Equal(t, http.StatusOK, statusCode)
}

func TestProbeLiveness_HTTPFallback(t *testing.T) {
	// Start a plain HTTP server (no TLS). ProbeLiveness should fail HTTPS
	// then succeed on HTTP fallback.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 3 * time.Second}
	addr := server.Listener.Addr().String()

	statusCode, isLive, err := ProbeLiveness(client, addr)
	require.NoError(t, err)
	assert.True(t, isLive, "HTTP-only server should still be detected as live via fallback")
	assert.Equal(t, http.StatusOK, statusCode)
}

func TestProbeLiveness_NoServer(t *testing.T) {
	// Use an address where nothing is listening.
	client := &http.Client{Timeout: 1 * time.Second}

	statusCode, isLive, err := ProbeLiveness(client, "127.0.0.1:1")
	assert.Error(t, err, "probe should fail when no server is running")
	assert.False(t, isLive)
	assert.Equal(t, 0, statusCode)
}

func TestProbeLiveness_ServerReturns4xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 3 * time.Second}
	addr := server.Listener.Addr().String()

	statusCode, isLive, err := ProbeLiveness(client, addr)
	require.NoError(t, err)
	assert.True(t, isLive, "4xx response still means the server is live")
	assert.Equal(t, http.StatusForbidden, statusCode)
}

func TestProbeLiveness_ServerReturns5xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 3 * time.Second}
	addr := server.Listener.Addr().String()

	statusCode, isLive, err := ProbeLiveness(client, addr)
	require.NoError(t, err)
	assert.True(t, isLive, "5xx response still means the server is live")
	assert.Equal(t, http.StatusInternalServerError, statusCode)
}

func TestProbeLiveness_Redirects(t *testing.T) {
	var redirectCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := redirectCount.Add(1)
		if count <= 2 {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	addr := server.Listener.Addr().String()

	statusCode, isLive, err := ProbeLiveness(client, addr)
	require.NoError(t, err)
	assert.True(t, isLive, "server that redirects should still be detected as live")
	assert.Equal(t, http.StatusOK, statusCode)
}

func TestProbeLiveness_Timeout(t *testing.T) {
	// Server that never responds (sleeps longer than client timeout).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(10 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 500 * time.Millisecond}
	addr := server.Listener.Addr().String()

	start := time.Now()
	statusCode, isLive, err := ProbeLiveness(client, addr)
	elapsed := time.Since(start)

	assert.Error(t, err, "slow server should cause timeout error")
	assert.False(t, isLive)
	assert.Equal(t, 0, statusCode)
	assert.Less(t, elapsed, 5*time.Second, "should timeout quickly, not wait for full server sleep")
}

func TestProbeLiveness_HEADMethod(t *testing.T) {
	var receivedMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 3 * time.Second}
	addr := server.Listener.Addr().String()

	_, _, err := ProbeLiveness(client, addr)
	require.NoError(t, err)
	assert.Equal(t, http.MethodHead, receivedMethod, "ProbeLiveness should use HEAD method")
}

func TestProbeLiveness_UserAgent(t *testing.T) {
	var receivedUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 3 * time.Second}
	addr := server.Listener.Addr().String()

	_, _, err := ProbeLiveness(client, addr)
	require.NoError(t, err)
	assert.Equal(t, userAgent, receivedUA, "ProbeLiveness should send the configured User-Agent header")
}

func TestDoHEAD_ValidURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 3 * time.Second}
	code, err := doHEAD(client, server.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusTeapot, code)
}

func TestDoHEAD_InvalidURL(t *testing.T) {
	client := &http.Client{Timeout: 1 * time.Second}
	_, err := doHEAD(client, "://invalid-url")
	assert.Error(t, err, "invalid URL should produce an error")
}
