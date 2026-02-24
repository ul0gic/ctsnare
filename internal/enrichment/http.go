package enrichment

import (
	"fmt"
	"net/http"
)

// userAgent is the User-Agent string sent with liveness probes.
const userAgent = "ctsnare/1.0 (domain-liveness-check)"

// ProbeLiveness sends an HTTP HEAD request to determine if the domain has
// a live web server. It tries HTTPS first, falling back to HTTP on failure.
//
// isLive is true if any HTTP response is received (even 4xx/5xx), because
// that means the domain resolves and a server is listening.
//
// The httpClient should have a timeout configured (typically 5s) and a
// redirect policy limiting redirects (typically 3).
//
// The response body is never read (HEAD only).
func ProbeLiveness(httpClient *http.Client, domainName string) (statusCode int, isLive bool, err error) {
	// Try HTTPS first.
	code, err := doHEAD(httpClient, "https://"+domainName+"/")
	if err == nil {
		return code, true, nil
	}

	// HTTPS failed -- try plain HTTP as fallback.
	code, err = doHEAD(httpClient, "http://"+domainName+"/")
	if err == nil {
		return code, true, nil
	}

	return 0, false, fmt.Errorf("both HTTPS and HTTP probes failed for %s: %w", domainName, err)
}

// doHEAD sends an HTTP HEAD request to the given URL and returns the status code.
func doHEAD(client *http.Client, url string) (int, error) {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return 0, fmt.Errorf("creating HEAD request for %s: %w", url, err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()

	return resp.StatusCode, nil
}
