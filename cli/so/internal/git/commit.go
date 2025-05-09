package git

import (
	"fmt"
	"strings"
)

// GetCurrentCommit returns the full commit hash of HEAD.
func GetCurrentCommit() (string, error) {
	output, err := RunGitCommand("rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current commit hash: %w", err)
	}
	return output, nil
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

// GetFirstCommitSubject returns the subject line of the first commit unique to branchRef compared to parentRef.
// Returns empty string if no unique commits found or on error getting log.
func GetFirstCommitSubject(parentRef, branchRef string) (string, error) {
	// --reverse lists oldest first
	// %s gives just the subject
	// Use parent..HEAD if branchRef is the current branch and parentRef is its direct tracked parent?
	// No, stick to explicit range for clarity.
	logRange := fmt.Sprintf("%s..%s", parentRef, branchRef)
	output, err := RunGitCommand("log", "--reverse", "--format=%s", logRange)
	if err != nil {
		// Log command itself might fail if refs are invalid, etc.
		// Return empty string and the error for the caller to handle/warn.
		return "", fmt.Errorf("failed to get log for range '%s': %w", logRange, err)
	}
	if output == "" {
		// No unique commits found in the range
		return "", nil // Return empty, no error
	}
	// Split into lines and take the first one
	lines := strings.SplitN(output, "\n", 2) // Split only once to get the first line
	subject := strings.TrimSpace(lines[0])
	return subject, nil
}

// GetCurrentBranchCommit returns the full commit hash for the tip of a specific local branch.
func GetCurrentBranchCommit(branchName string) (string, error) {
	// Ensure we are asking for the local branch ref
	ref := fmt.Sprintf("refs/heads/%s", branchName)
	// Use rev-parse without --verify, as we expect the branch to exist
	// when this is called in the restack loop.
	output, err := RunGitCommand("rev-parse", ref)
	if err != nil {
		// This error is more serious than BranchExists failing, as we expect
		// the branch (parent or current) to exist during the restack loop.
		// The error from RunGitCommand will include stderr detail.
		return "", fmt.Errorf("failed to get commit hash for branch '%s' (ref: '%s'): %w", branchName, ref, err)
	}
	// Check for empty output just in case rev-parse succeeds but returns nothing
	if output == "" {
		return "", fmt.Errorf("git rev-parse for branch '%s' succeeded but returned empty output", branchName)
	}
	return output, nil
}

// GetMultipleBranchCommits returns the full commit hashes for the tips of multiple specific local branches.
// It runs a single `git rev-parse` command for efficiency.
// Returns a map of branchName -> commitHash and an error if any occurred.
func GetMultipleBranchCommits(branchNames []string) (map[string]string, error) {
	if len(branchNames) == 0 {
		return make(map[string]string), nil
	}

	args := []string{"rev-parse"}
	for _, branchName := range branchNames {
		// Ensure we are asking for the local branch ref, e.g., refs/heads/mybranch
		// However, git rev-parse typically resolves simple branch names correctly.
		// For robustness with GetCurrentBranchCommit, it uses refs/heads/. We can just pass branchName.
		args = append(args, branchName)
	}

	output, err := RunGitCommand(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit hashes for branches %v: %w", branchNames, err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != len(branchNames) {
		return nil, fmt.Errorf("git rev-parse returned %d lines, expected %d for branches %v. Output: %s", len(lines), len(branchNames), branchNames, output)
	}

	commitMap := make(map[string]string)
	for i, branchName := range branchNames {
		commitMap[branchName] = strings.TrimSpace(lines[i])
	}

	return commitMap, nil
}
