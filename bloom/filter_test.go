package bloom_test

import (
	"fmt"
	"testing"

	"github.com/fwojciec/locdoc/bloom"
	"github.com/stretchr/testify/assert"
)

func TestFilter_AddAndTest(t *testing.T) {
	t.Parallel()

	f := bloom.NewFilter(1000, 0.01)

	// URL not yet added should return false
	assert.False(t, f.Test("https://example.com/page1"))

	// Add URL
	f.Add("https://example.com/page1")

	// Now it should return true
	assert.True(t, f.Test("https://example.com/page1"))

	// Different URL should still return false
	assert.False(t, f.Test("https://example.com/page2"))
}

func TestFilter_EstimatedCount(t *testing.T) {
	t.Parallel()

	f := bloom.NewFilter(1000, 0.01)

	// Empty filter should have count near 0
	assert.Equal(t, uint(0), f.EstimatedCount())

	// Add some URLs
	f.Add("https://example.com/page1")
	f.Add("https://example.com/page2")
	f.Add("https://example.com/page3")

	// Estimated count should be approximately 3
	count := f.EstimatedCount()
	assert.True(t, count >= 2 && count <= 4, "expected count near 3, got %d", count)
}

func TestFilter_AddIsIdempotent(t *testing.T) {
	t.Parallel()

	f := bloom.NewFilter(1000, 0.01)

	url := "https://example.com/page1"

	f.Add(url)
	countAfterFirst := f.EstimatedCount()

	// Adding the same URL multiple times should not change the filter
	f.Add(url)
	f.Add(url)
	f.Add(url)

	assert.Equal(t, countAfterFirst, f.EstimatedCount())
	assert.True(t, f.Test(url))
}

func TestFilter_FalsePositiveRate(t *testing.T) {
	t.Parallel()

	const (
		numItems   = 10000
		fpRate     = 0.01
		testProbes = 10000
	)

	f := bloom.NewFilter(numItems, fpRate)

	// Add 10k URLs
	for i := range numItems {
		f.Add(fmt.Sprintf("https://example.com/added/%d", i))
	}

	// Test with 10k URLs that were NOT added
	falsePositives := 0
	for i := range testProbes {
		url := fmt.Sprintf("https://example.com/notadded/%d", i)
		if f.Test(url) {
			falsePositives++
		}
	}

	// False positive rate should be approximately 1%
	// Allow up to 2% to account for statistical variance
	actualRate := float64(falsePositives) / float64(testProbes)
	assert.Less(t, actualRate, 0.02, "false positive rate %f exceeds 2%%", actualRate)
}
