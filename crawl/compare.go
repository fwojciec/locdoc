package crawl

import "github.com/fwojciec/locdoc"

// ContentDiffers compares content extracted from HTTP-fetched HTML vs Rod-fetched HTML.
// Returns true if the Rod content is significantly longer (>50%), suggesting JavaScript
// rendering adds meaningful content. Also returns true on extraction errors (assumes JS needed).
func ContentDiffers(httpHTML, rodHTML string, extractor locdoc.Extractor) bool {
	httpResult, err := extractor.Extract(httpHTML)
	if err != nil {
		return true // Assume JS needed on error
	}

	rodResult, err := extractor.Extract(rodHTML)
	if err != nil {
		return true // Assume JS needed on error
	}

	httpLen := len(httpResult.ContentHTML)
	rodLen := len(rodResult.ContentHTML)

	// Handle empty HTTP content
	if httpLen == 0 && rodLen > 0 {
		return true
	}

	// Check if Rod content is >50% longer
	threshold := float64(httpLen) * 1.5
	return float64(rodLen) > threshold
}
