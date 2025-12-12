package main_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	main "github.com/fwojciec/locdoc/cmd/locdoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testContext returns a background context for tests.
func testContext() context.Context {
	return context.Background()
}

func TestRun_HelpFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{"--help flag", []string{"--help"}},
		{"-h flag", []string{"-h"}},
		{"help command", []string{"help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := main.NewMain()
			m.DBPath = filepath.Join(t.TempDir(), "test.db")

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			err := m.Run(testContext(), tt.args, stdout, stderr)

			require.NoError(t, err)
			// Usage should be printed to stdout (not stderr) when explicitly requested
			assert.Contains(t, stdout.String(), "Usage: locdoc")
			assert.Contains(t, stdout.String(), "Commands:")
			assert.Empty(t, stderr.String())
		})
	}
}

func TestRun_NoArgs(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	m := main.NewMain()
	m.DBPath = filepath.Join(tmpDir, "test.db")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := m.Run(testContext(), []string{}, stdout, stderr)

	// No args should show usage to stdout and return error with guidance
	require.Error(t, err)
	assert.Contains(t, stdout.String(), "Usage: locdoc")
	assert.Contains(t, err.Error(), "locdoc", "error should mention the command name for context")
}

func TestRun_HelpWithoutCreatingDB(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "should-not-exist.db")

	m := main.NewMain()
	m.DBPath = dbPath

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := m.Run(testContext(), []string{"--help"}, stdout, stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Usage: locdoc")
	assert.Empty(t, stderr.String())

	// Verify database file was NOT created
	_, statErr := os.Stat(dbPath)
	assert.True(t, os.IsNotExist(statErr), "database file should not be created for --help")
}

func TestRun_DatabaseOpenError(t *testing.T) {
	t.Parallel()

	// Use a path inside a non-existent directory to trigger an error
	m := main.NewMain()
	m.DBPath = "/nonexistent/path/that/cannot/exist/test.db"

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := m.Run(testContext(), []string{"list"}, stdout, stderr)

	require.Error(t, err)
	// Error should mention the path to help user understand what went wrong
	assert.Contains(t, err.Error(), "database", "error should mention database")
	assert.Contains(t, err.Error(), "LOCDOC_DB", "error should mention LOCDOC_DB environment variable")
}
