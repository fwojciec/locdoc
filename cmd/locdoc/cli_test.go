package main_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	main "github.com/fwojciec/locdoc/cmd/locdoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLI_HelpShowsAllCommands(t *testing.T) {
	t.Parallel()

	cli := &main.CLI{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Use kong.Exit to prevent os.Exit from being called during tests
	parser, err := kong.New(cli,
		kong.Writers(stdout, stderr),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	// Parse --help (Kong writes help to stdout)
	_, _ = parser.Parse([]string{"--help"})

	// Kong prints help even if Parse returns an error
	// The help text should mention all commands
	helpOutput := stdout.String()

	expectedCommands := []string{"add", "list", "delete", "docs", "ask"}
	for _, cmd := range expectedCommands {
		assert.Contains(t, helpOutput, cmd, "Help should mention %s command", cmd)
	}
}

func TestMain_Run_HelpShowsKongOutput(t *testing.T) {
	t.Parallel()

	m := main.NewMain()
	m.DBPath = filepath.Join(t.TempDir(), "test.db")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// --help should return nil (success) and show commands
	err := m.Run(context.Background(), []string{"--help"}, stdout, stderr)
	require.NoError(t, err)

	// Kong should have written help to stdout with all commands
	helpOutput := stdout.String()
	expectedCommands := []string{"add", "list", "delete", "docs", "ask"}
	for _, cmd := range expectedCommands {
		assert.Contains(t, helpOutput, cmd, "Help should mention %s command", cmd)
	}

	// Verify Kong-style formatting (Kong has "Usage:" prefix and "Flags:" section)
	assert.Contains(t, helpOutput, "Usage:", "Help should have Kong-style Usage prefix")
	assert.Contains(t, helpOutput, "Flags:", "Help should have Kong-style Flags section")
}
