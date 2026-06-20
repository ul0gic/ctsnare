package poller

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ul0gic/ctsnare/internal/domain"
)

// liveCTEnvVar gates the live-network test; the default suite stays offline
// unless this is set, keeping go test deterministic.
const liveCTEnvVar = "CTSNARE_LIVE_CT"

// liveCTLog is a public RFC 6962 log from the shipped default config.
const liveCTLog = "https://ct.googleapis.com/logs/us1/argon2026h1"

// minLiveParseSuccessRatio is the parse-success floor; ISSUE-004 sank it to ~0%.
const minLiveParseSuccessRatio = 0.95

// TestLiveCTParseSuccess asserts the parser recovers domains from most real
// entries; opt in with CTSNARE_LIVE_CT=1. This is the ISSUE-004 regression net.
func TestLiveCTParseSuccess(t *testing.T) {
	if os.Getenv(liveCTEnvVar) == "" {
		t.Skipf("live CT integration test skipped; set %s=1 to run", liveCTEnvVar)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := NewCTLogClient(liveCTLog)

	sth, err := client.GetSTH(ctx)
	if err != nil {
		t.Fatalf("fetching STH from %s: %v", liveCTLog, err)
	}
	if sth.TreeSize < 64 {
		t.Fatalf("log %s tree too small to sample: %d", liveCTLog, sth.TreeSize)
	}

	// Sample a small, bounded window near the tip. Keep it modest to stay fast
	// and polite to the log operator.
	const window = 32
	start := sth.TreeSize - window
	end := sth.TreeSize - 1

	entries, err := client.GetEntries(ctx, start, end)
	if err != nil {
		t.Fatalf("fetching entries [%d,%d]: %v", start, end, err)
	}
	if len(entries) == 0 {
		t.Fatalf("no entries returned from %s [%d,%d]", liveCTLog, start, end)
	}

	parsed, withDomains := countParsedEntries(t, entries)
	ratio := float64(parsed) / float64(len(entries))
	t.Logf("live CT parse: %d/%d parsed (%.1f%%), %d with domains, window [%d,%d] on %s",
		parsed, len(entries), ratio*100, withDomains, start, end, liveCTLog)

	if ratio < minLiveParseSuccessRatio {
		t.Fatalf("live parse success ratio %.2f below floor %.2f — likely a parser regression (cf. ISSUE-004)",
			ratio, minLiveParseSuccessRatio)
	}
	if withDomains == 0 {
		t.Fatal("no domains extracted from any live entry — parser is recovering certs but not domains")
	}
}

// countParsedEntries parses each entry and returns the number that parsed
// successfully and the number that yielded at least one domain.
func countParsedEntries(t *testing.T, entries []domain.CTLogEntry) (parsed, withDomains int) {
	t.Helper()
	for _, entry := range entries {
		domains, _, err := ParseCertDomains(entry)
		if err != nil {
			t.Logf("parse failure at index %d: %v", entry.Index, err)
			continue
		}
		parsed++
		if len(domains) > 0 {
			withDomains++
		}
	}
	return parsed, withDomains
}
