package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runShellCommand always returns nil so the scheduler never retries a command that
// ran but failed (e.g. a side-effect partially succeeded).  Tests below verify this
// contract and show the supported calling conventions.

func TestRunShellCommand_simpleCommand(t *testing.T) {
	err := runShellCommand(context.Background(), t.TempDir(), "echo", "hello")
	assert.NoError(t, err)
}

func TestRunShellCommand_multipleArgs(t *testing.T) {
	// Any number of positional args is supported via the variadic signature.
	err := runShellCommand(context.Background(), t.TempDir(), "echo", "one", "two", "three")
	assert.NoError(t, err)
}

func TestRunShellCommand_noArgs(t *testing.T) {
	err := runShellCommand(context.Background(), t.TempDir(), "true")
	assert.NoError(t, err)
}

func TestRunShellCommand_runsInSpecifiedDirectory(t *testing.T) {
	dir := t.TempDir()
	err := runShellCommand(context.Background(), dir, "sh", "-c", "touch marker.txt")
	require.NoError(t, err)
	_, statErr := os.Stat(filepath.Join(dir, "marker.txt"))
	assert.NoError(t, statErr, "command should have created marker.txt in the specified directory")
}

func TestRunShellCommand_shellPipelineViaShC(t *testing.T) {
	// Shell features like pipes and redirects are supported by passing through sh -c.
	dir := t.TempDir()
	err := runShellCommand(context.Background(), dir, "sh", "-c", "echo hello | tr a-z A-Z > output.txt")
	require.NoError(t, err)
	content, readErr := os.ReadFile(filepath.Join(dir, "output.txt"))
	require.NoError(t, readErr)
	assert.Equal(t, "HELLO\n", string(content))
}

func TestRunShellCommand_nonZeroExitReturnsNil(t *testing.T) {
	// A command that exits non-zero must still return nil: the scheduler marks the
	// action as done and does not retry it on the next evaluation cycle.
	err := runShellCommand(context.Background(), t.TempDir(), "false")
	assert.NoError(t, err, "non-zero exit must not cause scheduler to retry")
}

func TestRunShellCommand_commandNotFoundReturnsNil(t *testing.T) {
	// An unknown command is logged as a warning but must not surface as an error.
	err := runShellCommand(context.Background(), t.TempDir(), "this-command-does-not-exist-xyzzy")
	assert.NoError(t, err, "command-not-found must not cause scheduler to retry")
}

func TestRunShellCommand_cancelledContextReturnsNil(t *testing.T) {
	// A cancelled context stops the child process but the scheduler action is still
	// considered done — no retry should happen.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := runShellCommand(ctx, t.TempDir(), "sleep", "60")
	assert.NoError(t, err, "cancelled context must not cause scheduler to retry")
}

func TestRunShellCommand_dockerComposePattern(t *testing.T) {
	// Mirrors the real scheduler call:
	//   runShellCommand(ctx, "/homeapp73-docker", "docker", "compose", "run", "--rm", "service")
	// Verified here with a real command that exercises the same argument structure.
	dir := t.TempDir()
	err := runShellCommand(context.Background(), dir,
		"sh", "-c", "echo compose run --rm service > compose.log")
	require.NoError(t, err)
	content, readErr := os.ReadFile(filepath.Join(dir, "compose.log"))
	require.NoError(t, readErr)
	assert.Equal(t, "compose run --rm service\n", string(content))
}
