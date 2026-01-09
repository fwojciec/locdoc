// Package fs provides file-based storage for documentation.
package fs

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/fwojciec/locdoc"
)

// URLToPath converts a documentation URL to a relative file path.
// Example: https://example.com/docs/api/users → docs/api/users.md
func URLToPath(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	path := u.Path

	// Handle root or trailing slash → index.md
	if path == "" || path == "/" {
		return "index.md", nil
	}

	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	// Trailing slash becomes index.md in that directory
	if strings.HasSuffix(path, "/") {
		return path + "index.md", nil
	}

	// Otherwise append .md
	return path + ".md", nil
}

// FormatDocument formats a document with YAML frontmatter.
func FormatDocument(doc *locdoc.Document) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("source: ")
	b.WriteString(doc.SourceURL)
	b.WriteString("\ntitle: ")
	b.WriteString(doc.Title)
	b.WriteString("\ncrawled: ")
	b.WriteString(doc.FetchedAt.Format("2006-01-02"))
	b.WriteString("\n---\n\n")
	b.WriteString(doc.Content)
	return b.String()
}

// Ensure Writer implements locdoc.DocumentWriter at compile time.
var _ locdoc.DocumentWriter = (*Writer)(nil)

// Writer writes documents as markdown files to a directory.
type Writer struct {
	baseDir string
}

// NewWriter creates a new Writer that writes to the given base directory.
func NewWriter(baseDir string) *Writer {
	return &Writer{baseDir: baseDir}
}

// CreateDocument writes a document to disk as a markdown file.
func (w *Writer) CreateDocument(ctx context.Context, doc *locdoc.Document) error {
	if err := doc.Validate(); err != nil {
		return err
	}

	relPath, err := URLToPath(doc.SourceURL)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(w.baseDir, relPath)

	// Create parent directories
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	content := FormatDocument(doc)
	return os.WriteFile(fullPath, []byte(content), 0644)
}
