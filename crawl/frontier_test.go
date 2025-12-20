package crawl_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
	"github.com/stretchr/testify/assert"
)

func TestFrontier_Push_rejects_duplicate_URLs(t *testing.T) {
	t.Parallel()

	f := crawl.NewFrontier(1000, 0.01)

	link := locdoc.DiscoveredLink{
		URL:      "https://example.com/docs/page1",
		Priority: locdoc.PriorityNavigation,
	}

	// First push should succeed
	ok := f.Push(link)
	assert.True(t, ok, "first push should succeed")

	// Second push of same URL should be rejected
	ok = f.Push(link)
	assert.False(t, ok, "duplicate URL should be rejected")
}

func TestFrontier_Push_deduplicates_URLs_with_different_fragments(t *testing.T) {
	t.Parallel()

	f := crawl.NewFrontier(1000, 0.01)

	// First push without fragment should succeed
	ok := f.Push(locdoc.DiscoveredLink{
		URL:      "https://example.com/docs/overview",
		Priority: locdoc.PriorityNavigation,
	})
	assert.True(t, ok, "first push should succeed")

	// Push of same URL with fragment should be rejected as duplicate
	ok = f.Push(locdoc.DiscoveredLink{
		URL:      "https://example.com/docs/overview#motivation",
		Priority: locdoc.PriorityNavigation,
	})
	assert.False(t, ok, "URL with fragment should be rejected as duplicate of base URL")

	// Push of same URL with different fragment should also be rejected
	ok = f.Push(locdoc.DiscoveredLink{
		URL:      "https://example.com/docs/overview#getting-started",
		Priority: locdoc.PriorityNavigation,
	})
	assert.False(t, ok, "URL with different fragment should be rejected as duplicate")

	// Queue should only have one URL
	assert.Equal(t, 1, f.Len(), "should have exactly one URL in queue")

	// The URL in the queue should be the base URL without fragment
	link, ok := f.Pop()
	assert.True(t, ok)
	assert.Equal(t, "https://example.com/docs/overview", link.URL, "URL should be stored without fragment")
}

func TestFrontier_Push_strips_fragment_from_first_URL(t *testing.T) {
	t.Parallel()

	f := crawl.NewFrontier(1000, 0.01)

	// First push with fragment should succeed but store without fragment
	ok := f.Push(locdoc.DiscoveredLink{
		URL:      "https://example.com/docs/overview#section",
		Priority: locdoc.PriorityNavigation,
	})
	assert.True(t, ok, "first push should succeed")

	// The URL in the queue should be without fragment
	link, ok := f.Pop()
	assert.True(t, ok)
	assert.Equal(t, "https://example.com/docs/overview", link.URL, "URL should be stored without fragment")
}

func TestFrontier_Pop_returns_highest_priority_first(t *testing.T) {
	t.Parallel()

	f := crawl.NewFrontier(1000, 0.01)

	// Push links in random priority order
	f.Push(locdoc.DiscoveredLink{URL: "https://example.com/footer", Priority: locdoc.PriorityFooter})
	f.Push(locdoc.DiscoveredLink{URL: "https://example.com/nav", Priority: locdoc.PriorityNavigation})
	f.Push(locdoc.DiscoveredLink{URL: "https://example.com/content", Priority: locdoc.PriorityContent})
	f.Push(locdoc.DiscoveredLink{URL: "https://example.com/toc", Priority: locdoc.PriorityTOC})

	// Pop should return in priority order (highest first)
	link, ok := f.Pop()
	assert.True(t, ok)
	assert.Equal(t, locdoc.PriorityTOC, link.Priority)
	assert.Equal(t, "https://example.com/toc", link.URL)

	link, ok = f.Pop()
	assert.True(t, ok)
	assert.Equal(t, locdoc.PriorityNavigation, link.Priority)

	link, ok = f.Pop()
	assert.True(t, ok)
	assert.Equal(t, locdoc.PriorityContent, link.Priority)

	link, ok = f.Pop()
	assert.True(t, ok)
	assert.Equal(t, locdoc.PriorityFooter, link.Priority)

	// Queue should now be empty
	_, ok = f.Pop()
	assert.False(t, ok, "pop on empty frontier should return false")
}

func TestFrontier_Len_tracks_queue_size(t *testing.T) {
	t.Parallel()

	f := crawl.NewFrontier(1000, 0.01)

	assert.Equal(t, 0, f.Len(), "new frontier should be empty")

	f.Push(locdoc.DiscoveredLink{URL: "https://example.com/a", Priority: locdoc.PriorityContent})
	assert.Equal(t, 1, f.Len())

	f.Push(locdoc.DiscoveredLink{URL: "https://example.com/b", Priority: locdoc.PriorityContent})
	assert.Equal(t, 2, f.Len())

	f.Pop()
	assert.Equal(t, 1, f.Len())

	f.Pop()
	assert.Equal(t, 0, f.Len())
}

func TestFrontier_Seen_tracks_all_pushed_URLs(t *testing.T) {
	t.Parallel()

	f := crawl.NewFrontier(1000, 0.01)

	assert.False(t, f.Seen("https://example.com/page"), "unseen URL should return false")

	f.Push(locdoc.DiscoveredLink{URL: "https://example.com/page", Priority: locdoc.PriorityContent})

	assert.True(t, f.Seen("https://example.com/page"), "pushed URL should be seen")

	// Pop the URL - it should still be seen
	f.Pop()
	assert.True(t, f.Seen("https://example.com/page"), "popped URL should still be seen")
}

func TestFrontier_Seen_ignores_fragments(t *testing.T) {
	t.Parallel()

	f := crawl.NewFrontier(1000, 0.01)

	// Push URL without fragment
	f.Push(locdoc.DiscoveredLink{URL: "https://example.com/page", Priority: locdoc.PriorityContent})

	// Seen should return true for URL with fragment
	assert.True(t, f.Seen("https://example.com/page#section"), "URL with fragment should be seen if base URL was pushed")

	// Push URL with fragment
	f2 := crawl.NewFrontier(1000, 0.01)
	f2.Push(locdoc.DiscoveredLink{URL: "https://example.com/page#section", Priority: locdoc.PriorityContent})

	// Seen should return true for URL without fragment
	assert.True(t, f2.Seen("https://example.com/page"), "base URL should be seen if URL with fragment was pushed")
}

func TestFrontier_fragment_edge_cases(t *testing.T) {
	t.Parallel()

	t.Run("handles URL with empty fragment", func(t *testing.T) {
		t.Parallel()

		f := crawl.NewFrontier(1000, 0.01)

		// URL with empty fragment should be treated as base URL
		ok := f.Push(locdoc.DiscoveredLink{URL: "https://example.com/page#", Priority: locdoc.PriorityContent})
		assert.True(t, ok)

		// Should be stored without the trailing #
		link, ok := f.Pop()
		assert.True(t, ok)
		assert.Equal(t, "https://example.com/page", link.URL)
	})

	t.Run("handles URL with multiple hash characters", func(t *testing.T) {
		t.Parallel()

		f := crawl.NewFrontier(1000, 0.01)

		// Only the first # should be the fragment delimiter
		// "page#a#b" should become "page"
		ok := f.Push(locdoc.DiscoveredLink{URL: "https://example.com/page#section#subsection", Priority: locdoc.PriorityContent})
		assert.True(t, ok)

		link, ok := f.Pop()
		assert.True(t, ok)
		assert.Equal(t, "https://example.com/page", link.URL)
	})

	t.Run("handles fragment with special characters", func(t *testing.T) {
		t.Parallel()

		f := crawl.NewFrontier(1000, 0.01)

		// Fragment with special characters should be stripped
		ok := f.Push(locdoc.DiscoveredLink{URL: "https://example.com/page#section%20with%20spaces", Priority: locdoc.PriorityContent})
		assert.True(t, ok)

		link, ok := f.Pop()
		assert.True(t, ok)
		assert.Equal(t, "https://example.com/page", link.URL)
	})

	t.Run("preserves query params when stripping fragment", func(t *testing.T) {
		t.Parallel()

		f := crawl.NewFrontier(1000, 0.01)

		// Query params should be preserved, only fragment stripped
		ok := f.Push(locdoc.DiscoveredLink{URL: "https://example.com/page?q=test#section", Priority: locdoc.PriorityContent})
		assert.True(t, ok)

		link, ok := f.Pop()
		assert.True(t, ok)
		assert.Equal(t, "https://example.com/page?q=test", link.URL)
	})
}

func TestFrontier_concurrent_access(t *testing.T) {
	t.Parallel()

	f := crawl.NewFrontier(10000, 0.01)

	const numGoroutines = 10
	const numOpsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // pushers + poppers

	// Start pushers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				url := fmt.Sprintf("https://example.com/%d/%d", id, j)
				f.Push(locdoc.DiscoveredLink{
					URL:      url,
					Priority: locdoc.PriorityContent,
				})
			}
		}(i)
	}

	// Start poppers
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				f.Pop()
				f.Len()
			}
		}()
	}

	wg.Wait()

	// Verify no panic occurred and state is consistent
	// All pushed URLs should be seen
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < numOpsPerGoroutine; j++ {
			url := fmt.Sprintf("https://example.com/%d/%d", i, j)
			assert.True(t, f.Seen(url), "pushed URL %s should be seen", url)
		}
	}
}
