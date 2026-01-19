package main

import (
	"fmt"
	"net/url"

	"github.com/fwojciec/locdoc"
)

// Run executes the fetch command.
func (c *FetchCmd) Run(deps *Dependencies) error {
	// Preview mode: show URLs without creating files
	if c.Preview {
		return c.runPreview(deps)
	}

	// Full fetch mode
	return c.runFetch(deps)
}

func (c *FetchCmd) runPreview(deps *Dependencies) error {
	urls, err := deps.Source.Discover(deps.Ctx, c.URL)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return err
	}

	for _, u := range urls {
		fmt.Fprintln(deps.Stdout, u)
	}

	return nil
}

func (c *FetchCmd) runFetch(deps *Dependencies) error {
	// Discover URLs
	urls, err := deps.Source.Discover(deps.Ctx, c.URL)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return err
	}

	fmt.Fprintf(deps.Stdout, "Found %d URLs\n", len(urls))

	// Fetch pages with progress reporting
	progress := func(p locdoc.FetchProgress) {
		if p.Error != nil {
			fmt.Fprintf(deps.Stderr, "skip %s: %v\n", p.URL, p.Error)
		}
		fmt.Fprintf(deps.Stdout, "\r[%d/%d] %s", p.Completed, p.Total, truncateURL(p.URL, 40))
	}

	pages, err := deps.Fetcher.FetchAll(deps.Ctx, urls, progress)
	if err != nil {
		_ = deps.Store.Abort()
		fmt.Fprintf(deps.Stderr, "error fetching: %v\n", err)
		return err
	}

	// Clear progress line
	fmt.Fprintf(deps.Stdout, "\r%80s\r", "")

	// Save pages
	for _, page := range pages {
		if err := deps.Store.Save(deps.Ctx, page); err != nil {
			_ = deps.Store.Abort()
			fmt.Fprintf(deps.Stderr, "error saving %s: %v\n", page.URL, err)
			return err
		}
	}

	// Commit or abort based on success
	if len(pages) > 0 {
		if err := deps.Store.Commit(); err != nil {
			fmt.Fprintf(deps.Stderr, "error committing: %v\n", err)
			return err
		}
		fmt.Fprintf(deps.Stdout, "Saved %d pages\n", len(pages))
	} else {
		_ = deps.Store.Abort()
		fmt.Fprintln(deps.Stdout, "No pages saved")
	}

	return nil
}

// truncateURL shortens a URL for display by showing only the path.
// This makes progress more useful when many URLs share the same host prefix.
func truncateURL(rawURL string, maxLen int) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		// Fallback to simple right-truncation
		if len(rawURL) <= maxLen {
			return rawURL
		}
		return rawURL[:maxLen-3] + "..."
	}

	path := parsed.Path
	if path == "" {
		path = "/"
	}

	if len(path) <= maxLen {
		return path
	}

	// Truncate from the left to show the unique suffix
	return "..." + path[len(path)-maxLen+3:]
}
