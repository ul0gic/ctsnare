package enrichment

import (
	"context"
	"fmt"
	"net/http"
)

const userAgent = "ctsnare/1.0 (domain-liveness-check)"

// ProbeLiveness HEAD-probes the domain, HTTPS then HTTP. isLive is true for any
// response (even 4xx/5xx): a server is listening.
func ProbeLiveness(ctx context.Context, httpClient *http.Client, domainName string) (statusCode int, isLive bool, err error) {
	code, err := doHEAD(ctx, httpClient, "https://"+domainName+"/")
	if err == nil {
		return code, true, nil
	}

	code, err = doHEAD(ctx, httpClient, "http://"+domainName+"/")
	if err == nil {
		return code, true, nil
	}

	return 0, false, fmt.Errorf("both HTTPS and HTTP probes failed for %s: %w", domainName, err)
}

func doHEAD(ctx context.Context, client *http.Client, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
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
