package poller

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// corpusFixture is a recorded CT-log entry stored under testdata/corpus. Each
// fixture carries a real, byte-exact leaf_input captured from a public CT log
// (provenance in source_log + index + captured_at) along with the domains and
// signature algorithm the parser is expected to recover. Synthetic certs cannot
// reproduce real-world DER structures (poison extensions, real issuer chains,
// RSA precerts, IDN labels), so this corpus guards the parser against the class
// of bug that caused ISSUE-004 — a parse failure that only manifests on real
// precertificates.
type corpusFixture struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	SourceLog       string   `json:"source_log"`
	Index           int64    `json:"index"`
	CapturedAt      string   `json:"captured_at"`
	EntryType       int      `json:"entry_type"`
	LeafInputB64    string   `json:"leaf_input_b64"`
	ExpectedDomains []string `json:"expected_domains"`
	ExpectedSigAlg  string   `json:"expected_sig_alg"`

	// path is the source file, populated by loadCorpus for clearer failures.
	path string
}

// sigAlgByName maps the fixture's expected_sig_alg string to the x509 enum.
// Only the algorithms present in the corpus are listed; an unmapped value is a
// test error so a fixture can't silently skip the signature-algorithm check.
var sigAlgByName = map[string]x509.SignatureAlgorithm{
	"ECDSAWithSHA256":  x509.ECDSAWithSHA256,
	"ECDSAWithSHA384":  x509.ECDSAWithSHA384,
	"ECDSAWithSHA512":  x509.ECDSAWithSHA512,
	"SHA256WithRSA":    x509.SHA256WithRSA,
	"SHA384WithRSA":    x509.SHA384WithRSA,
	"SHA512WithRSA":    x509.SHA512WithRSA,
	"SHA256WithRSAPSS": x509.SHA256WithRSAPSS,
	"SHA384WithRSAPSS": x509.SHA384WithRSAPSS,
}

// corpusDir is the directory holding recorded fixtures relative to this package.
const corpusDir = "testdata/corpus"

// loadCorpus reads and decodes every *.json fixture under testdata/corpus.
// It fails the test if the directory is empty so the harness can never silently
// degrade to a no-op.
func loadCorpus(t *testing.T) []corpusFixture {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(corpusDir, "*.json"))
	require.NoError(t, err, "globbing corpus fixtures")
	require.NotEmpty(t, paths, "no corpus fixtures found under %s — the recorded harness must not be empty", corpusDir)

	fixtures := make([]corpusFixture, 0, len(paths))
	for _, p := range paths {
		data, err := os.ReadFile(p) //nolint:gosec // test fixture path from a fixed glob
		require.NoError(t, err, "reading fixture %s", p)

		var f corpusFixture
		require.NoError(t, json.Unmarshal(data, &f), "decoding fixture %s", p)
		f.path = p
		fixtures = append(fixtures, f)
	}
	return fixtures
}

// TestParseCorpus feeds each recorded real leaf_input through the production
// ParseCertDomains and asserts the expected domains (as an order-independent
// set) and, where recorded, the reconstructed signature algorithm. This is the
// durable regression net for real CT-log structures.
func TestParseCorpus(t *testing.T) {
	for _, f := range loadCorpus(t) {
		t.Run(f.Name, func(t *testing.T) {
			require.NotEmpty(t, f.LeafInputB64, "fixture %s has empty leaf_input_b64", f.path)
			require.NotEmpty(t, f.ExpectedDomains, "fixture %s must assert at least one expected domain", f.path)

			leafInput, err := base64.StdEncoding.DecodeString(f.LeafInputB64)
			require.NoError(t, err, "decoding leaf_input_b64 for %s", f.Name)

			entry := domain.CTLogEntry{
				LeafInput: leafInput,
				LogURL:    f.SourceLog,
				Index:     f.Index,
			}

			domains, cert, err := ParseCertDomains(entry)
			require.NoError(t, err, "%s (%s index %d) must parse", f.Name, f.SourceLog, f.Index)
			require.NotNil(t, cert)

			assert.ElementsMatch(t, f.ExpectedDomains, domains,
				"%s: extracted domains must match the recorded set", f.Name)

			if f.ExpectedSigAlg != "" {
				want, ok := sigAlgByName[f.ExpectedSigAlg]
				require.Truef(t, ok, "fixture %s lists unknown expected_sig_alg %q — add it to sigAlgByName", f.Name, f.ExpectedSigAlg)
				assert.Equalf(t, want, cert.SignatureAlgorithm,
					"%s: reconstructed cert should report the recorded signature algorithm", f.Name)
			}
		})
	}
}

// TestEntryTypePathsExercised is a coverage-erosion guard: it hard-fails if the
// corpus does not cover BOTH RFC 6962 entry types — x509_entry (0) and
// precert_entry (1). The precert path carried the ISSUE-004 bug and previously
// had zero coverage; this guard ensures a future fixture deletion can't silently
// re-open that gap. It also confirms each represented type actually parses.
func TestEntryTypePathsExercised(t *testing.T) {
	fixtures := loadCorpus(t)

	parsedByType := map[int][]string{}
	for _, f := range fixtures {
		leafInput, err := base64.StdEncoding.DecodeString(f.LeafInputB64)
		require.NoErrorf(t, err, "decoding leaf_input_b64 for %s", f.Name)

		_, _, err = ParseCertDomains(domain.CTLogEntry{LeafInput: leafInput, LogURL: f.SourceLog, Index: f.Index})
		require.NoErrorf(t, err, "fixture %s (entry_type=%d) must parse", f.Name, f.EntryType)

		parsedByType[f.EntryType] = append(parsedByType[f.EntryType], f.Name)
	}

	require.NotEmptyf(t, parsedByType[0],
		"corpus must include at least one x509_entry (entry_type=0) fixture that parses; have types %v", entryTypesPresent(parsedByType))
	require.NotEmptyf(t, parsedByType[1],
		"corpus must include at least one precert_entry (entry_type=1) fixture that parses; have types %v", entryTypesPresent(parsedByType))
}

// entryTypesPresent returns the sorted entry types present in the corpus, for
// readable guard-test failure messages.
func entryTypesPresent(byType map[int][]string) []int {
	types := make([]int, 0, len(byType))
	for et := range byType {
		types = append(types, et)
	}
	sort.Ints(types)
	return types
}
