package main

import (
	"context"

	"github.com/fwojciec/locdoc"
)

// ProbeFetcher probes a source URL to determine which fetcher to use.
// It fetches HTML using the HTTP fetcher, detects the framework,
// and returns the appropriate fetcher based on JS requirements.
func ProbeFetcher(
	ctx context.Context,
	sourceURL string,
	httpFetcher locdoc.Fetcher,
	rodFetcher locdoc.Fetcher,
	prober locdoc.Prober,
) (locdoc.Fetcher, error) {
	// Fetch HTML using HTTP fetcher for probing
	html, err := httpFetcher.Fetch(ctx, sourceURL)
	if err != nil {
		return nil, err
	}

	// Detect the framework
	framework := prober.Detect(html)

	// Check if the framework requires JavaScript
	requiresJS, _ := prober.RequiresJS(framework)

	if requiresJS {
		return rodFetcher, nil
	}

	return httpFetcher, nil
}
