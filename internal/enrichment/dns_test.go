package enrichment

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveDomain_Localhost(t *testing.T) {
	ips, provider, err := ResolveDomain("localhost")
	require.NoError(t, err)
	assert.NotEmpty(t, ips, "localhost should resolve to at least one IP")

	// localhost typically resolves to 127.0.0.1 or ::1.
	found := false
	for _, ip := range ips {
		if ip == "127.0.0.1" || ip == "::1" {
			found = true
			break
		}
	}
	assert.True(t, found, "localhost should resolve to 127.0.0.1 or ::1, got %v", ips)
	assert.Equal(t, "unknown", provider, "localhost should have unknown provider")
}

func TestResolveDomain_NonexistentDomain(t *testing.T) {
	ips, provider, err := ResolveDomain("this-domain-definitely-does-not-exist-7291.invalid")
	assert.Error(t, err, "nonexistent domain should produce an error")
	assert.Nil(t, ips)
	assert.Equal(t, "unknown", provider)
}

func TestMatchCIDR_CloudflareIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{"Cloudflare IPv4", "104.16.0.1", "cloudflare"},
		{"Cloudflare IPv6", "2606:4700::1", "cloudflare"},
		{"Fastly IPv4", "151.101.1.1", "fastly"},
		{"Akamai IPv4", "23.1.2.3", "akamai"},
		{"DigitalOcean IPv4", "167.172.1.1", "digitalocean"},
		{"Unknown IP", "8.8.8.8", ""},
		{"Loopback", "127.0.0.1", ""},
		{"Private RFC1918", "192.168.1.1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchCIDR([]string{tt.ip})
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchCIDR_MultipleIPs(t *testing.T) {
	// First IP is unknown, second is Cloudflare -- should still detect.
	result := matchCIDR([]string{"8.8.8.8", "104.16.0.1"})
	assert.Equal(t, "cloudflare", result)
}

func TestMatchCIDR_InvalidIP(t *testing.T) {
	result := matchCIDR([]string{"not-an-ip"})
	assert.Empty(t, result, "invalid IP should return empty provider")
}

func TestMatchCIDR_EmptySlice(t *testing.T) {
	result := matchCIDR([]string{})
	assert.Empty(t, result)
}

func TestReverseDNSPatterns_Coverage(t *testing.T) {
	// Verify all patterns in the table are present and correctly mapped.
	expected := map[string]string{
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

	for pattern, provider := range expected {
		got, ok := reverseDNSPatterns[pattern]
		assert.True(t, ok, "pattern %q should exist in reverseDNSPatterns", pattern)
		assert.Equal(t, provider, got, "pattern %q should map to %q", pattern, provider)
	}
}

func TestParsedCIDRs_Init(t *testing.T) {
	// Verify init() parsed all CIDRs without error.
	assert.NotEmpty(t, parsedCIDRs, "parsedCIDRs should be populated at init time")

	for provider, nets := range parsedCIDRs {
		assert.NotEmpty(t, nets, "provider %q should have at least one parsed CIDR", provider)
		for _, ipNet := range nets {
			assert.NotNil(t, ipNet, "parsed CIDR should not be nil for provider %q", provider)
		}
	}
}

func TestMatchCIDR_AllProviderRanges(t *testing.T) {
	// For each provider, pick one IP inside its first range and verify detection.
	tests := []struct {
		provider string
		ip       string
	}{
		{"cloudflare", "104.16.0.1"},
		{"cloudflare", "172.64.0.1"},
		{"cloudflare", "131.0.72.1"},
		{"fastly", "151.101.0.1"},
		{"fastly", "199.232.0.1"},
		{"akamai", "23.0.0.1"},
		{"akamai", "104.64.0.1"},
		{"digitalocean", "167.172.0.1"},
		{"digitalocean", "164.90.0.1"},
		{"digitalocean", "143.198.0.1"},
		{"digitalocean", "137.184.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"_"+tt.ip, func(t *testing.T) {
			result := matchCIDR([]string{tt.ip})
			assert.Equal(t, tt.provider, result)
		})
	}
}

func TestKnownCIDRs_AllValid(t *testing.T) {
	// Verify every string in knownCIDRs is a valid CIDR.
	for provider, cidrs := range knownCIDRs {
		for _, cidr := range cidrs {
			_, _, err := net.ParseCIDR(cidr)
			assert.NoError(t, err, "CIDR %q for provider %q should be valid", cidr, provider)
		}
	}
}
