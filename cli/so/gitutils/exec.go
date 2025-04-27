package gitutils

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func RunGitCommand(args ...string) (string, error) {

	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	// ... handle general errors, wrap ExitError without specific code checks ...
	if exitErr, ok := err.(*exec.ExitError); ok {
		// Simplified error wrapping, include stderr
		stderrStr := strings.TrimSpace(stderr.String())
		errMsg := fmt.Sprintf("git command failed (%s)", exitErr.Error())
		if stderrStr != "" {
			errMsg = fmt.Sprintf("%s\nstderr: %s", errMsg, stderrStr)
		}
		return "", fmt.Errorf("%s: %w", errMsg, exitErr) // Wrap the original ExitError
	} else if err != nil {
		return "", fmt.Errorf("git command execution failed: %w", err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func RunGitCommandInteractive(args ...string) error {
	cmd := exec.Command("git", args...) // Don't add --no-pager here

	// Connect standard streams directly
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command and wait for it to finish
	err := cmd.Run()
	if err != nil {
		// Unlike RunGitCommand, we don't capture output, so just return the error.
		// The user will have seen any error messages directly in their terminal.
		return fmt.Errorf("interactive git command failed: %w", err)
	}
	return nil
}

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
