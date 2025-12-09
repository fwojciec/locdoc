package locdoc

import (
	"context"
	"regexp"
)

// SitemapService discovers URLs from website sitemaps.
type SitemapService interface {
	// DiscoverURLs finds all URLs from a site's sitemap.
	// It first checks robots.txt for sitemap directives, then falls back
	// to /sitemap.xml. Sitemap indexes are resolved recursively.
	//
	// The filter can be used to include/exclude URLs by pattern.
	// If filter is nil, all URLs are returned.
	DiscoverURLs(ctx context.Context, baseURL string, filter *URLFilter) ([]string, error)
}

// URLFilter specifies patterns for including/excluding URLs.
type URLFilter struct {
	// Include patterns - if set, only URLs matching at least one pattern are included.
	Include []*regexp.Regexp

	// Exclude patterns - URLs matching any pattern are excluded.
	// Exclude is applied after Include.
	Exclude []*regexp.Regexp
}

// Match returns true if the URL passes the filter.
// If the filter is nil, all URLs pass.
func (f *URLFilter) Match(url string) bool {
	if f == nil {
		return true
	}

	// If include patterns exist, URL must match at least one
	if len(f.Include) > 0 {
		matched := false
		for _, re := range f.Include {
			if re.MatchString(url) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check exclude patterns
	for _, re := range f.Exclude {
		if re.MatchString(url) {
			return false
		}
	}

	return true
}
