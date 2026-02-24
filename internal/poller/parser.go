package poller

import (
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log/slog"

	"github.com/ul0gic/ctsnare/internal/domain"
)

// ParseCertDomains extracts all unique domain names from a CT log entry.
// It decodes the MerkleTreeLeaf structure, parses the x509 certificate,
// and returns the Subject CN plus all DNS SANs. Parse errors are logged
// and returned; they should be handled gracefully (skip entry, don't crash).
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

// extractCertFromLeaf decodes the MerkleTreeLeaf structure (RFC 6962 section 3.4)
// and extracts the DER-encoded certificate.
//
// MerkleTreeLeaf structure:
//
//	struct {
//	    Version version;                      // 1 byte (0 = v1)
//	    MerkleLeafType leaf_type;             // 1 byte (0 = timestamped_entry)
//	    select (leaf_type) {
//	        case timestamped_entry: TimestampedEntry;
//	    }
//	} MerkleTreeLeaf;
//
//	struct {
//	    uint64 timestamp;                     // 8 bytes
//	    LogEntryType entry_type;              // 2 bytes (0 = x509_entry, 1 = precert_entry)
//	    select (entry_type) {
//	        case x509_entry: ASN1Cert;        // opaque<1..2^24-1>
//	        case precert_entry: PreCert;
//	    }
//	    CtExtensions extensions;
//	} TimestampedEntry;
func extractCertFromLeaf(leafInput []byte) ([]byte, error) {
	if len(leafInput) < 15 {
		return nil, fmt.Errorf("leaf input too short: %d bytes", len(leafInput))
	}

	// Skip Version (1) + LeafType (1) + Timestamp (8) = 10 bytes.
	entryType := binary.BigEndian.Uint16(leafInput[10:12])

	switch entryType {
	case 0: // x509_entry
		// ASN1Cert is an opaque<1..2^24-1>: 3-byte length prefix + DER cert.
		if len(leafInput) < 15 {
			return nil, fmt.Errorf("x509_entry too short")
		}
		certLen := int(leafInput[12])<<16 | int(leafInput[13])<<8 | int(leafInput[14])
		if len(leafInput) < 15+certLen {
			return nil, fmt.Errorf("x509_entry cert truncated: need %d, have %d", 15+certLen, len(leafInput))
		}
		return leafInput[15 : 15+certLen], nil

	case 1: // precert_entry
		// PreCert: issuer_key_hash (32 bytes) + TBSCertificate opaque<1..2^24-1>.
		offset := 12
		if len(leafInput) < offset+32+3 {
			return nil, fmt.Errorf("precert_entry too short")
		}
		offset += 32 // skip issuer_key_hash
		tbsLen := int(leafInput[offset])<<16 | int(leafInput[offset+1])<<8 | int(leafInput[offset+2])
		offset += 3
		if len(leafInput) < offset+tbsLen {
			return nil, fmt.Errorf("precert TBS truncated")
		}
		tbsBytes := leafInput[offset : offset+tbsLen]

		// Try to parse the TBSCertificate directly.
		// Pre-certificates contain a TBSCertificate without the signature.
		// We wrap it in a minimal Certificate structure for x509 parsing.
		return wrapTBSCertificate(tbsBytes)

	default:
		return nil, fmt.Errorf("unknown entry type: %d", entryType)
	}
}

// wrapTBSCertificate wraps a raw TBSCertificate in a minimal ASN.1
// Certificate structure so x509.ParseCertificate can handle it.
// This is needed for pre-certificates (entry_type=1).
func wrapTBSCertificate(tbs []byte) ([]byte, error) {
	// A Certificate is: SEQUENCE { TBSCertificate, AlgorithmIdentifier, BIT STRING }
	// We use a dummy signature algorithm (SHA256WithRSA) and empty signature.
	dummyAlgID := asn1.RawValue{
		Class:      asn1.ClassUniversal,
		Tag:        asn1.TagSequence,
		IsCompound: true,
		Bytes: func() []byte {
			// SHA256WithRSA OID: 1.2.840.113549.1.1.11
			oid, _ := asn1.Marshal(asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 11})
			null, _ := asn1.Marshal(asn1.RawValue{Tag: asn1.TagNull})
			return append(oid, null...)
		}(),
	}
	algBytes, err := asn1.Marshal(dummyAlgID)
	if err != nil {
		return nil, fmt.Errorf("marshaling dummy algorithm: %w", err)
	}

	// Empty signature as BIT STRING.
	sigBytes, err := asn1.Marshal(asn1.BitString{Bytes: []byte{}, BitLength: 0})
	if err != nil {
		return nil, fmt.Errorf("marshaling dummy signature: %w", err)
	}

	// Wrap in outer SEQUENCE.
	inner := append(tbs, algBytes...)
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

// uniqueDomains extracts all unique domain names from a certificate:
// the Subject Common Name and all DNS Subject Alternative Names.
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

// decodeBase64 decodes a standard base64-encoded string, as used by CT log
// JSON responses.
func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// logParseWarning logs a certificate parse warning without panicking.
func logParseWarning(logURL string, index int64, err error) {
	slog.Warn("failed to parse certificate",
		"log", logURL, "index", index, "error", err)
}
