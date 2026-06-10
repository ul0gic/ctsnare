package profile

import "strings"

// CategoryKeywords returns a lowercased keyword -> category map built from the
// three built-in profiles (crypto, phishing, ai). It lets the scorer attribute
// a hit to the profile bucket that produced its strongest match, independent of
// which profile (often "all") was active. When a keyword appears in more than
// one built-in, the first registered profile wins; this is rare and the
// attribution is a best-effort hint, not an authoritative classification.
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
