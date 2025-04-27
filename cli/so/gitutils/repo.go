package gitutils

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetRepoRoot (Keep the existing one)
func GetRepoRoot() (string, error) {
	return RunGitCommand("rev-parse", "--show-toplevel")
}

// IsGitRepo (Keep the existing one)
func IsGitRepo() bool {
	_, err := RunGitCommand("rev-parse", "--is-inside-work-tree")
	return err == nil
}

var prTemplatePaths = []string{
	".github/pull_request_template.md",
	".github/PULL_REQUEST_TEMPLATE.md",
	"pull_request_template.md",
	"docs/pull_request_template.md", // Older, less common
}

// FindAndReadPRTemplate searches for a PR template file and reads its content.
func FindAndReadPRTemplate() (string, error) {
	repoRoot, err := GetRepoRoot()
	if err != nil {
		return "", fmt.Errorf("cannot find repo root to search for PR template: %w", err)
	}

	for _, relPath := range prTemplatePaths {
		absPath := filepath.Join(repoRoot, relPath)
		_, err := os.Stat(absPath)
		if err == nil {
			// File exists
			contentBytes, errRead := os.ReadFile(absPath)
			if errRead != nil {
				// File exists but couldn't read it - return error
				return "", fmt.Errorf("failed to read PR template '%s': %w", relPath, errRead)
			}
			fmt.Printf("Using PR template: %s\n", relPath) // Inform user
			return string(contentBytes), nil
		}
		if !os.IsNotExist(err) {
			// Error other than "not found" during stat - return error
			return "", fmt.Errorf("error checking for PR template '%s': %w", relPath, err)
		}
		// If os.IsNotExist, continue to the next path
	}

	// No template found after checking all paths
	return "", nil // Return empty string, not an error
}
