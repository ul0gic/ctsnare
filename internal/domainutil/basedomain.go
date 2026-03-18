// Package domainutil provides lightweight domain name utilities for ctsnare.
// It avoids external dependencies like golang.org/x/net/publicsuffix by using
// a simple heuristic for extracting the registrable base domain.
package domainutil

import "strings"

// ccTLDParts lists second-level labels that commonly appear under country-code
// TLDs, indicating that the registrable domain includes three labels instead
// of the usual two. For example, "example.co.uk" has base domain
// "example.co.uk", not "co.uk".
var ccTLDParts = map[string]bool{
	"co":  true,
	"com": true,
	"org": true,
	"net": true,
	"gov": true,
	"ac":  true,
	"edu": true,
}

// BaseDomain extracts the registrable base domain from a full domain string.
// It strips a leading "*." wildcard prefix, splits on dots, and returns the
// last two labels -- or last three if the second-to-last label is a common
// ccTLD part (co, com, org, net, gov, ac, edu).
//
// Examples:
//
//	"foo.bar.netflixconfirmation.net"  -> "netflixconfirmation.net"
//	"insightandsound.co.uk"           -> "insightandsound.co.uk"
//	"*.sub.example.com"               -> "example.com"
//	"example.com"                     -> "example.com"
//	""                                -> ""
func BaseDomain(domainName string) string {
	// Strip leading wildcard prefix.
	domainName = strings.TrimPrefix(domainName, "*.")

	// Remove trailing dot if present (FQDN notation).
	domainName = strings.TrimSuffix(domainName, ".")

	if domainName == "" {
		return ""
	}

	labels := strings.Split(domainName, ".")
	n := len(labels)

	if n <= 2 {
		return domainName
	}

	// Check if the second-to-last label is a common ccTLD part.
	secondToLast := strings.ToLower(labels[n-2])
	if ccTLDParts[secondToLast] && n >= 3 {
		return strings.Join(labels[n-3:], ".")
	}

	return strings.Join(labels[n-2:], ".")
}
