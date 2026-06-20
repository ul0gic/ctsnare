package enrichment

import (
	"context"
	"net"
	"strings"
	"time"
)

const dnsTimeout = 3 * time.Second

// knownCIDRs holds a representative subset of provider ranges, enough for
// common CDN/cloud detection.
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
				// Static-table typo is a programming error, not a runtime condition.
				panic("invalid CIDR in knownCIDRs: " + cidr + ": " + err.Error())
			}
			nets = append(nets, ipNet)
		}
		parsedCIDRs[provider] = nets
	}
}

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

// ResolveDomain looks up A/AAAA records and detects the provider via CIDR or
// reverse DNS; provider is "unknown" when undetected.
func ResolveDomain(ctx context.Context, domainName string) (ips []string, provider string, err error) {
	ctx, cancel := context.WithTimeout(ctx, dnsTimeout)
	defer cancel()

	addrs, err := net.DefaultResolver.LookupHost(ctx, domainName)
	if err != nil {
		return nil, "unknown", err
	}

	ips = addrs

	// CIDR match first -- faster and more reliable than a PTR lookup.
	if p := matchCIDR(addrs); p != "" {
		return ips, p, nil
	}

	if p := matchReverseDNS(ctx, addrs); p != "" {
		return ips, p, nil
	}

	return ips, "unknown", nil
}

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
