package exec

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// RunExternalCommand executes an arbitrary command (non-Git) and returns its
// standard output as a string, or an error if the command fails.
// The error will include context like the command run, stderr, and exit code.
func RunExternalCommand(commandName string, args ...string) (string, error) {
	cmd := exec.Command(commandName, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run() // Run the command and wait for completion

	stdoutStr := strings.TrimSpace(stdoutBuf.String())
	stderrStr := strings.TrimSpace(stderrBuf.String())

	if err != nil {
		// Format a comprehensive error message
		errMsg := fmt.Sprintf("command '%s %s' failed: %v", commandName, strings.Join(args, " "), err)
		if stderrStr != "" {
			errMsg = fmt.Sprintf("%s\nstderr: %s", errMsg, stderrStr)
		}

		// Try to wrap the original *exec.ExitError to preserve exit code info
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Wrap the specific exit error type
			return "", fmt.Errorf("%s: %w", errMsg, exitErr)
		}

		// If it wasn't an ExitError, return a generic error with the message
		return "", errors.New(errMsg)
	}

	// Success, return standard output
	return stdoutStr, nil
}
