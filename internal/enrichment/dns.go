package enrichment

import (
	"context"
	"net"
	"strings"
	"time"
)

// dnsTimeout is the maximum time allowed for DNS resolution.
const dnsTimeout = 3 * time.Second

// knownCIDRs maps hosting provider names to their known IP CIDR ranges.
// Only a representative subset is included -- enough for common CDN/cloud detection.
var knownCIDRs = map[string][]string{
	"cloudflare": {
		"104.16.0.0/12",
		"172.64.0.0/13",
		"131.0.72.0/22",
		"2606:4700::/32",
	},
	"fastly": {
		"151.101.0.0/16",
		"199.232.0.0/16",
	},
	"akamai": {
		"23.0.0.0/12",
		"104.64.0.0/10",
	},
	"digitalocean": {
		"167.172.0.0/16",
		"164.90.0.0/16",
		"143.198.0.0/16",
		"137.184.0.0/16",
	},
}

// parsedCIDRs is the compiled form of knownCIDRs, built at init time.
var parsedCIDRs map[string][]*net.IPNet

func init() {
	parsedCIDRs = make(map[string][]*net.IPNet, len(knownCIDRs))
	for provider, cidrs := range knownCIDRs {
		nets := make([]*net.IPNet, 0, len(cidrs))
		for _, cidr := range cidrs {
			_, ipNet, err := net.ParseCIDR(cidr)
			if err != nil {
				// Programming error in the static table -- panic at init.
				panic("invalid CIDR in knownCIDRs: " + cidr + ": " + err.Error())
			}
			nets = append(nets, ipNet)
		}
		parsedCIDRs[provider] = nets
	}
}

// reverseDNSPatterns maps substrings in reverse DNS names to provider names.
var reverseDNSPatterns = map[string]string{
	"cloudflare":        "cloudflare",
	"amazonaws.com":     "aws",
	"googleusercontent": "gcp",
	"1e100.net":         "gcp",
	"azure.com":         "azure",
	"msedge.net":        "azure",
	"fastly":            "fastly",
	"akamai":            "akamai",
	"digitalocean.com":  "digitalocean",
}

// ResolveDomain performs DNS A/AAAA lookups and attempts to identify the
// hosting provider via CIDR range matching or reverse DNS. Returns the
// resolved IP addresses, detected provider name ("unknown" if undetected),
// and any error from the resolution.
func ResolveDomain(domainName string) (ips []string, provider string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), dnsTimeout)
	defer cancel()

	addrs, err := net.DefaultResolver.LookupHost(ctx, domainName)
	if err != nil {
		return nil, "unknown", err
	}

	ips = addrs

	// Try CIDR matching first -- faster and more reliable.
	if p := matchCIDR(addrs); p != "" {
		return ips, p, nil
	}

	// Fall back to reverse DNS lookup on the first IP.
	if p := matchReverseDNS(ctx, addrs); p != "" {
		return ips, p, nil
	}

	return ips, "unknown", nil
}

// matchCIDR checks each resolved IP against known provider CIDR ranges.
func matchCIDR(addrs []string) string {
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		for provider, nets := range parsedCIDRs {
			for _, ipNet := range nets {
				if ipNet.Contains(ip) {
					return provider
				}
			}
		}
	}
	return ""
}

// matchReverseDNS does a PTR lookup on the first resolvable IP and checks
// the result against known reverse DNS patterns.
func matchReverseDNS(ctx context.Context, addrs []string) string {
	for _, addr := range addrs {
		names, err := net.DefaultResolver.LookupAddr(ctx, addr)
		if err != nil || len(names) == 0 {
			continue
		}
		for _, name := range names {
			lower := strings.ToLower(name)
			for pattern, provider := range reverseDNSPatterns {
				if strings.Contains(lower, pattern) {
					return provider
				}
			}
		}
		// Only check the first IP that resolves to avoid slow cascading lookups.
		break
	}
	return ""
}
