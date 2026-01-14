// Package htmltomarkdown provides HTML to Markdown conversion
// using JohannesKaufmann/html-to-markdown.
package htmltomarkdown

import (
	"regexp"
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

	return c.postProcess(result), nil
}

// postProcess applies cleanup transformations to the converted markdown.
func (c *Converter) postProcess(md string) string {
	return trimCodeBlockWhitespace(md)
}

// trimCodeBlockWhitespace removes leading and trailing blank lines from
// fenced code blocks. These blank lines are artifacts from HTML structure
// (whitespace between tags and content) and have no value in documentation.
func trimCodeBlockWhitespace(md string) string {
	// Remove blank lines immediately after opening fence: ```lang\n\n → ```lang\n
	leadingRe := regexp.MustCompile("(```\\w*)\n(\n)+")
	md = leadingRe.ReplaceAllString(md, "$1\n")

	// Remove blank lines immediately before closing fence: \n\n``` → \n```
	trailingRe := regexp.MustCompile("\n(\n)+```")
	return trailingRe.ReplaceAllString(md, "\n```")
}
