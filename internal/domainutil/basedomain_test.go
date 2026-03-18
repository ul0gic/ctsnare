package domainutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseDomain(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "simple two-label domain",
			input:  "example.com",
			expect: "example.com",
		},
		{
			name:   "subdomain stripped to base",
			input:  "foo.bar.netflixconfirmation.net",
			expect: "netflixconfirmation.net",
		},
		{
			name:   "deep subdomain stripped to base",
			input:  "kupdate.cbaupdate.yxwupdate.netflixconfirmation.net",
			expect: "netflixconfirmation.net",
		},
		{
			name:   "ccTLD co.uk keeps three labels",
			input:  "insightandsound.co.uk",
			expect: "insightandsound.co.uk",
		},
		{
			name:   "subdomain under co.uk keeps three labels",
			input:  "www.insightandsound.co.uk",
			expect: "insightandsound.co.uk",
		},
		{
			name:   "ccTLD com.au keeps three labels",
			input:  "example.com.au",
			expect: "example.com.au",
		},
		{
			name:   "ccTLD org.uk keeps three labels",
			input:  "example.org.uk",
			expect: "example.org.uk",
		},
		{
			name:   "ccTLD gov.uk keeps three labels",
			input:  "example.gov.uk",
			expect: "example.gov.uk",
		},
		{
			name:   "ccTLD ac.uk keeps three labels",
			input:  "example.ac.uk",
			expect: "example.ac.uk",
		},
		{
			name:   "ccTLD edu.au keeps three labels",
			input:  "example.edu.au",
			expect: "example.edu.au",
		},
		{
			name:   "wildcard prefix stripped",
			input:  "*.sub.example.com",
			expect: "example.com",
		},
		{
			name:   "wildcard on two-label domain",
			input:  "*.example.com",
			expect: "example.com",
		},
		{
			name:   "single label returns as-is",
			input:  "localhost",
			expect: "localhost",
		},
		{
			name:   "empty string returns empty",
			input:  "",
			expect: "",
		},
		{
			name:   "trailing dot stripped (FQDN)",
			input:  "example.com.",
			expect: "example.com",
		},
		{
			name:   "wildcard with trailing dot",
			input:  "*.example.com.",
			expect: "example.com",
		},
		{
			name:   "three labels no ccTLD part returns two",
			input:  "sub.example.xyz",
			expect: "example.xyz",
		},
		{
			name:   "ccTLD net.au keeps three labels",
			input:  "sub.example.net.au",
			expect: "example.net.au",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BaseDomain(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}
