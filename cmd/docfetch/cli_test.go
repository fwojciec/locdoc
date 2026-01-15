package main_test

import (
	"bytes"
	"context"
	"testing"

	main "github.com/fwojciec/locdoc/cmd/docfetch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Story: CLI Help and Discovery
//
// Users discover docfetch capabilities through help output. The CLI should
// make it easy to understand what arguments are required and what options
// are available.

func TestCLI_ShowsHelpWhenAsked(t *testing.T) {
	t.Parallel()

	// Given: a CLI instance
	m := main.NewMain()
	var stdout, stderr bytes.Buffer

	// When: running with --help flag
	err := m.Run(context.Background(), []string{"--help"}, &stdout, &stderr)

	// Then: help is displayed without error
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "docfetch")
	assert.Contains(t, stdout.String(), "url")
}

func TestCLI_ShowsHelpWhenNoArgumentsProvided(t *testing.T) {
	t.Parallel()

	// Given: a CLI instance
	m := main.NewMain()
	var stdout, stderr bytes.Buffer

	// When: running with no arguments
	err := m.Run(context.Background(), []string{}, &stdout, &stderr)

	// Then: help is shown but an error is returned
	require.Error(t, err)
	assert.Contains(t, stdout.String(), "docfetch")
}

// Story: CLI Validation
//
// The CLI validates arguments before attempting to fetch. URL is required
// for all operations. Name is required for fetch mode but optional for
// preview mode.

func TestCLI_RequiresURLForAllOperations(t *testing.T) {
	t.Parallel()

	// Given: a CLI instance
	m := main.NewMain()
	var stdout, stderr bytes.Buffer

	// When: running preview without URL
	err := m.Run(context.Background(), []string{"--preview"}, &stdout, &stderr)

	// Then: an error is returned
	assert.Error(t, err)
}

func TestCLI_RequiresNameForFetchMode(t *testing.T) {
	t.Parallel()

	// Given: a CLI instance
	m := main.NewMain()
	var stdout, stderr bytes.Buffer

	// When: running with URL but no name (fetch mode)
	err := m.Run(context.Background(), []string{"https://example.com/docs"}, &stdout, &stderr)

	// Then: an error is returned because name is required
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestCLI_AllowsPreviewWithoutName(t *testing.T) {
	t.Parallel()

	// Given: a CLI instance
	m := main.NewMain()
	var stdout, stderr bytes.Buffer

	// When: running preview with URL but no name
	err := m.Run(context.Background(), []string{"--preview", "https://example.com/docs"}, &stdout, &stderr)

	// Then: validation passes (no "name is required" error)
	// Execution may fail for other reasons (browser, network) but CLI validation succeeds
	if err != nil {
		assert.NotContains(t, err.Error(), "name is required")
	}
}
