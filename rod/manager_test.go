//go:build integration

package rod_test

import (
	"testing"

	"github.com/fwojciec/locdoc/rod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowserManager_RecyclesBrowserAfterMaxPages(t *testing.T) {
	t.Parallel()

	// Create manager that recycles after 3 pages
	manager, err := rod.NewBrowserManager(rod.WithMaxPages(3))
	require.NoError(t, err)
	defer manager.Close()

	// Get first browser and record its identity
	firstBrowser := manager.Browser()
	require.NotNil(t, firstBrowser)

	// Increment page count 3 times (reaches max)
	manager.IncrementPageCount()
	manager.IncrementPageCount()
	manager.IncrementPageCount()

	// Next Browser() call should recycle and return a different instance
	secondBrowser := manager.Browser()
	require.NotNil(t, secondBrowser)

	// The browsers should be different instances (recycled)
	assert.NotSame(t, firstBrowser, secondBrowser)
}

func TestBrowserManager_DoesNotRecycleBeforeMaxPages(t *testing.T) {
	t.Parallel()

	manager, err := rod.NewBrowserManager(rod.WithMaxPages(5))
	require.NoError(t, err)
	defer manager.Close()

	firstBrowser := manager.Browser()
	require.NotNil(t, firstBrowser)

	// Increment page count but stay below max
	manager.IncrementPageCount()
	manager.IncrementPageCount()

	// Should still be the same browser
	sameBrowser := manager.Browser()
	assert.Same(t, firstBrowser, sameBrowser)
}
