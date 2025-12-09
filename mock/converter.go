package mock

import "github.com/fwojciec/locdoc"

var _ locdoc.Converter = (*Converter)(nil)

// Converter is a mock implementation of locdoc.Converter.
type Converter struct {
	ConvertFn func(html string) (string, error)
}

func (c *Converter) Convert(html string) (string, error) {
	return c.ConvertFn(html)
}
