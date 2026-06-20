// Package domainutil extracts the registrable base domain by heuristic,
// avoiding a golang.org/x/net/publicsuffix dependency.
package domainutil

import "strings"

// Second-level labels under ccTLDs that pull the base domain out to three labels
// (e.g. "example.co.uk" not "co.uk").
var ccTLDParts = map[string]bool{
	"co":  true,
	"com": true,
	"org": true,
	"net": true,
	"gov": true,
	"ac":  true,
	"edu": true,
}

// BaseDomain returns the registrable base domain: the last two labels, or three
// when the second-to-last is a ccTLD part (see ccTLDParts).
//
//	"foo.bar.netflixconfirmation.net" -> "netflixconfirmation.net"
//	"insightandsound.co.uk"           -> "insightandsound.co.uk"
//	"*.sub.example.com"               -> "example.com"
func BaseDomain(domainName string) string {
	domainName = strings.TrimPrefix(domainName, "*.")
	domainName = strings.TrimSuffix(domainName, ".")

	if domainName == "" {
		return ""
	}

	labels := strings.Split(domainName, ".")
	n := len(labels)

	if n <= 2 {
		return domainName
	}

	secondToLast := strings.ToLower(labels[n-2])
	if ccTLDParts[secondToLast] && n >= 3 {
		return strings.Join(labels[n-3:], ".")
	}

	return strings.Join(labels[n-2:], ".")
}
