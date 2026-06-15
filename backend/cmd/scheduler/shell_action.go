package main

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/rs/zerolog/log"
)

// runShellCommand executes name with args in dir, logging all output.
// Non-zero exit codes are logged as warnings but do not cause the scheduler to
// retry the action — the daily run is considered done even if the command fails
// partially (e.g. DB unreachable after the data scrape succeeded).
func runShellCommand(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	logger := log.Info().
		Str("dir", dir).
		Str("command", name).
		Strs("args", args).
		Str("stdout", stdout.String()).
		Str("stderr", stderr.String())

	if err != nil {
		logger.Err(err).Msg("command finished with error (not retrying)")
		return nil
	}
	logger.Msg("command finished successfully")
	return nil
}
