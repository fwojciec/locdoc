//go:build integration && !windows

package rod_test

import (
	"syscall"
	"testing"
	"time"

	"github.com/fwojciec/locdoc/rod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetcher_Close_KillsLauncherProcess(t *testing.T) {
	t.Parallel()

	fetcher, err := rod.NewFetcher()
	require.NoError(t, err)

	// Get the launcher PID before closing
	pid := fetcher.LauncherPID()
	require.NotZero(t, pid, "launcher PID should be set")

	// Verify process is running before close using signal 0
	// On Unix, FindProcess always succeeds, so we must use Signal to verify
	err = syscall.Kill(pid, syscall.Signal(0))
	require.NoError(t, err, "launcher process should be running before Close()")

	// Close should clean up all resources including launcher
	err = fetcher.Close()
	require.NoError(t, err)

	// Give the OS a moment to clean up the process
	time.Sleep(100 * time.Millisecond)

	// Signal 0 checks if process exists without affecting it
	// If the process doesn't exist, Kill returns an error
	err = syscall.Kill(pid, syscall.Signal(0))
	assert.Error(t, err, "launcher process should be terminated after Close()")
}
