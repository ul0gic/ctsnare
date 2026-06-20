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

// corpusFixture is a byte-exact leaf_input recorded from a public CT log; real
// DER (poison extensions, RSA precerts, IDN) catches the ISSUE-004 parse class synthetic certs miss.
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

// sigAlgByName maps expected_sig_alg to the x509 enum; an unmapped value is a
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

// loadCorpus decodes every *.json fixture under testdata/corpus, failing if the
// directory is empty so the harness can never silently degrade to a no-op.
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

// TestParseCorpus asserts ParseCertDomains recovers the expected domain set and
// signature algorithm from each recorded real leaf_input.
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

// TestEntryTypePathsExercised hard-fails unless the corpus covers both x509_entry
// and precert_entry, so a fixture deletion can't silently re-open the ISSUE-004 gap.
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
