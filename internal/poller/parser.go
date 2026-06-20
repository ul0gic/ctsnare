package poller

import (
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ul0gic/ctsnare/internal/domain"
)

// maxCertDERBytes caps opaque DER blobs (RFC 6962 24-bit length prefix allows
// ~16 MB) so a hostile length prefix cannot drive an outsized allocation.
const maxCertDERBytes = 1 << 19 // 512 KiB

// ParseCertDomains returns the Subject CN plus all DNS SANs from a CT log entry.
// Callers should skip the entry on error, not crash.
func ParseCertDomains(entry domain.CTLogEntry) ([]string, *x509.Certificate, error) {
	certBytes, err := extractCertFromLeaf(entry.LeafInput)
	if err != nil {
		return nil, nil, fmt.Errorf("extracting certificate from leaf: %w", err)
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing x509 certificate: %w", err)
	}

	return uniqueDomains(cert), cert, nil
}

// extractCertFromLeaf decodes a MerkleTreeLeaf (RFC 6962 §3.4) and returns the
// DER-encoded certificate. Byte offsets below follow that wire layout.
func extractCertFromLeaf(leafInput []byte) ([]byte, error) {
	if len(leafInput) < 15 {
		return nil, fmt.Errorf("leaf input too short: %d bytes", len(leafInput))
	}

	// Skip Version (1) + LeafType (1) + Timestamp (8) = 10 bytes.
	entryType := binary.BigEndian.Uint16(leafInput[10:12])

	switch entryType {
	case 0: // x509_entry
		return readOpaqueDER(leafInput, 12, "x509_entry cert")

	case 1: // precert_entry
		// PreCert is issuer_key_hash (32 bytes) + TBSCertificate opaque.
		if len(leafInput) < 12+32+3 {
			return nil, errors.New("precert_entry too short")
		}
		tbsBytes, err := readOpaqueDER(leafInput, 12+32, "precert TBS")
		if err != nil {
			return nil, err
		}
		return wrapTBSCertificate(tbsBytes)

	default:
		return nil, fmt.Errorf("unknown entry type: %d", entryType)
	}
}

// readOpaqueDER parses an RFC 6962 opaque<1..2^24-1> field: a 3-byte big-endian
// length prefix at lenOffset, capped at maxCertDERBytes before any slicing.
func readOpaqueDER(buf []byte, lenOffset int, label string) ([]byte, error) {
	if len(buf) < lenOffset+3 {
		return nil, fmt.Errorf("%s too short", label)
	}
	n := int(buf[lenOffset])<<16 | int(buf[lenOffset+1])<<8 | int(buf[lenOffset+2])
	if n > maxCertDERBytes {
		return nil, fmt.Errorf("%s too large: %d bytes", label, n)
	}
	start := lenOffset + 3
	if len(buf) < start+n {
		return nil, fmt.Errorf("%s truncated: need %d, have %d", label, start+n, len(buf))
	}
	return buf[start : start+n], nil
}

// wrapTBSCertificate wraps a precert TBSCertificate in a minimal Certificate;
// the outer signatureAlgorithm reuses the inner DER bytes (ParseCertificate enforces inner == outer).
func wrapTBSCertificate(tbs []byte) ([]byte, error) {
	algBytes, err := extractTBSSignatureAlgorithm(tbs)
	if err != nil {
		return nil, fmt.Errorf("extracting TBS signature algorithm: %w", err)
	}

	// Empty signature: ParseCertificate does not verify the value.
	sigBytes, err := asn1.Marshal(asn1.BitString{Bytes: []byte{}, BitLength: 0})
	if err != nil {
		return nil, fmt.Errorf("marshaling dummy signature: %w", err)
	}

	inner := make([]byte, 0, len(tbs)+len(algBytes)+len(sigBytes))
	inner = append(inner, tbs...)
	inner = append(inner, algBytes...)
	inner = append(inner, sigBytes...)

	outer := asn1.RawValue{
		Class:      asn1.ClassUniversal,
		Tag:        asn1.TagSequence,
		IsCompound: true,
		Bytes:      inner,
	}

	result, err := asn1.Marshal(outer)
	if err != nil {
		return nil, fmt.Errorf("marshaling certificate wrapper: %w", err)
	}

	return result, nil
}

// extractTBSSignatureAlgorithm returns the raw DER of the signature
// AlgorithmIdentifier, the third TBSCertificate element after optional version and serialNumber.
func extractTBSSignatureAlgorithm(tbs []byte) ([]byte, error) {
	seq, err := unmarshalSequence(tbs, "TBSCertificate")
	if err != nil {
		return nil, err
	}

	rest := seq.Bytes

	// version [0] EXPLICIT is optional; skip it only when the context tag is present.
	var first asn1.RawValue
	remaining, err := asn1.Unmarshal(rest, &first)
	if err != nil {
		return nil, fmt.Errorf("parsing first TBS element: %w", err)
	}
	if first.Class == asn1.ClassContextSpecific && first.Tag == 0 {
		rest = remaining
	}

	var serial asn1.RawValue
	rest, err = asn1.Unmarshal(rest, &serial)
	if err != nil {
		return nil, fmt.Errorf("parsing serialNumber: %w", err)
	}

	sigAlg, err := unmarshalSequence(rest, "signature AlgorithmIdentifier")
	if err != nil {
		return nil, err
	}
	return sigAlg.FullBytes, nil
}

// unmarshalSequence parses the next ASN.1 value and asserts it is a compound SEQUENCE.
func unmarshalSequence(b []byte, name string) (asn1.RawValue, error) {
	var seq asn1.RawValue
	if _, err := asn1.Unmarshal(b, &seq); err != nil {
		return seq, fmt.Errorf("parsing %s sequence: %w", name, err)
	}
	if seq.Tag != asn1.TagSequence || !seq.IsCompound {
		return seq, fmt.Errorf("%s is not a SEQUENCE", name)
	}
	return seq, nil
}

// uniqueDomains returns the certificate's Common Name plus DNS SANs, deduplicated.
func uniqueDomains(cert *x509.Certificate) []string {
	seen := make(map[string]struct{})
	var domains []string

	if cert.Subject.CommonName != "" {
		cn := cert.Subject.CommonName
		if _, ok := seen[cn]; !ok {
			seen[cn] = struct{}{}
			domains = append(domains, cn)
		}
	}

	for _, san := range cert.DNSNames {
		if _, ok := seen[san]; !ok {
			seen[san] = struct{}{}
			domains = append(domains, san)
		}
	}

	return domains
}

func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func logParseWarning(logURL string, index int64, err error) {
	slog.Warn("failed to parse certificate",
		"log", logURL, "index", index, "error", err)
}
