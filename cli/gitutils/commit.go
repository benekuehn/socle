package gitutils

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
