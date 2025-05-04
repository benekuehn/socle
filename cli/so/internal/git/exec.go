package git

import (
	"bytes"
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
