package gitutils

import (
	"errors"
	"fmt"
	"log/slog"
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
		if errors.As(err, &exitErr) {
			// Exit code 1 is the documented code for "not found".
			// Exit code 128 on user's system also indicates ref cannot be resolved.
			if exitErr.ExitCode() == 1 || exitErr.ExitCode() == 128 {
				return false, nil // Ref not found, treat as non-error for this function
			}
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

// GetFullStackForSubmit determines the complete stack of branches to be processed,
// starting from the base of the current stack and including all its descendants.
// It takes the current stack (from base to the currently checked-out branch) as input.
// It returns the full stack (including the base) and the map of all parent relationships.
func GetFullStackForSubmit(currentStack []string) ([]string, map[string]string, error) {
	slog.Debug("Determining full stack...")
	// Need all parent relationships to build the descendant tree accurately
	allParents, err := GetAllSocleParents()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read all tracking relationships: %w", err)
	}

	// Handle the edge case where currentStack might be empty or just the base
	if len(currentStack) <= 1 {
		slog.Debug("Current stack is empty or only contains the base.", "currentStack", currentStack)
		// Return the current stack itself and the parents map, as there are no descendants to find.
		// The caller (e.g., submit) should handle the case of len <= 1 appropriately.
		return currentStack, allParents, nil
	}

	childMap := BuildChildMap(allParents)
	// Find all descendants of the *last* branch in the current stack
	// This assumes the current checkout is somewhere within the stack of interest
	tipOfCurrentStack := currentStack[len(currentStack)-1]
	descendants := FindAllDescendants(tipOfCurrentStack, childMap)

	// Combine the current stack (base -> current -> tip) with any further descendants
	fullStack := currentStack
	processedDescendants := make(map[string]bool) // Avoid adding duplicates if currentStack already had some
	for _, b := range currentStack {
		processedDescendants[b] = true
	}

	// Simple append - order might not be perfect topological, but contains all nodes
	// TODO: Improve ordering if needed later (e.g., topological sort)
	for _, desc := range descendants {
		if !processedDescendants[desc] {
			fullStack = append(fullStack, desc)
			processedDescendants[desc] = true
		}
	}
	slog.Debug("Full stack identified for processing:", "fullStack", fullStack)

	// Return the combined stack and the parent map needed by the caller
	return fullStack, allParents, nil
}
