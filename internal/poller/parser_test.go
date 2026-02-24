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
	// MerkleTreeLeaf: Version(1) + LeafType(1) + Timestamp(8) + EntryType(2)
	header := make([]byte, 12)
	header[0] = 0 // version v1
	header[1] = 0 // timestamped_entry
	binary.BigEndian.PutUint64(header[2:10], uint64(time.Now().UnixMilli()))
	binary.BigEndian.PutUint16(header[10:12], 0) // x509_entry

	// ASN1Cert: 3-byte length prefix + DER cert
	certLen := len(certDER)
	lenPrefix := []byte{
		byte(certLen >> 16),
		byte(certLen >> 8),
		byte(certLen),
	}

	leaf := append(header, lenPrefix...)
	leaf = append(leaf, certDER...)
	return leaf
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
