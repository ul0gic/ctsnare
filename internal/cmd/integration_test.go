package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ul0gic/ctsnare/internal/domain"
	"github.com/ul0gic/ctsnare/internal/storage"
)

// resetFlags clears package-level flag variables that leak between tests
// because Cobra shares a single rootCmd with persistent flags.
func resetFlags() {
	cfgFile = ""
	dbPath = ""
	verbose = false
	queryKeyword = ""
	queryScoreMin = 0
	querySince = 0
	queryTLD = ""
	querySession = ""
	querySeverity = ""
	queryFormat = "table"
	queryLimit = 50
	dbClearConfirm = false
	dbClearSession = ""
	dbExportFormat = "jsonl"
	dbExportOutput = ""
}

// testHits returns a set of known hits for populating a test database.
func testHits() []domain.Hit {
	return []domain.Hit{
		{
			Domain:   "fake-bitcoin-exchange.xyz",
			Score:    8,
			Severity: domain.SeverityHigh,
			Keywords: []string{"bitcoin", "exchange"},
			Issuer:   "Let's Encrypt",
			IssuerCN: "R3",
			SANDomains: []string{
				"fake-bitcoin-exchange.xyz",
				"www.fake-bitcoin-exchange.xyz",
			},
			CertNotBefore: time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC),
			CTLog:         "Google Argon 2025h1",
			Profile:       "crypto",
			Session:       "test-session",
		},
		{
			Domain:   "secure-paypal-login.top",
			Score:    6,
			Severity: domain.SeverityHigh,
			Keywords: []string{"paypal", "login", "secure"},
			Issuer:   "Let's Encrypt",
			IssuerCN: "R3",
			SANDomains: []string{
				"secure-paypal-login.top",
			},
			CertNotBefore: time.Date(2026, 2, 24, 11, 0, 0, 0, time.UTC),
			CTLog:         "Google Xenon 2025h1",
			Profile:       "phishing",
			Session:       "test-session",
		},
		{
			Domain:   "mywalletcrypto.com",
			Score:    4,
			Severity: domain.SeverityMed,
			Keywords: []string{"wallet", "crypto"},
			Issuer:   "DigiCert",
			IssuerCN: "DigiCert CN",
			SANDomains: []string{
				"mywalletcrypto.com",
			},
			CertNotBefore: time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC),
			CTLog:         "Google Argon 2025h2",
			Profile:       "crypto",
			Session:       "other-session",
		},
		{
			Domain:   "free-token-claim.buzz",
			Score:    2,
			Severity: domain.SeverityLow,
			Keywords: []string{"token"},
			Issuer:   "Let's Encrypt",
			IssuerCN: "E1",
			SANDomains: []string{
				"free-token-claim.buzz",
			},
			CertNotBefore: time.Date(2026, 2, 24, 13, 0, 0, 0, time.UTC),
			CTLog:         "Google Argon 2025h1",
			Profile:       "all",
			Session:       "test-session",
		},
	}
}

// setupTestDB creates a temporary database and populates it with test hits.
func setupTestDB(t *testing.T) (string, *storage.DB) {
	t.Helper()
	dbFile := filepath.Join(t.TempDir(), "test.db")
	store, err := storage.NewDB(dbFile)
	require.NoError(t, err)

	ctx := context.Background()
	for _, hit := range testHits() {
		require.NoError(t, store.UpsertHit(ctx, hit))
	}

	return dbFile, store
}

// captureStdout runs a function while capturing stdout, returning the output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, readErr := buf.ReadFrom(r)
	require.NoError(t, readErr)
	return buf.String()
}

func TestQueryCommandWithPrePopulatedDB(t *testing.T) {
	dbFile, store := setupTestDB(t)
	defer store.Close()

	tests := []struct {
		name     string
		args     []string
		contains []string
		notEmpty bool
	}{
		{
			name:     "table format default",
			args:     []string{"query", "--db", dbFile},
			contains: []string{"fake-bitcoin-exchange.xyz", "SEVERITY", "SCORE"},
			notEmpty: true,
		},
		{
			name:     "json format",
			args:     []string{"query", "--db", dbFile, "--format", "json"},
			contains: []string{"fake-bitcoin-exchange.xyz", "\"Score\""},
			notEmpty: true,
		},
		{
			name:     "csv format",
			args:     []string{"query", "--db", dbFile, "--format", "csv"},
			contains: []string{"severity,score,domain"},
			notEmpty: true,
		},
		{
			name:     "filter by keyword",
			args:     []string{"query", "--db", dbFile, "--keyword", "bitcoin"},
			contains: []string{"fake-bitcoin-exchange.xyz"},
			notEmpty: true,
		},
		{
			name:     "filter by severity HIGH",
			args:     []string{"query", "--db", dbFile, "--severity", "HIGH"},
			contains: []string{"fake-bitcoin-exchange.xyz"},
			notEmpty: true,
		},
		{
			name:     "filter by session",
			args:     []string{"query", "--db", dbFile, "--session", "other-session"},
			contains: []string{"mywalletcrypto.com"},
			notEmpty: true,
		},
		{
			name:     "filter by score-min",
			args:     []string{"query", "--db", dbFile, "--score-min", "6"},
			contains: []string{"fake-bitcoin-exchange.xyz"},
			notEmpty: true,
		},
		{
			name:     "limit results to 2",
			args:     []string{"query", "--db", dbFile, "--limit", "2"},
			notEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags()

			output := captureStdout(t, func() {
				rootCmd.SetArgs(tt.args)
				execErr := rootCmd.Execute()
				assert.NoError(t, execErr)
			})

			if tt.notEmpty {
				assert.NotEmpty(t, output, "expected non-empty output")
			}
			for _, s := range tt.contains {
				assert.Contains(t, output, s, "expected output to contain %q", s)
			}
		})
	}
}

func TestDBStatsWithKnownData(t *testing.T) {
	resetFlags()
	dbFile, store := setupTestDB(t)
	defer store.Close()

	output := captureStdout(t, func() {
		rootCmd.SetArgs([]string{"db", "stats", "--db", dbFile})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Total Hits:  4")
	assert.Contains(t, output, "HIGH")
	assert.Contains(t, output, "MED")
	assert.Contains(t, output, "LOW")
	assert.Contains(t, output, "Top Keywords")
}

func TestDBClearRequiresConfirm(t *testing.T) {
	resetFlags()
	dbFile, store := setupTestDB(t)
	defer store.Close()

	rootCmd.SetArgs([]string{"db", "clear", "--db", dbFile})
	err := rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--confirm")
}

func TestDBClearWithConfirm(t *testing.T) {
	resetFlags()
	dbFile, store := setupTestDB(t)

	rootCmd.SetArgs([]string{"db", "clear", "--db", dbFile, "--confirm"})
	err := rootCmd.Execute()
	assert.NoError(t, err)

	// Verify database is empty.
	ctx := context.Background()
	stats, statsErr := store.Stats(ctx)
	require.NoError(t, statsErr)
	assert.Equal(t, 0, stats.TotalHits)
	store.Close()
}

func TestDBClearSession(t *testing.T) {
	resetFlags()
	dbFile, store := setupTestDB(t)

	rootCmd.SetArgs([]string{"db", "clear", "--db", dbFile, "--confirm", "--session", "test-session"})
	err := rootCmd.Execute()
	assert.NoError(t, err)

	// Verify only the target session was cleared.
	ctx := context.Background()
	stats, statsErr := store.Stats(ctx)
	require.NoError(t, statsErr)
	assert.Equal(t, 1, stats.TotalHits, "expected 1 hit remaining (other-session)")
	store.Close()
}

func TestDBExportJSONL(t *testing.T) {
	resetFlags()
	dbFile, store := setupTestDB(t)
	defer store.Close()

	outputFile := filepath.Join(t.TempDir(), "export.jsonl")
	rootCmd.SetArgs([]string{"db", "export", "--db", dbFile, "--format", "jsonl", "--output", outputFile})
	err := rootCmd.Execute()
	assert.NoError(t, err)

	data, readErr := os.ReadFile(outputFile)
	require.NoError(t, readErr)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Len(t, lines, 4, "expected 4 JSONL lines for 4 hits")
}

func TestDBExportCSV(t *testing.T) {
	resetFlags()
	dbFile, store := setupTestDB(t)
	defer store.Close()

	outputFile := filepath.Join(t.TempDir(), "export.csv")
	rootCmd.SetArgs([]string{"db", "export", "--db", dbFile, "--format", "csv", "--output", outputFile})
	err := rootCmd.Execute()
	assert.NoError(t, err)

	data, readErr := os.ReadFile(outputFile)
	require.NoError(t, readErr)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Len(t, lines, 5, "expected 5 CSV lines: 1 header + 4 data rows")
	assert.Contains(t, lines[0], "domain")
}

func TestDBPath(t *testing.T) {
	resetFlags()

	output := captureStdout(t, func() {
		rootCmd.SetArgs([]string{"db", "path"})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	trimmed := strings.TrimSpace(output)
	assert.Contains(t, trimmed, "ctsnare.db", "path should contain the database filename")
}

func TestProfilesList(t *testing.T) {
	resetFlags()

	output := captureStdout(t, func() {
		rootCmd.SetArgs([]string{"profiles"})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "all")
	assert.Contains(t, output, "crypto")
	assert.Contains(t, output, "phishing")
}

func TestProfilesShow(t *testing.T) {
	resetFlags()

	output := captureStdout(t, func() {
		rootCmd.SetArgs([]string{"profiles", "show", "crypto"})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Profile: crypto")
	assert.Contains(t, output, "Keywords (20)")
	assert.Contains(t, output, "bitcoin")
	assert.Contains(t, output, "Suspicious TLDs")
	assert.Contains(t, output, ".xyz")
	assert.Contains(t, output, "Skip Suffixes")
}

func TestProfilesShowUnknown(t *testing.T) {
	resetFlags()
	rootCmd.SetArgs([]string{"profiles", "show", "nonexistent"})
	err := rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown profile")
}

func TestQueryNoDatabase(t *testing.T) {
	resetFlags()
	rootCmd.SetArgs([]string{"query", "--db", "/tmp/nonexistent-ctsnare-test.db"})
	err := rootCmd.Execute()
	// The query command prints a message and returns nil when DB doesn't exist.
	assert.NoError(t, err)
}

func TestRootHelpShowsSubcommands(t *testing.T) {
	resetFlags()

	output := captureStdout(t, func() {
		rootCmd.SetArgs([]string{"--help"})
		_ = rootCmd.Execute()
	})

	assert.Contains(t, output, "watch")
	assert.Contains(t, output, "query")
	assert.Contains(t, output, "db")
	assert.Contains(t, output, "profiles")
}
