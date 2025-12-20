package crawl_test

import (
	"testing"

	"github.com/fwojciec/locdoc/crawl"
	"github.com/stretchr/testify/assert"
)

func TestTruncateURL(t *testing.T) {
	t.Parallel()

	t.Run("returns URL unchanged when shorter than max", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "https://x.com", crawl.TruncateURL("https://x.com", 50))
	})

	t.Run("truncates with ellipsis when longer than max", func(t *testing.T) {
		t.Parallel()
		url := "https://example.com/very/long/path/to/documentation"
		result := crawl.TruncateURL(url, 20)
		assert.Equal(t, ".../to/documentation", result)
		assert.Len(t, result, 20)
	})

	t.Run("returns URL unchanged when exactly max length", func(t *testing.T) {
		t.Parallel()
		url := "https://example.com"
		assert.Equal(t, url, crawl.TruncateURL(url, len(url)))
	})

	t.Run("returns empty string when maxLen is zero", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, crawl.TruncateURL("https://example.com", 0))
	})

	t.Run("returns empty string when maxLen is negative", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, crawl.TruncateURL("https://example.com", -1))
	})

	t.Run("returns prefix of URL when maxLen is very small", func(t *testing.T) {
		t.Parallel()
		// When maxLen < 4, we can't fit "..." prefix, so return URL prefix
		assert.Equal(t, "htt", crawl.TruncateURL("https://example.com", 3))
		assert.Equal(t, "ht", crawl.TruncateURL("https://example.com", 2))
		assert.Equal(t, "h", crawl.TruncateURL("https://example.com", 1))
	})

	t.Run("handles short URL with small maxLen", func(t *testing.T) {
		t.Parallel()
		// URL shorter than maxLen should return unchanged
		assert.Equal(t, "ab", crawl.TruncateURL("ab", 3))
		assert.Equal(t, "a", crawl.TruncateURL("a", 2))
	})
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	t.Run("formats bytes as B", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "512 B", crawl.FormatBytes(512))
	})

	t.Run("formats kilobytes as KB", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "1.5 KB", crawl.FormatBytes(1536))
	})

	t.Run("formats megabytes as MB", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "2.0 MB", crawl.FormatBytes(2*1024*1024))
	})
}

func TestFormatTokens(t *testing.T) {
	t.Parallel()

	t.Run("formats small token counts", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "~500 tokens", crawl.FormatTokens(500))
	})

	t.Run("formats large token counts as k", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "~10k tokens", crawl.FormatTokens(10000))
	})

	t.Run("rounds token counts", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "~2k tokens", crawl.FormatTokens(1500))
	})
}

func TestComputeHash(t *testing.T) {
	t.Parallel()

	t.Run("returns consistent hash for same content", func(t *testing.T) {
		t.Parallel()
		content := "test content"
		hash1 := crawl.ComputeHash(content)
		hash2 := crawl.ComputeHash(content)
		assert.Equal(t, hash1, hash2)
	})

	t.Run("returns different hashes for different content", func(t *testing.T) {
		t.Parallel()
		hash1 := crawl.ComputeHash("content a")
		hash2 := crawl.ComputeHash("content b")
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("returns hex string", func(t *testing.T) {
		t.Parallel()
		hash := crawl.ComputeHash("test")
		assert.Regexp(t, `^[0-9a-f]+$`, hash)
	})
}
