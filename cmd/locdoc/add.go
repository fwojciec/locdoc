package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
)

// Run executes the add command.
func (c *AddCmd) Run(deps *Dependencies) error {
	// Compile filters to URLFilter (validates regex patterns early)
	var urlFilter *locdoc.URLFilter
	if len(c.Filter) > 0 {
		urlFilter = &locdoc.URLFilter{}
		for _, pattern := range c.Filter {
			re, err := regexp.Compile(pattern)
			if err != nil {
				fmt.Fprintf(deps.Stderr, "error: invalid regex filter pattern %q: %v\n", pattern, err)
				fmt.Fprintln(deps.Stderr, "Filter patterns use Go regex syntax. Example patterns:")
				fmt.Fprintln(deps.Stderr, "  /api/       - match URLs containing '/api/'")
				fmt.Fprintln(deps.Stderr, "  ^https://   - match URLs starting with 'https://'")
				fmt.Fprintln(deps.Stderr, "  \\.md$       - match URLs ending with '.md'")
				return err
			}
			urlFilter.Include = append(urlFilter.Include, re)
		}
	}

	// Preview mode: show URLs without creating project
	if c.Preview {
		urls, err := deps.Sitemaps.DiscoverURLs(deps.Ctx, c.URL, urlFilter)
		if err != nil {
			fmt.Fprintf(deps.Stderr, "error: %s\n", locdoc.ErrorMessage(err))
			return err
		}

		// Fall back to recursive discovery if sitemap returns no URLs
		if len(urls) == 0 && deps.Fetcher != nil && deps.LinkSelectors != nil && deps.RateLimiter != nil {
			urls, err = crawl.DiscoverURLs(deps.Ctx, c.URL, urlFilter, deps.Fetcher, deps.LinkSelectors, deps.RateLimiter,
				crawl.WithConcurrency(c.Concurrency))
			if err != nil {
				fmt.Fprintf(deps.Stderr, "error: %s\n", locdoc.ErrorMessage(err))
				return err
			}
		}

		for _, u := range urls {
			fmt.Fprintln(deps.Stdout, u)
		}
		return nil
	}

	// Force mode: delete existing project first
	if c.Force {
		existing, err := deps.Projects.FindProjects(deps.Ctx, locdoc.ProjectFilter{Name: &c.Name})
		if err != nil {
			fmt.Fprintf(deps.Stderr, "error: %s\n", locdoc.ErrorMessage(err))
			return err
		}
		if len(existing) > 0 {
			if err := deps.Projects.DeleteProject(deps.Ctx, existing[0].ID); err != nil {
				fmt.Fprintf(deps.Stderr, "error: %s\n", locdoc.ErrorMessage(err))
				return err
			}
		}
	}

	// Create project
	project := &locdoc.Project{
		Name:      c.Name,
		SourceURL: c.URL,
		Filter:    strings.Join(c.Filter, "\n"),
	}

	if err := deps.Projects.CreateProject(deps.Ctx, project); err != nil {
		fmt.Fprintf(deps.Stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return err
	}

	fmt.Fprintf(deps.Stdout, "Added project %q (%s)\n", c.Name, project.ID)

	// Crawl documents if Crawler is provided
	if deps.Crawler != nil {
		// Apply user-specified concurrency
		if c.Concurrency > 0 {
			deps.Crawler.Concurrency = c.Concurrency
		}

		var total int

		progress := func(event crawl.ProgressEvent) {
			switch event.Type {
			case crawl.ProgressStarted:
				total = event.Total
				fmt.Fprintf(deps.Stdout, "  Found %d URLs\n", event.Total)
			case crawl.ProgressCompleted:
				// Update progress line in place
				// Show [N/M] when total is known, [N] when total is unknown (recursive crawl)
				if total > 0 {
					fmt.Fprintf(deps.Stdout, "\r  [%d/%d] %s",
						event.Completed, total, crawl.TruncateURL(event.URL, 40))
				} else {
					fmt.Fprintf(deps.Stdout, "\r  [%d] %s",
						event.Completed, crawl.TruncateURL(event.URL, 40))
				}
			case crawl.ProgressFailed:
				// Print failure on its own line (persists in scroll history)
				fmt.Fprintf(deps.Stderr, "  skip %s: %v\n", event.URL, event.Error)
				// Update progress line after failure message
				if total > 0 {
					fmt.Fprintf(deps.Stdout, "\r  [%d/%d] %s",
						event.Completed, total, crawl.TruncateURL(event.URL, 40))
				} else {
					fmt.Fprintf(deps.Stdout, "\r  [%d] %s",
						event.Completed, crawl.TruncateURL(event.URL, 40))
				}
			case crawl.ProgressFinished:
				// Clear progress line
				fmt.Fprintf(deps.Stdout, "\r%s\r", strings.Repeat(" ", 80))
			}
		}

		result, err := deps.Crawler.CrawlProject(deps.Ctx, project, progress)
		if err != nil {
			fmt.Fprintf(deps.Stderr, "error crawling: %v\n", err)
			return err
		}

		fmt.Fprintf(deps.Stdout, "  Saved %d pages (%s, %s)\n",
			result.Saved, crawl.FormatBytes(result.Bytes), crawl.FormatTokens(result.Tokens))
	}

	return nil
}
