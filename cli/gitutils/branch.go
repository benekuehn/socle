package gitutils

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// GetCurrentBranch (Keep the existing one)
func GetCurrentBranch() (string, error) {
	return RunGitCommand("rev-parse", "--abbrev-ref", "HEAD")
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

// CreateBranch creates a new branch pointing to a specific start point (commit hash or branch name).
func CreateBranch(name, startPoint string) error {
	_, err := RunGitCommand("branch", name, startPoint)
	if err != nil {
		return fmt.Errorf("failed to create branch '%s' from '%s': %w", name, startPoint, err)
	}
	return nil
}

// CheckoutBranch switches the working directory to the specified branch.
func CheckoutBranch(name string) error {
	// We assume the branch exists when calling this
	_, err := RunGitCommand("checkout", name)
	if err != nil {
		return fmt.Errorf("failed to checkout branch '%s': %w", name, err)
	}
	return nil
}

// BranchExists checks if a local branch with the given name exists.
func BranchExists(name string) (bool, error) {
	ref := fmt.Sprintf("refs/heads/%s", name)
	_, err := RunGitCommand("rev-parse", "--verify", ref)
	if err == nil {
		return true, nil // Command succeeded (exit 0)
	}

	// Check if the error is the specific non-zero exit code we expect for "not found"
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) { // Use errors.As for type assertion
		if exitErr.ExitCode() == 1 { // Exit code 1 for rev-parse --verify means not found
			return false, nil // Not found is not an *error* for this function's purpose
		}
	}

	// Any other error (or different exit code) is unexpected
	return false, fmt.Errorf("failed to verify branch existence for '%s': %w", name, err)
}

// IsValidBranchName checks if a string is a valid Git branch name.
func IsValidBranchName(name string) error {
	// `git check-ref-format` is the command for this.
	// It exits with 0 if valid, 1 if invalid.
	_, err := RunGitCommand("check-ref-format", "--branch", name)
	if err == nil {
		return nil // Exit code 0 means valid
	}
	// Check if the error is the specific "invalid format" error
	if strings.Contains(err.Error(), "invalid format (exit status 1)") {
		// Provide a slightly more user-friendly error message than the raw git output
		return fmt.Errorf("'%s' is not a valid branch name", name)
	}
	// Any other error is unexpected
	return fmt.Errorf("failed to validate branch name '%s': %w", name, err)
}

// BranchDelete force deletes a local branch. Used for cleanup.
func BranchDelete(name string) error {
	// Use -D for force delete
	_, err := RunGitCommand("branch", "-D", name)
	if err != nil {
		return fmt.Errorf("failed to delete branch '%s': %w", name, err)
	}
	return nil
}
