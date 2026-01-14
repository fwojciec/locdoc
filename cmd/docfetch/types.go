package main

// Page represents a fetched documentation page.
type Page struct {
	URL     string
	Title   string
	Content string // Markdown
}

// ProgressEvent reports fetch progress.
type ProgressEvent struct {
	URL       string
	Completed int
	Total     int
	Error     error
}

// ProgressFunc is called as pages are processed.
type ProgressFunc func(ProgressEvent)
