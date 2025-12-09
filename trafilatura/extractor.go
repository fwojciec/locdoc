package trafilatura

import (
	"bytes"
	"errors"
	"strings"

	"github.com/fwojciec/locdoc"
	"github.com/markusmobius/go-trafilatura"
	"golang.org/x/net/html"
)

// Ensure Extractor implements locdoc.Extractor at compile time.
var _ locdoc.Extractor = (*Extractor)(nil)

// Extractor wraps go-trafilatura to extract main content from HTML.
type Extractor struct{}

// NewExtractor creates a new Extractor.
func NewExtractor() *Extractor {
	return &Extractor{}
}

// Extract processes raw HTML and returns the main content.
func (e *Extractor) Extract(rawHTML string) (*locdoc.ExtractResult, error) {
	if rawHTML == "" {
		return nil, errors.New("empty HTML input")
	}

	opts := trafilatura.Options{
		EnableFallback: true,
	}

	result, err := trafilatura.Extract(strings.NewReader(rawHTML), opts)
	if err != nil {
		return nil, err
	}

	var contentHTML string
	if result.ContentNode != nil {
		contentHTML, err = renderNode(result.ContentNode)
		if err != nil {
			return nil, err
		}
	}

	return &locdoc.ExtractResult{
		Title:       result.Metadata.Title,
		ContentHTML: contentHTML,
	}, nil
}

// renderNode converts an html.Node to a string.
func renderNode(n *html.Node) (string, error) {
	var buf bytes.Buffer
	if err := html.Render(&buf, n); err != nil {
		return "", err
	}
	return buf.String(), nil
}
