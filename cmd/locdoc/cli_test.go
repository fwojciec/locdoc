package main_test

import (
	"bytes"
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/alecthomas/kong"
	main "github.com/fwojciec/locdoc/cmd/locdoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLI_HasAllCommands(t *testing.T) {
	t.Parallel()

	cli := &main.CLI{}
	cliType := reflect.TypeOf(*cli)

	// Verify all 5 command fields exist with cmd struct tags
	expectedCommands := []string{"Add", "List", "Delete", "Docs", "Ask"}
	for _, cmdName := range expectedCommands {
		t.Run(cmdName+" command exists", func(t *testing.T) {
			t.Parallel()

			field, found := cliType.FieldByName(cmdName)
			require.True(t, found, "CLI struct should have %s field", cmdName)

			// Verify it has cmd struct tag for Kong (value can be empty, just needs tag)
			_, hasCmdTag := field.Tag.Lookup("cmd")
			assert.True(t, hasCmdTag, "%s should have cmd struct tag", cmdName)
		})
	}
}

func TestCLI_KongParserCreation(t *testing.T) {
	t.Parallel()

	cli := &main.CLI{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Kong parser should be creatable with the CLI struct
	parser, err := kong.New(cli, kong.Writers(stdout, stderr))
	require.NoError(t, err, "Kong parser should be created without error")
	require.NotNil(t, parser, "Kong parser should not be nil")
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

func TestAddCmd_Run_ReturnsNil(t *testing.T) {
	t.Parallel()

	cmd := &main.AddCmd{
		Name: "test",
		URL:  "https://example.com",
	}

	err := cmd.Run()
	assert.NoError(t, err, "AddCmd.Run() should return nil (stub)")
}

func TestListCmd_Run_ReturnsNil(t *testing.T) {
	t.Parallel()

	cmd := &main.ListCmd{}

	err := cmd.Run()
	assert.NoError(t, err, "ListCmd.Run() should return nil (stub)")
}

func TestDeleteCmd_Run_ReturnsNil(t *testing.T) {
	t.Parallel()

	cmd := &main.DeleteCmd{Name: "test"}

	err := cmd.Run()
	assert.NoError(t, err, "DeleteCmd.Run() should return nil (stub)")
}

func TestDocsCmd_Run_ReturnsNil(t *testing.T) {
	t.Parallel()

	cmd := &main.DocsCmd{Name: "test"}

	err := cmd.Run()
	assert.NoError(t, err, "DocsCmd.Run() should return nil (stub)")
}

func TestAskCmd_Run_ReturnsNil(t *testing.T) {
	t.Parallel()

	cmd := &main.AskCmd{Name: "test", Question: "what?"}

	err := cmd.Run()
	assert.NoError(t, err, "AskCmd.Run() should return nil (stub)")
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
