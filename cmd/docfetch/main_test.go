package main_test

import (
	"bytes"
	"context"
	"testing"

	main "github.com/fwojciec/locdoc/cmd/docfetch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain_Run_Help(t *testing.T) {
	t.Parallel()

	m := main.NewMain()
	var stdout, stderr bytes.Buffer

	err := m.Run(context.Background(), []string{"--help"}, &stdout, &stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "docfetch")
	assert.Contains(t, stdout.String(), "url")
}

func TestMain_Run_NoArgs(t *testing.T) {
	t.Parallel()

	m := main.NewMain()
	var stdout, stderr bytes.Buffer

	err := m.Run(context.Background(), []string{}, &stdout, &stderr)

	assert.Error(t, err)
}

func TestMain_Run_PreviewRequiresURL(t *testing.T) {
	t.Parallel()

	m := main.NewMain()
	var stdout, stderr bytes.Buffer

	err := m.Run(context.Background(), []string{"--preview"}, &stdout, &stderr)

	assert.Error(t, err)
}

func TestMain_Run_FetchRequiresNameAndURL(t *testing.T) {
	t.Parallel()

	m := main.NewMain()
	var stdout, stderr bytes.Buffer

	// Only URL, no name
	err := m.Run(context.Background(), []string{"https://example.com/docs"}, &stdout, &stderr)

	// Should error because name is required for fetch mode
	assert.Error(t, err)
}
