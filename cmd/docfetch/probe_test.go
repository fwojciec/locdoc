package main_test

import (
	"context"
	"testing"

	"github.com/fwojciec/locdoc"
	main "github.com/fwojciec/locdoc/cmd/docfetch"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Story: Probing selects the right fetcher based on framework requirements
//
// When fetching documentation, we need to determine whether the site requires
// JavaScript rendering. ProbeFetcher fetches a sample page, detects the framework,
// and returns the appropriate fetcher.

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

		// When: probing the URL
		result, err := main.ProbeFetcher(
			context.Background(),
			"https://example.com/docs",
			httpFetcher,
			rodFetcher,
			prober,
		)

		// Then: HTTP fetcher is returned
		require.NoError(t, err)
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

		// When: probing the URL
		result, err := main.ProbeFetcher(
			context.Background(),
			"https://example.com/docs",
			httpFetcher,
			rodFetcher,
			prober,
		)

		// Then: rod fetcher is returned
		require.NoError(t, err)
		assert.Same(t, rodFetcher, result)
	})

	t.Run("returns HTTP fetcher for unknown frameworks", func(t *testing.T) {
		t.Parallel()

		// Given: an HTTP fetcher returns HTML
		httpFetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "<html><body>Unknown framework</body></html>", nil
			},
		}

		rodFetcher := &mock.Fetcher{}

		// Given: prober doesn't recognize the framework
		prober := &mock.Prober{
			DetectFn: func(html string) locdoc.Framework {
				return locdoc.FrameworkUnknown
			},
			RequiresJSFn: func(framework locdoc.Framework) (bool, bool) {
				return false, false // Unknown framework
			},
		}

		// When: probing the URL
		result, err := main.ProbeFetcher(
			context.Background(),
			"https://example.com/docs",
			httpFetcher,
			rodFetcher,
			prober,
		)

		// Then: HTTP fetcher is returned (default for unknown)
		require.NoError(t, err)
		assert.Same(t, httpFetcher, result)
	})

	t.Run("returns error when probe fetch fails", func(t *testing.T) {
		t.Parallel()

		// Given: HTTP fetcher fails
		httpFetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "", locdoc.Errorf(locdoc.EINTERNAL, "network error")
			},
		}

		rodFetcher := &mock.Fetcher{}
		prober := &mock.Prober{}

		// When: probing the URL
		result, err := main.ProbeFetcher(
			context.Background(),
			"https://example.com/docs",
			httpFetcher,
			rodFetcher,
			prober,
		)

		// Then: error is returned
		require.Error(t, err)
		assert.Nil(t, result)
	})
}
