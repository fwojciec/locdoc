package crawl_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
)

func TestContentDiffers(t *testing.T) {
	t.Parallel()

	t.Run("returns true when Rod content is more than 50% longer", func(t *testing.T) {
		t.Parallel()

		extractor := &mock.Extractor{
			ExtractFn: func(html string) (*locdoc.ExtractResult, error) {
				// Return different lengths based on input
				if html == "http-html" {
					return &locdoc.ExtractResult{
						ContentHTML: "short content", // 13 chars
					}, nil
				}
				return &locdoc.ExtractResult{
					ContentHTML: "much longer content from rod which is significantly bigger", // 58 chars, >50% longer
				}, nil
			},
		}

		result := crawl.ContentDiffers("http-html", "rod-html", extractor)

		assert.True(t, result, "should return true when Rod content is >50% longer")
	})

	t.Run("returns false when content lengths are similar", func(t *testing.T) {
		t.Parallel()

		extractor := &mock.Extractor{
			ExtractFn: func(html string) (*locdoc.ExtractResult, error) {
				if html == "http-html" {
					return &locdoc.ExtractResult{
						ContentHTML: "some content here", // 17 chars
					}, nil
				}
				return &locdoc.ExtractResult{
					ContentHTML: "similar size text", // 17 chars (equal)
				}, nil
			},
		}

		result := crawl.ContentDiffers("http-html", "rod-html", extractor)

		assert.False(t, result, "should return false when content is similar length")
	})

	t.Run("returns false when Rod content is only 50% longer", func(t *testing.T) {
		t.Parallel()

		extractor := &mock.Extractor{
			ExtractFn: func(html string) (*locdoc.ExtractResult, error) {
				if html == "http-html" {
					return &locdoc.ExtractResult{
						ContentHTML: "0123456789", // 10 chars
					}, nil
				}
				return &locdoc.ExtractResult{
					ContentHTML: "012345678901234", // 15 chars (exactly 50% longer)
				}, nil
			},
		}

		result := crawl.ContentDiffers("http-html", "rod-html", extractor)

		assert.False(t, result, "should return false when Rod content is exactly 50% longer (boundary)")
	})

	t.Run("returns true when HTTP extraction fails", func(t *testing.T) {
		t.Parallel()

		extractor := &mock.Extractor{
			ExtractFn: func(html string) (*locdoc.ExtractResult, error) {
				if html == "http-html" {
					return nil, locdoc.Errorf(locdoc.EINTERNAL, "extraction failed")
				}
				return &locdoc.ExtractResult{
					ContentHTML: "rod content",
				}, nil
			},
		}

		result := crawl.ContentDiffers("http-html", "rod-html", extractor)

		assert.True(t, result, "should return true when HTTP extraction fails (assume JS needed)")
	})

	t.Run("returns true when Rod extraction fails", func(t *testing.T) {
		t.Parallel()

		extractor := &mock.Extractor{
			ExtractFn: func(html string) (*locdoc.ExtractResult, error) {
				if html == "http-html" {
					return &locdoc.ExtractResult{
						ContentHTML: "http content",
					}, nil
				}
				return nil, locdoc.Errorf(locdoc.EINTERNAL, "extraction failed")
			},
		}

		result := crawl.ContentDiffers("http-html", "rod-html", extractor)

		assert.True(t, result, "should return true when Rod extraction fails (assume JS needed)")
	})

	t.Run("returns true when HTTP content is empty", func(t *testing.T) {
		t.Parallel()

		extractor := &mock.Extractor{
			ExtractFn: func(html string) (*locdoc.ExtractResult, error) {
				if html == "http-html" {
					return &locdoc.ExtractResult{
						ContentHTML: "", // Empty
					}, nil
				}
				return &locdoc.ExtractResult{
					ContentHTML: "rod has content",
				}, nil
			},
		}

		result := crawl.ContentDiffers("http-html", "rod-html", extractor)

		assert.True(t, result, "should return true when HTTP content is empty but Rod has content")
	})

	t.Run("returns true when both extractions fail", func(t *testing.T) {
		t.Parallel()

		extractor := &mock.Extractor{
			ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
				return nil, locdoc.Errorf(locdoc.EINTERNAL, "extraction failed")
			},
		}

		result := crawl.ContentDiffers("http-html", "rod-html", extractor)

		assert.True(t, result, "should return true when both extractions fail (assume JS needed)")
	})
}
