package main

import (
	"context"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
)

// ProbeFetcher probes a source URL to determine which fetcher to use.
// It fetches HTML using the HTTP fetcher, detects the framework,
// and returns the appropriate fetcher based on JS requirements.
//
// Decision flow:
//   - Known JS-required framework (GitBook) → Use Rod
//   - Known HTTP-only framework (Sphinx, MkDocs, etc.) → Use HTTP
//   - Unknown framework → Fetch with both, compare content
//   - HTTP fetch fails → Fall back to Rod
//
// Always returns a valid fetcher; never fails.
func ProbeFetcher(
	ctx context.Context,
	sourceURL string,
	httpFetcher locdoc.Fetcher,
	rodFetcher locdoc.Fetcher,
	prober locdoc.Prober,
	extractor locdoc.Extractor,
) locdoc.Fetcher {
	// Fetch HTML using HTTP fetcher for probing
	httpHTML, httpErr := httpFetcher.Fetch(ctx, sourceURL)
	if httpErr != nil {
		// HTTP failed, fall back to Rod
		return rodFetcher
	}

	// Detect the framework
	framework := prober.Detect(httpHTML)

	// Check if the framework requires JavaScript
	requiresJS, known := prober.RequiresJS(framework)

	if known {
		if requiresJS {
			return rodFetcher
		}
		return httpFetcher
	}

	// Unknown framework: fetch with Rod and compare content
	rodHTML, rodErr := rodFetcher.Fetch(ctx, sourceURL)
	if rodErr != nil {
		// Rod failed, use HTTP (best effort)
		return httpFetcher
	}

	if crawl.ContentDiffers(httpHTML, rodHTML, extractor) {
		return rodFetcher
	}

	return httpFetcher
}
