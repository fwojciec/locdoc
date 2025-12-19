// Package bloom provides URL deduplication using Bloom filters.
package bloom

import "github.com/bits-and-blooms/bloom/v3"

// Filter wraps a Bloom filter for URL deduplication.
type Filter struct {
	f *bloom.BloomFilter
}

// NewFilter creates a new Bloom filter sized for n expected items
// with the given false positive rate.
func NewFilter(n uint, fpRate float64) *Filter {
	return &Filter{
		f: bloom.NewWithEstimates(n, fpRate),
	}
}

// Add adds a URL to the filter.
func (f *Filter) Add(url string) {
	f.f.AddString(url)
}

// Test returns true if the URL might be in the filter.
// False positives are possible; false negatives are not.
func (f *Filter) Test(url string) bool {
	return f.f.TestString(url)
}

// EstimatedCount returns the approximate number of items in the filter.
func (f *Filter) EstimatedCount() uint {
	return uint(f.f.ApproximatedSize())
}
