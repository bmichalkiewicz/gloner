// Package exec wraps os/exec to run external commands with captured output.
package exec

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

// Run executes a command in dir (current directory when empty) and returns its stdout.
// Stderr is captured and included in the returned error, so failures are self-explanatory.
// The command is killed when ctx is cancelled.
func Run(ctx context.Context, dir string, command string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Debug().Msgf("Running command: %s %s", command, strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		return stdout.String(), &RunError{
			Command:   cmd,
			ExecError: err,
			Stderr:    strings.TrimSpace(stderr.String()),
		}
	}

	return stdout.String(), nil
}

// RunError represents an error that occurred while running a command.
type RunError struct {
	Command   *exec.Cmd
	ExecError error
	Stderr    string
}

// Error implements the error interface for RunError.
func (e *RunError) Error() string {
	if e.Stderr == "" {
		return fmt.Sprintf("%s: %s", e.Command.Path, e.ExecError)
	}
	return fmt.Sprintf("%s: %s: %s", e.Command.Path, e.ExecError, e.Stderr)
}

// Unwrap returns the underlying execution error.
func (e *RunError) Unwrap() error {
	return e.ExecError
}
