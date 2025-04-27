package gitutils

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// GetGitConfig retrieves a specific git config key's value.
// Returns an error containing "exit status 1" if the key doesn't exist.
func GetGitConfig(key string) (string, error) {
	// Using --null to handle potential whitespace issues, although less likely for our keys
	// Using --default "" makes the command succeed even if key doesn't exist, returning empty.
	// Let's stick to --get which fails if not found, making non-existence explicit.
	output, err := RunGitCommand("config", "--get", key)
	if err != nil {
		// Return the specific error from RunGitCommand, which includes exit code info
		return "", err
	}
	return output, nil
}

// SetGitConfig sets (or adds) a git config key-value pair.
// Uses --add to avoid deleting other values if the key somehow exists multiple times,
// though for our usage, a simple set would likely be fine too.
func SetGitConfig(key, value string) error {
	// Using --local ensures we write to .git/config, not global or system
	_, err := RunGitCommand("config", "--local", "--add", key, value)
	return err
}

// UnsetGitConfig removes a git config key.
// Useful for cleanup or an 'untrack' command.
func UnsetGitConfig(key string) error {
	// Use --unset-all in case --add resulted in multiple entries (unlikely for us)
	_, err := RunGitCommand("config", "--local", "--unset-all", key)
	// Ignore exit code 5 which means the section or key was not found
	if err != nil && !strings.Contains(err.Error(), "exit status 5") {
		return err
	}
	return nil
}

func IsRerereEnabled() (bool, error) {
	output, err := RunGitCommand("config", "--get", "rerere.enabled")

	if err == nil {
		// Config key exists, check its value
		return strings.ToLower(strings.TrimSpace(output)) == "true", nil
	}

	// Check if the error was simply "key not found"
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		// Key not found means rerere is not explicitly enabled (or disabled)
		// via this specific key, treat as disabled.
		return false, nil
	}

	// Any other error is unexpected
	return false, fmt.Errorf("failed to check git config rerere.enabled: %w", err)
}
