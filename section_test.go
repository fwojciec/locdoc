package locdoc_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/stretchr/testify/assert"
)

func TestExtractSections(t *testing.T) {
	t.Parallel()

	t.Run("extracts H1 heading", func(t *testing.T) {
		t.Parallel()

		markdown := "# Introduction\n\nSome content here."

		sections := locdoc.ExtractSections(markdown)

		assert.Len(t, sections, 1)
		assert.Equal(t, 1, sections[0].Level)
		assert.Equal(t, "Introduction", sections[0].Title)
		assert.Equal(t, "introduction", sections[0].Anchor)
	})

	t.Run("extracts H2 through H6 headings", func(t *testing.T) {
		t.Parallel()

		markdown := `# H1 Title
## H2 Title
### H3 Title
#### H4 Title
##### H5 Title
###### H6 Title`

		sections := locdoc.ExtractSections(markdown)

		assert.Len(t, sections, 6)
		assert.Equal(t, 1, sections[0].Level)
		assert.Equal(t, 2, sections[1].Level)
		assert.Equal(t, 3, sections[2].Level)
		assert.Equal(t, 4, sections[3].Level)
		assert.Equal(t, 5, sections[4].Level)
		assert.Equal(t, 6, sections[5].Level)
	})

	t.Run("generates URL-safe anchors", func(t *testing.T) {
		t.Parallel()

		markdown := "# Getting Started With Go"

		sections := locdoc.ExtractSections(markdown)

		assert.Len(t, sections, 1)
		assert.Equal(t, "getting-started-with-go", sections[0].Anchor)
	})

	t.Run("handles duplicate headings with numeric suffixes", func(t *testing.T) {
		t.Parallel()

		markdown := `# Example
## Example
### Example`

		sections := locdoc.ExtractSections(markdown)

		assert.Len(t, sections, 3)
		assert.Equal(t, "example", sections[0].Anchor)
		assert.Equal(t, "example-1", sections[1].Anchor)
		assert.Equal(t, "example-2", sections[2].Anchor)
	})

	t.Run("returns empty slice for empty markdown", func(t *testing.T) {
		t.Parallel()

		sections := locdoc.ExtractSections("")

		assert.Empty(t, sections)
	})

	t.Run("returns empty slice for markdown without headings", func(t *testing.T) {
		t.Parallel()

		markdown := "Just some text\n\nWith paragraphs."

		sections := locdoc.ExtractSections(markdown)

		assert.Empty(t, sections)
	})

	t.Run("strips special characters from anchors", func(t *testing.T) {
		t.Parallel()

		markdown := "# API Reference (v2.0)"

		sections := locdoc.ExtractSections(markdown)

		assert.Len(t, sections, 1)
		assert.Equal(t, "api-reference-v20", sections[0].Anchor)
	})

	t.Run("ignores code blocks with hash symbols", func(t *testing.T) {
		t.Parallel()

		markdown := `# Real Heading

` + "```bash\n# This is a comment\necho hello\n```" + `

## Another Real Heading`

		sections := locdoc.ExtractSections(markdown)

		assert.Len(t, sections, 2)
		assert.Equal(t, "Real Heading", sections[0].Title)
		assert.Equal(t, "Another Real Heading", sections[1].Title)
	})
}
