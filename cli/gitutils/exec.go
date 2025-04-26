package gitutils

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func RunGitCommand(args ...string) (string, error) {

	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	stderrStr := strings.TrimSpace(stderr.String())
	// Special handling: merge-base returns exit code 1 if no common ancestor found
	if cmd.ProcessState != nil && !cmd.ProcessState.Success() {
		isMergeBaseCmd := len(args) > 0 && args[0] == "merge-base"
		if isMergeBaseCmd && cmd.ProcessState.ExitCode() == 1 {
			// Return a specific error message for merge-base failure
			return "", fmt.Errorf("git merge-base failed: %w - %s", err, stderrStr)
		}
		// Handle other errors
		if stderrStr != "" {
			return "", fmt.Errorf("git command failed (%s): %w\nstderr: %s", cmd.ProcessState.String(), err, stderrStr)
		}
		return "", fmt.Errorf("git command failed (%s): %w", cmd.ProcessState.String(), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetCurrentBranch (Keep the existing one)
func GetCurrentBranch() (string, error) {
	return RunGitCommand("rev-parse", "--abbrev-ref", "HEAD")
}

// GetRepoRoot (Keep the existing one)
func GetRepoRoot() (string, error) {
	return RunGitCommand("rev-parse", "--show-toplevel")
}

// IsGitRepo (Keep the existing one)
func IsGitRepo() bool {
	_, err := RunGitCommand("rev-parse", "--is-inside-work-tree")
	return err == nil
}

// GetMergeBase finds the best common ancestor commit between two refs.
func GetMergeBase(ref1, ref2 string) (string, error) {
	output, err := RunGitCommand("merge-base", ref1, ref2)
	if err != nil {
		// Propagate the specific error from RunGitCommand for merge-base
		if strings.Contains(err.Error(), "failed:") { // Check if it's the specific error we added
			return "", fmt.Errorf("no common ancestor found between '%s' and '%s'", ref1, ref2)
		}
		return "", err // Other unexpected errors
	}
	return output, nil
}

// GetLocalBranches returns a list of local branch names.
func GetLocalBranches() ([]string, error) {
	// Using --format='%(refname:short)' is generally robust
	output, err := RunGitCommand("branch", "--list", "--format=%(refname:short)")
	if err != nil {
		return nil, fmt.Errorf("failed to list local branches: %w", err)
	}
	if output == "" {
		return []string{}, nil // No branches found
	}
	return strings.Split(output, "\n"), nil
}

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
