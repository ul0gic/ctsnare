// Package poller implements Certificate Transparency log polling.
package poller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ul0gic/ctsnare/internal/domain"
)

// SignedTreeHead represents the response from the CT log get-sth endpoint.
// It reports the current size of the log tree and when it was last updated.
type SignedTreeHead struct {
	// TreeSize is the total number of entries in the CT log.
	TreeSize int64 `json:"tree_size"`

	// Timestamp is the Unix millisecond timestamp when this tree head was signed.
	Timestamp int64 `json:"timestamp"`
}

// ctlogEntry is the JSON structure returned by the get-entries endpoint.
type ctlogEntry struct {
	LeafInput string `json:"leaf_input"`
	ExtraData string `json:"extra_data"`
}

// ctlogEntriesResponse wraps the get-entries JSON response.
type ctlogEntriesResponse struct {
	Entries []ctlogEntry `json:"entries"`
}

// CTLogClient communicates with a single Certificate Transparency log
// using the RFC 6962 HTTP API.
type CTLogClient struct {
	httpClient *http.Client
	baseURL    string
}

// maxResponseBodySize caps HTTP response reads at 50 MB to prevent
// memory exhaustion from a compromised or malicious CT log server.
const maxResponseBodySize = 50 * 1024 * 1024

// limitedReadCloser wraps a size-limited Reader with the original body's
// Close method, ensuring the underlying connection is released.
type limitedReadCloser struct {
	io.Reader
	io.Closer
}

// NewCTLogClient creates a client for the given CT log base URL.
// Redirects are disabled because CT log APIs should never redirect,
// and following redirects could enable SSRF to internal endpoints.
func NewCTLogClient(baseURL string) *CTLogClient {
	return &CTLogClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		baseURL: baseURL,
	}
}

// GetSTH fetches the Signed Tree Head from the CT log.
func (c *CTLogClient) GetSTH(ctx context.Context) (*SignedTreeHead, error) {
	url := c.baseURL + "/ct/v1/get-sth"

	body, err := c.doGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetching STH from %s: %w", c.baseURL, err)
	}
	defer body.Close()

	var sth SignedTreeHead
	if err := json.NewDecoder(body).Decode(&sth); err != nil {
		return nil, fmt.Errorf("decoding STH from %s: %w", c.baseURL, err)
	}

	return &sth, nil
}

// GetEntries fetches a range of entries from the CT log.
func (c *CTLogClient) GetEntries(ctx context.Context, start, end int64) ([]domain.CTLogEntry, error) {
	url := fmt.Sprintf("%s/ct/v1/get-entries?start=%d&end=%d", c.baseURL, start, end)

	body, err := c.doGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetching entries [%d,%d] from %s: %w", start, end, c.baseURL, err)
	}
	defer body.Close()

	var resp ctlogEntriesResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decoding entries from %s: %w", c.baseURL, err)
	}

	entries := make([]domain.CTLogEntry, 0, len(resp.Entries))
	for i, e := range resp.Entries {
		leafInput, err := decodeBase64(e.LeafInput)
		if err != nil {
			slog.Warn("skipping entry with invalid leaf_input",
				"log", c.baseURL, "index", start+int64(i), "error", err)
			continue
		}
		extraData, err := decodeBase64(e.ExtraData)
		if err != nil {
			slog.Warn("skipping entry with invalid extra_data",
				"log", c.baseURL, "index", start+int64(i), "error", err)
			continue
		}

		entries = append(entries, domain.CTLogEntry{
			LeafInput: leafInput,
			ExtraData: extraData,
			Index:     start + int64(i),
			LogURL:    c.baseURL,
		})
	}

	return entries, nil
}

// doGet executes an HTTP GET with context, handling rate limiting (429) with
// exponential backoff.
func (c *CTLogClient) doGet(ctx context.Context, url string) (io.ReadCloser, error) {
	maxRetries := 3
	backoff := 1 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("executing request: %w", err)
		}

		if resp.StatusCode == http.StatusOK {
			// Cap response size to prevent memory exhaustion from
			// oversized payloads. The underlying body is still closed
			// by the caller via the wrapping ReadCloser.
			limited := &limitedReadCloser{
				Reader: io.LimitReader(resp.Body, maxResponseBodySize),
				Closer: resp.Body,
			}
			return limited, nil
		}

		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxRetries {
			slog.Debug("rate limited, backing off",
				"url", url, "attempt", attempt+1, "backoff", backoff)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
				continue
			}
		}

		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	return nil, fmt.Errorf("max retries exceeded for %s", url)
}
