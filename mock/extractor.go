package mock

import "github.com/fwojciec/locdoc"

var _ locdoc.Extractor = (*Extractor)(nil)

// Extractor is a mock implementation of locdoc.Extractor.
type Extractor struct {
	ExtractFn func(html string) (*locdoc.ExtractResult, error)
}

func (e *Extractor) Extract(html string) (*locdoc.ExtractResult, error) {
	return e.ExtractFn(html)
}
