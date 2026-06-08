package poller

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// buildTestLeafInput constructs a valid MerkleTreeLeaf wrapping a DER certificate
// as an x509_entry (entry_type=0).
func buildTestLeafInput(certDER []byte) []byte {
	// MerkleTreeLeaf: Version(1) + LeafType(1) + Timestamp(8) + EntryType(2),
	// followed by a 3-byte ASN1Cert length prefix and the DER cert.
	header := make([]byte, 12, 12+3+len(certDER))
	header[0] = 0 // version v1
	header[1] = 0 // timestamped_entry
	binary.BigEndian.PutUint64(header[2:10], uint64(time.Now().UnixMilli()))
	binary.BigEndian.PutUint16(header[10:12], 0) // x509_entry

	// ASN1Cert: 3-byte length prefix + DER cert. Mask each byte explicitly to
	// make the 24-bit big-endian truncation intentional.
	certLen := len(certDER)
	lenPrefix := []byte{
		byte((certLen >> 16) & 0xFF),
		byte((certLen >> 8) & 0xFF),
		byte(certLen & 0xFF),
	}

	header = append(header, lenPrefix...)
	header = append(header, certDER...)
	return header
}

// generateTestCert creates a self-signed test certificate with the given
// common name and SAN DNS names.
func generateTestCert(t *testing.T, cn string, dnsNames []string) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
		DNSNames:     dnsNames,
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	return der
}

// buildPrecertLeafInput constructs a valid MerkleTreeLeaf wrapping a bare
// TBSCertificate as a precert_entry (entry_type=1). The PreCert body is:
// issuer_key_hash (32 bytes) + TBSCertificate opaque<1..2^24-1>.
func buildPrecertLeafInput(tbsDER []byte) []byte {
	// MerkleTreeLeaf: Version(1) + LeafType(1) + Timestamp(8) + EntryType(2),
	// followed by issuer_key_hash (32) + 3-byte TBS length prefix + TBS DER.
	header := make([]byte, 12, 12+32+3+len(tbsDER))
	header[0] = 0 // version v1
	header[1] = 0 // timestamped_entry
	binary.BigEndian.PutUint64(header[2:10], uint64(time.Now().UnixMilli()))
	binary.BigEndian.PutUint16(header[10:12], 1) // precert_entry

	// issuer_key_hash: 32 bytes (content is irrelevant to domain extraction).
	issuerKeyHash := make([]byte, 32)

	// TBSCertificate opaque<1..2^24-1>: 3-byte length prefix + TBS DER. Mask
	// each byte explicitly to make the 24-bit big-endian truncation intentional.
	tbsLen := len(tbsDER)
	lenPrefix := []byte{
		byte((tbsLen >> 16) & 0xFF),
		byte((tbsLen >> 8) & 0xFF),
		byte(tbsLen & 0xFF),
	}

	header = append(header, issuerKeyHash...)
	header = append(header, lenPrefix...)
	header = append(header, tbsDER...)
	return header
}

// generateTestTBS creates an ECDSA P-256 certificate and returns its
// RawTBSCertificate — the exact bytes a precert leaf carries.
func generateTestTBS(t *testing.T, cn string, dnsNames []string) []byte {
	t.Helper()
	der := generateTestCert(t, cn, dnsNames)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert.RawTBSCertificate
}

func TestParseCertDomains_ECDSAPrecertExtractsDomains(t *testing.T) {
	// Regression for ISSUE-004: precert (entry_type=1) leaves carrying an
	// ECDSA-signed TBSCertificate must parse. The previous wrapper hardcoded a
	// SHA256WithRSA outer algorithm, which failed x509's inner==outer check for
	// the ECDSA precerts that dominate modern CT logs.
	tbs := generateTestTBS(t, "evil-phish.com", []string{"evil-phish.com", "login.evil-phish.com"})
	leafInput := buildPrecertLeafInput(tbs)

	entry := domain.CTLogEntry{
		LeafInput: leafInput,
		LogURL:    "https://ct.example.com/log",
		Index:     7,
	}

	domains, cert, err := ParseCertDomains(entry)
	require.NoError(t, err, "ECDSA precert must parse without error")
	require.NotNil(t, cert)

	assert.Equal(t, x509.ECDSAWithSHA256, cert.SignatureAlgorithm,
		"reconstructed cert should report the real ECDSA signature algorithm")
	assert.Contains(t, domains, "evil-phish.com")
	assert.Contains(t, domains, "login.evil-phish.com")
}

func TestParseCertDomains_ExtractsCNAndSANs(t *testing.T) {
	der := generateTestCert(t, "example.com", []string{"example.com", "www.example.com", "api.example.com"})
	leafInput := buildTestLeafInput(der)

	entry := domain.CTLogEntry{
		LeafInput: leafInput,
		LogURL:    "https://ct.example.com/log",
		Index:     42,
	}

	domains, cert, err := ParseCertDomains(entry)
	require.NoError(t, err)
	require.NotNil(t, cert)

	assert.Contains(t, domains, "example.com")
	assert.Contains(t, domains, "www.example.com")
	assert.Contains(t, domains, "api.example.com")
}

func TestParseCertDomains_DeduplicatesCNAndSAN(t *testing.T) {
	// CN is also in SANs -- should appear only once.
	der := generateTestCert(t, "example.com", []string{"example.com", "www.example.com"})
	leafInput := buildTestLeafInput(der)

	entry := domain.CTLogEntry{
		LeafInput: leafInput,
		LogURL:    "https://ct.example.com/log",
		Index:     1,
	}

	domains, _, err := ParseCertDomains(entry)
	require.NoError(t, err)

	// Count occurrences of example.com.
	count := 0
	for _, d := range domains {
		if d == "example.com" {
			count++
		}
	}
	assert.Equal(t, 1, count, "CN should not be duplicated when also in SANs")
}

func TestParseCertDomains_MalformedCertReturnsError(t *testing.T) {
	// Build a leaf with garbage certificate data.
	garbage := []byte("this is not a valid DER certificate")
	leafInput := buildTestLeafInput(garbage)

	entry := domain.CTLogEntry{
		LeafInput: leafInput,
		LogURL:    "https://ct.example.com/log",
		Index:     99,
	}

	_, _, err := ParseCertDomains(entry)
	assert.Error(t, err)
}

func TestParseCertDomains_TooShortLeafReturnsError(t *testing.T) {
	entry := domain.CTLogEntry{
		LeafInput: []byte{0, 1, 2},
		LogURL:    "https://ct.example.com/log",
		Index:     1,
	}

	_, _, err := ParseCertDomains(entry)
	assert.Error(t, err)
}

func TestParseCertDomains_CertWithNoDomains(t *testing.T) {
	// Certificate with empty CN and no SANs.
	der := generateTestCert(t, "", nil)
	leafInput := buildTestLeafInput(der)

	entry := domain.CTLogEntry{
		LeafInput: leafInput,
		LogURL:    "https://ct.example.com/log",
		Index:     1,
	}

	domains, _, err := ParseCertDomains(entry)
	require.NoError(t, err)
	assert.Empty(t, domains)
}

func TestUniqueDomains(t *testing.T) {
	der := generateTestCert(t, "foo.com", []string{"foo.com", "bar.com", "foo.com"})
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	domains := uniqueDomains(cert)

	// foo.com appears in CN and twice in SANs, but should appear only once.
	count := 0
	for _, d := range domains {
		if d == "foo.com" {
			count++
		}
	}
	assert.Equal(t, 1, count)
	assert.Contains(t, domains, "bar.com")
}

func TestExtractCertFromLeaf_UnknownEntryType(t *testing.T) {
	leaf := make([]byte, 20)
	binary.BigEndian.PutUint16(leaf[10:12], 99) // unknown type

	_, err := extractCertFromLeaf(leaf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown entry type")
}
