package profile

import "strings"

// CategoryKeywords maps lowercased keyword -> built-in bucket for best-effort hit
// attribution; on overlap the first registered profile (crypto, then phishing, then ai) wins.
func CategoryKeywords() map[string]string {
	m := make(map[string]string)
	register := func(category string, keywords ...[]string) {
		for _, list := range keywords {
			for _, kw := range list {
				k := strings.ToLower(kw)
				if _, exists := m[k]; !exists {
					m[k] = category
				}
			}
		}
	}
	register("crypto", CryptoProfile.BrandKeywords, CryptoProfile.Keywords)
	register("phishing", PhishingProfile.BrandKeywords, PhishingProfile.Keywords)
	register("ai", AIProfile.BrandKeywords, AIProfile.Keywords)
	return m
}
