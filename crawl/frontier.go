package crawl

import (
	"container/heap"
	"sync"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/bloom"
)

// Compile-time interface verification.
var _ locdoc.URLFrontier = (*Frontier)(nil)

// Frontier is an in-memory URL frontier with priority queue and Bloom filter deduplication.
// It is safe for concurrent use by multiple goroutines.
type Frontier struct {
	mu    sync.Mutex
	seen  *bloom.Filter
	queue *linkHeap
}

// NewFrontier creates a new Frontier sized for n expected URLs
// with the given false positive rate for deduplication.
func NewFrontier(n uint, fpRate float64) *Frontier {
	h := &linkHeap{}
	heap.Init(h)
	return &Frontier{
		seen:  bloom.NewFilter(n, fpRate),
		queue: h,
	}
}

// Push adds a link to the frontier.
// Returns false if the URL has already been seen.
func (f *Frontier) Push(link locdoc.DiscoveredLink) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.seen.Test(link.URL) {
		return false
	}
	f.seen.Add(link.URL)
	heap.Push(f.queue, link)
	return true
}

// Pop returns the next link by priority.
// The bool result is false if the frontier is empty.
func (f *Frontier) Pop() (locdoc.DiscoveredLink, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.queue.Len() == 0 {
		return locdoc.DiscoveredLink{}, false
	}
	link, _ := heap.Pop(f.queue).(locdoc.DiscoveredLink)
	return link, true
}

// Len returns the number of URLs in the queue.
func (f *Frontier) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.queue.Len()
}

// Seen returns true if the URL has been processed or queued.
func (f *Frontier) Seen(url string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.seen.Test(url)
}

// linkHeap implements heap.Interface for DiscoveredLink priority queue.
// Higher priority links are popped first.
type linkHeap []locdoc.DiscoveredLink

func (h linkHeap) Len() int { return len(h) }

// Less returns true if i has higher priority than j (max-heap).
func (h linkHeap) Less(i, j int) bool {
	return h[i].Priority > h[j].Priority
}

func (h linkHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *linkHeap) Push(x any) {
	link, _ := x.(locdoc.DiscoveredLink)
	*h = append(*h, link)
}

func (h *linkHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
