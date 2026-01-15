package main_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/fwojciec/locdoc"
	main "github.com/fwojciec/locdoc/cmd/docfetch"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Story: Preview Mode
//
// Preview mode allows users to see what URLs would be fetched without
// actually downloading or storing any content. It only uses the URLSource
// interface.

func TestPreview_ShowsURLsFromSource(t *testing.T) {
	t.Parallel()

	// Given: a URL source that returns discovered URLs
	source := &mock.URLSource{
		DiscoverFn: func(_ context.Context, sourceURL string) ([]string, error) {
			return []string{
				"https://example.com/docs/page1",
				"https://example.com/docs/page2",
			}, nil
		},
	}

	stdout := &bytes.Buffer{}
	deps := &main.Dependencies{
		Ctx:    context.Background(),
		Stdout: stdout,
		Stderr: &bytes.Buffer{},
		Source: source,
		// Fetcher and Store not needed for preview
	}

	cmd := &main.FetchCmd{
		URL:     "https://example.com/docs",
		Preview: true,
	}

	// When: running in preview mode
	err := cmd.Run(deps)

	// Then: URLs are printed without fetching or storing
	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "https://example.com/docs/page1")
	assert.Contains(t, output, "https://example.com/docs/page2")
}

func TestPreview_ReportsDiscoveryErrors(t *testing.T) {
	t.Parallel()

	// Given: a source that fails to discover URLs
	source := &mock.URLSource{
		DiscoverFn: func(_ context.Context, sourceURL string) ([]string, error) {
			return nil, locdoc.Errorf(locdoc.EINTERNAL, "discovery failed")
		},
	}

	deps := &main.Dependencies{
		Ctx:    context.Background(),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Source: source,
	}

	cmd := &main.FetchCmd{
		URL:     "https://example.com/docs",
		Preview: true,
	}

	// When: running preview with failing source
	err := cmd.Run(deps)

	// Then: error is returned
	require.Error(t, err)
	assert.Contains(t, err.Error(), "discovery failed")
}
