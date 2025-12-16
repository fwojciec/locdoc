//go:build integration

package rod_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fwojciec/locdoc/rod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetcher_Integration_HtmxDocs(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fetcher, err := rod.NewFetcher()
	require.NoError(t, err)
	defer fetcher.Close()

	// htmx.org is a documentation site that uses JS for some rendering
	html, err := fetcher.Fetch(ctx, "https://htmx.org/docs/")
	require.NoError(t, err)

	// Should contain actual content (not just loading indicators)
	assert.NotEmpty(t, html)
	assert.Contains(t, html, "htmx", "expected htmx content in page")
	t.Logf("Fetched %d bytes from htmx.org/docs/", len(html))
}

func TestFetcher_Integration_ReactDocs(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fetcher, err := rod.NewFetcher()
	require.NoError(t, err)
	defer fetcher.Close()

	// React docs is heavily JS-rendered
	html, err := fetcher.Fetch(ctx, "https://react.dev/learn")
	require.NoError(t, err)

	// Should contain actual React content
	assert.NotEmpty(t, html)
	assert.Contains(t, html, "React", "expected React content in page")
	t.Logf("Fetched %d bytes from react.dev/learn", len(html))
}

func TestFetcher_Close_KillsLauncherProcess(t *testing.T) {
	t.Parallel()

	fetcher, err := rod.NewFetcher()
	require.NoError(t, err)

	// Get the launcher PID before closing
	pid := fetcher.LauncherPID()
	require.NotZero(t, pid, "launcher PID should be set")

	// Verify process is running before close
	process, err := os.FindProcess(pid)
	require.NoError(t, err)
	require.NotNil(t, process)

	// Close should clean up all resources including launcher
	err = fetcher.Close()
	require.NoError(t, err)

	// Give the OS a moment to clean up the process
	time.Sleep(100 * time.Millisecond)

	// On Unix systems, sending signal 0 checks if process exists without affecting it
	// If the process doesn't exist, Signal(0) returns an error
	err = process.Signal(os.Signal(nil))
	assert.Error(t, err, "launcher process should be terminated after Close()")
}
