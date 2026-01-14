package readability

import (
	"strings"

	"github.com/fwojciec/locdoc"
	"github.com/go-shiori/go-readability"
)

// Ensure Extractor implements locdoc.Extractor at compile time.
var _ locdoc.Extractor = (*Extractor)(nil)

// Extractor wraps go-readability to extract main content from HTML.
type Extractor struct{}

// NewExtractor creates a new Extractor.
func NewExtractor() *Extractor {
	return &Extractor{}
}

// Extract processes raw HTML and returns the main content.
func (e *Extractor) Extract(rawHTML string) (*locdoc.ExtractResult, error) {
	if rawHTML == "" {
		return nil, locdoc.Errorf(locdoc.EINVALID, "empty HTML input")
	}

	article, err := readability.FromReader(strings.NewReader(rawHTML), nil)
	if err != nil {
		return nil, err
	}

	return &locdoc.ExtractResult{
		Title:       article.Title,
		ContentHTML: article.Content,
	}, nil
}
