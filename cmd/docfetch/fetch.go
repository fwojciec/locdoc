package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
	"github.com/fwojciec/locdoc/fs"
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
	// Try sitemap discovery first
	urls, err := deps.Sitemaps.DiscoverURLs(deps.Ctx, c.URL, nil)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return err
	}

	// Print URLs from sitemap
	if len(urls) > 0 {
		for _, u := range urls {
			fmt.Fprintln(deps.Stdout, u)
		}
		return nil
	}

	// Fall back to recursive discovery
	if deps.Discoverer != nil {
		_, err = deps.Discoverer.DiscoverURLs(deps.Ctx, c.URL, nil,
			crawl.WithConcurrency(c.Concurrency),
			crawl.WithOnURL(func(url string) {
				fmt.Fprintln(deps.Stdout, url)
			}))
		if err != nil {
			fmt.Fprintf(deps.Stderr, "error: %s\n", locdoc.ErrorMessage(err))
			return err
		}
	}

	return nil
}

func (c *FetchCmd) runFetch(deps *Dependencies) error {
	// Final output directory
	outputDir := filepath.Join(c.Path, c.Name)

	// Temporary directory for atomic update
	tempDir := outputDir + ".tmp"

	// Clean up temp directory if it exists from a previous failed run
	_ = os.RemoveAll(tempDir)

	// Create fs.Writer for temporary directory
	writer := fs.NewWriter(tempDir)

	// Set the writer on the crawler
	deps.Crawler.Documents = writer

	// Create a fake project for the crawler (it needs a Project struct)
	project := &locdoc.Project{
		ID:        c.Name, // Use name as ID for file-based storage
		Name:      c.Name,
		SourceURL: c.URL,
	}

	// Apply concurrency if set
	if c.Concurrency > 0 {
		deps.Crawler.Concurrency = c.Concurrency
	}

	var total int

	progress := func(event crawl.ProgressEvent) {
		switch event.Type {
		case crawl.ProgressStarted:
			total = event.Total
			fmt.Fprintf(deps.Stdout, "Found %d URLs\n", event.Total)
		case crawl.ProgressCompleted:
			if total > 0 {
				fmt.Fprintf(deps.Stdout, "\r[%d/%d] %s",
					event.Completed, total, crawl.TruncateURL(event.URL, 40))
			} else {
				fmt.Fprintf(deps.Stdout, "\r[%d] %s",
					event.Completed, crawl.TruncateURL(event.URL, 40))
			}
		case crawl.ProgressFailed:
			fmt.Fprintf(deps.Stderr, "skip %s: %v\n", event.URL, event.Error)
			if total > 0 {
				fmt.Fprintf(deps.Stdout, "\r[%d/%d] %s",
					event.Completed, total, crawl.TruncateURL(event.URL, 40))
			} else {
				fmt.Fprintf(deps.Stdout, "\r[%d] %s",
					event.Completed, crawl.TruncateURL(event.URL, 40))
			}
		case crawl.ProgressFinished:
			// Clear progress line
			fmt.Fprintf(deps.Stdout, "\r%s\r", strings.Repeat(" ", 80))
		}
	}

	result, err := deps.Crawler.CrawlProject(deps.Ctx, project, progress)
	if err != nil {
		// Clean up temp directory on failure
		_ = os.RemoveAll(tempDir)
		fmt.Fprintf(deps.Stderr, "error crawling: %v\n", err)
		return err
	}

	// Atomic update: only replace if we saved at least one page
	if result.Saved > 0 {
		// Remove existing directory
		_ = os.RemoveAll(outputDir)

		// Rename temp to final
		if err := os.Rename(tempDir, outputDir); err != nil {
			// Clean up temp directory on rename failure
			_ = os.RemoveAll(tempDir)
			return fmt.Errorf("failed to rename temp directory: %w", err)
		}
	} else {
		// No pages saved, clean up temp directory
		_ = os.RemoveAll(tempDir)
	}

	if result.Failed > 0 {
		fmt.Fprintf(deps.Stdout, "Saved %d pages (%d failed, %s)\n",
			result.Saved, result.Failed, crawl.FormatBytes(result.Bytes))
	} else {
		fmt.Fprintf(deps.Stdout, "Saved %d pages (%s)\n",
			result.Saved, crawl.FormatBytes(result.Bytes))
	}

	return nil
}
