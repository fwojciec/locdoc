package main_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/alecthomas/kong"
	main "github.com/fwojciec/locdoc/cmd/locdoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddCmd_TimeoutFlag(t *testing.T) {
	t.Parallel()

	cli := &main.CLI{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	parser, err := kong.New(cli,
		kong.Writers(stdout, stderr),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	// Parse add command with --timeout flag
	_, err = parser.Parse([]string{"add", "--timeout", "30s", "myproject", "https://example.com"})
	require.NoError(t, err)

	// Verify the timeout was parsed correctly
	assert.Equal(t, 30*time.Second, cli.Add.Timeout)
}

func TestAddCmd_TimeoutFlagDefault(t *testing.T) {
	t.Parallel()

	cli := &main.CLI{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	parser, err := kong.New(cli,
		kong.Writers(stdout, stderr),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	// Parse add command without --timeout flag
	_, err = parser.Parse([]string{"add", "myproject", "https://example.com"})
	require.NoError(t, err)

	// Verify the default timeout is 10 seconds
	assert.Equal(t, 10*time.Second, cli.Add.Timeout)
}

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
