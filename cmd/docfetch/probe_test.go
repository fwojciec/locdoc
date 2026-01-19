package main_test

import (
	"context"
	"testing"

	"github.com/fwojciec/locdoc"
	main "github.com/fwojciec/locdoc/cmd/docfetch"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
)

// Story: Probing selects the right fetcher based on framework requirements
//
// When fetching documentation, we need to determine whether the site requires
// JavaScript rendering. ProbeFetcher fetches a sample page, detects the framework,
// and returns the appropriate fetcher.
//
// Decision flow:
//   - Known JS-required framework (GitBook) -> Use Rod
//   - Known HTTP-only framework (Sphinx, MkDocs, etc.) -> Use HTTP
//   - Unknown framework -> Fetch with both, compare content
//   - HTTP fetch fails -> Fall back to Rod

func TestProbeFetcher(t *testing.T) {
	t.Parallel()

	t.Run("returns HTTP fetcher when framework does not require JS", func(t *testing.T) {
		t.Parallel()

		// Given: an HTTP fetcher returns HTML
		httpFetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "<html><body>Static content</body></html>", nil
			},
		}

		// Given: a rod fetcher (unused in this case)
		rodFetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "<html><body>JS rendered</body></html>", nil
			},
		}

		// Given: prober detects MkDocs (static framework)
		prober := &mock.Prober{
			DetectFn: func(html string) locdoc.Framework {
				return locdoc.FrameworkMkDocs
			},
			RequiresJSFn: func(framework locdoc.Framework) (bool, bool) {
				return false, true // MkDocs doesn't require JS
			},
		}

		// Given: extractor (unused for known frameworks)
		extractor := &mock.Extractor{}

		// When: probing the URL
		result := main.ProbeFetcher(
			context.Background(),
			"https://example.com/docs",
			httpFetcher,
			rodFetcher,
			prober,
			extractor,
		)

		// Then: HTTP fetcher is returned
		assert.Same(t, httpFetcher, result)
	})

	t.Run("returns rod fetcher when framework requires JS", func(t *testing.T) {
		t.Parallel()

		// Given: an HTTP fetcher returns HTML
		httpFetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "<html><body>Loading...</body></html>", nil
			},
		}

		// Given: a rod fetcher for JS-heavy sites
		rodFetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "<html><body>JS rendered content</body></html>", nil
			},
		}

		// Given: prober detects GitBook (requires JS)
		prober := &mock.Prober{
			DetectFn: func(html string) locdoc.Framework {
				return locdoc.FrameworkGitBook
			},
			RequiresJSFn: func(framework locdoc.Framework) (bool, bool) {
				return true, true // GitBook requires JS
			},
		}

		// Given: extractor (unused for known frameworks)
		extractor := &mock.Extractor{}

		// When: probing the URL
		result := main.ProbeFetcher(
			context.Background(),
			"https://example.com/docs",
			httpFetcher,
			rodFetcher,
			prober,
			extractor,
		)

		// Then: rod fetcher is returned
		assert.Same(t, rodFetcher, result)
	})

	t.Run("returns rod fetcher for unknown frameworks when content differs", func(t *testing.T) {
		t.Parallel()

		// Given: HTTP fetcher returns shell HTML (no content)
		httpFetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "<html><body><div id='root'></div></body></html>", nil
			},
		}

		// Given: rod fetcher returns JS-rendered content
		rodFetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "<html><body><div id='root'><article>Lots of documentation content here...</article></div></body></html>", nil
			},
		}

		// Given: prober doesn't recognize the framework
		prober := &mock.Prober{
			DetectFn: func(html string) locdoc.Framework {
				return locdoc.FrameworkUnknown
			},
			RequiresJSFn: func(framework locdoc.Framework) (bool, bool) {
				return false, false // Unknown framework
			},
		}

		// Given: extractor returns empty for HTTP, content for rod
		extractCalls := 0
		extractor := &mock.Extractor{
			ExtractFn: func(rawHTML string) (*locdoc.ExtractResult, error) {
				extractCalls++
				if extractCalls == 1 {
					// HTTP HTML - empty content
					return &locdoc.ExtractResult{ContentHTML: ""}, nil
				}
				// Rod HTML - has content
				return &locdoc.ExtractResult{ContentHTML: "<article>Lots of documentation content here...</article>"}, nil
			},
		}

		// When: probing the URL
		result := main.ProbeFetcher(
			context.Background(),
			"https://example.com/docs",
			httpFetcher,
			rodFetcher,
			prober,
			extractor,
		)

		// Then: rod fetcher is returned (content differs)
		assert.Same(t, rodFetcher, result)
	})

	t.Run("returns HTTP fetcher for unknown frameworks when content is similar", func(t *testing.T) {
		t.Parallel()

		// Given: both fetchers return similar content
		httpFetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "<html><body><article>Static docs</article></body></html>", nil
			},
		}

		rodFetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "<html><body><article>Static docs</article></body></html>", nil
			},
		}

		// Given: prober doesn't recognize the framework
		prober := &mock.Prober{
			DetectFn: func(html string) locdoc.Framework {
				return locdoc.FrameworkUnknown
			},
			RequiresJSFn: func(framework locdoc.Framework) (bool, bool) {
				return false, false // Unknown framework
			},
		}

		// Given: extractor returns same content for both
		extractor := &mock.Extractor{
			ExtractFn: func(rawHTML string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{ContentHTML: "<article>Static docs</article>"}, nil
			},
		}

		// When: probing the URL
		result := main.ProbeFetcher(
			context.Background(),
			"https://example.com/docs",
			httpFetcher,
			rodFetcher,
			prober,
			extractor,
		)

		// Then: HTTP fetcher is returned (content similar, use faster option)
		assert.Same(t, httpFetcher, result)
	})

	t.Run("falls back to rod fetcher when HTTP fetch fails", func(t *testing.T) {
		t.Parallel()

		// Given: HTTP fetcher fails
		httpFetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "", locdoc.Errorf(locdoc.EINTERNAL, "network error")
			},
		}

		// Given: rod fetcher is available
		rodFetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "<html><body>Content</body></html>", nil
			},
		}

		prober := &mock.Prober{}
		extractor := &mock.Extractor{}

		// When: probing the URL
		result := main.ProbeFetcher(
			context.Background(),
			"https://example.com/docs",
			httpFetcher,
			rodFetcher,
			prober,
			extractor,
		)

		// Then: rod fetcher is returned as fallback
		assert.Same(t, rodFetcher, result)
	})

	t.Run("returns HTTP fetcher when rod fails for unknown framework", func(t *testing.T) {
		t.Parallel()

		// Given: HTTP fetcher works
		httpFetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "<html><body>Static content</body></html>", nil
			},
		}

		// Given: rod fetcher fails
		rodFetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "", locdoc.Errorf(locdoc.EINTERNAL, "browser error")
			},
		}

		// Given: prober doesn't recognize the framework
		prober := &mock.Prober{
			DetectFn: func(html string) locdoc.Framework {
				return locdoc.FrameworkUnknown
			},
			RequiresJSFn: func(framework locdoc.Framework) (bool, bool) {
				return false, false // Unknown framework
			},
		}

		extractor := &mock.Extractor{}

		// When: probing the URL
		result := main.ProbeFetcher(
			context.Background(),
			"https://example.com/docs",
			httpFetcher,
			rodFetcher,
			prober,
			extractor,
		)

		// Then: HTTP fetcher is returned (rod failed, best effort with HTTP)
		assert.Same(t, httpFetcher, result)
	})
}
