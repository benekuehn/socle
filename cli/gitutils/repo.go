package gitutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

// GetGitVersion returns the major and minor version numbers.
func GetGitVersion() (major int, minor int, err error) {
	output, err := RunGitCommand("--version") // git --version
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get git version: %w", err)
	}
	// Output looks like "git version 2.40.1" or "git version 2.38.0.windows.1"
	parts := strings.Fields(output)
	if len(parts) < 3 || parts[0] != "git" || parts[1] != "version" {
		return 0, 0, fmt.Errorf("unexpected git version format: %s", output)
	}
	versionParts := strings.Split(parts[2], ".")
	if len(versionParts) < 2 {
		return 0, 0, fmt.Errorf("unexpected version number format in: %s", parts[2])
	}
	major, errMajor := strconv.Atoi(versionParts[0])
	minor, errMinor := strconv.Atoi(versionParts[1])
	if errMajor != nil || errMinor != nil {
		return 0, 0, fmt.Errorf("failed to parse version numbers from: %s", parts[2])
	}
	return major, minor, nil
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
