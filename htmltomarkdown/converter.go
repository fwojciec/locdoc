package htmltomarkdown

import (
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"github.com/fwojciec/locdoc"
)

// Ensure Converter implements locdoc.Converter at compile time.
var _ locdoc.Converter = (*Converter)(nil)

// Converter wraps html-to-markdown to convert HTML to Markdown.
type Converter struct {
	conv *converter.Converter
}

// NewConverter creates a new Converter.
func NewConverter() *Converter {
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			table.NewTablePlugin(),
		),
	)
	return &Converter{conv: conv}
}

// Convert transforms HTML content into Markdown.
func (c *Converter) Convert(html string) (string, error) {
	if strings.TrimSpace(html) == "" {
		return "", locdoc.Errorf(locdoc.EINVALID, "empty HTML input")
	}

	result, err := c.conv.ConvertString(html)
	if err != nil {
		return "", err
	}

	return result, nil
}
