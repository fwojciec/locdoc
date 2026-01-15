package fs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fwojciec/locdoc"
)

// Ensure FileStore implements locdoc.PageStore at compile time.
var _ locdoc.PageStore = (*FileStore)(nil)

// FileStore implements locdoc.PageStore with atomic update semantics.
// Pages are saved to a temporary directory, then moved atomically on Commit.
type FileStore struct {
	baseDir string
	name    string
}

// NewFileStore creates a new FileStore.
// baseDir is the parent directory, name is the output directory name.
// Files are saved to baseDir/name.tmp and moved to baseDir/name on Commit.
func NewFileStore(baseDir, name string) *FileStore {
	return &FileStore{
		baseDir: baseDir,
		name:    name,
	}
}

func (s *FileStore) tempDir() string {
	return filepath.Join(s.baseDir, s.name+".tmp")
}

func (s *FileStore) finalDir() string {
	return filepath.Join(s.baseDir, s.name)
}

func (s *FileStore) Save(ctx context.Context, page *locdoc.Page) error {
	relPath, err := URLToPath(page.URL)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(s.tempDir(), relPath)

	// Create parent directories
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	content := FormatPage(page)
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// FormatPage formats a page with YAML frontmatter.
func FormatPage(page *locdoc.Page) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("source: ")
	b.WriteString(page.URL)
	b.WriteString("\ntitle: ")
	b.WriteString(page.Title)
	b.WriteString("\ncrawled: ")
	b.WriteString(time.Now().Format("2006-01-02"))
	b.WriteString("\n---\n\n")
	b.WriteString(page.Content)
	return b.String()
}

func (s *FileStore) Commit() error {
	// Remove existing final directory if present
	if err := os.RemoveAll(s.finalDir()); err != nil {
		return err
	}

	// Atomically rename temp to final
	if err := os.Rename(s.tempDir(), s.finalDir()); err != nil {
		return err
	}

	return nil
}

func (s *FileStore) Abort() error {
	return os.RemoveAll(s.tempDir())
}
