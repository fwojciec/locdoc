package crawl

import (
	"fmt"

	"github.com/cespare/xxhash/v2"
)

// computeHash computes a hash of the content using xxhash.
func computeHash(content string) string {
	h := xxhash.Sum64String(content)
	return fmt.Sprintf("%x", h)
}

// ComputeHash computes a hash of the content using xxhash.
// This is the exported version for use in CLI commands.
func ComputeHash(content string) string {
	return computeHash(content)
}

// TruncateURL shortens a URL for display, keeping the end which is more informative.
func TruncateURL(url string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if maxLen < 4 {
		// Too short for "..." prefix, just return dots
		return url[:min(len(url), maxLen)]
	}
	if len(url) <= maxLen {
		return url
	}
	return "..." + url[len(url)-maxLen+3:]
}

// FormatBytes formats bytes in human-readable form.
func FormatBytes(bytes int) string {
	const (
		KB = 1024
		MB = KB * 1024
	)
	switch {
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatTokens formats token count in human-readable form.
func FormatTokens(tokens int) string {
	if tokens < 1000 {
		return fmt.Sprintf("~%d tokens", tokens)
	}
	return fmt.Sprintf("~%dk tokens", (tokens+500)/1000)
}
