package gitutils

import (
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
)

// GetRemoteURL returns the fetch URL for a given remote.
func GetRemoteURL(remoteName string) (string, error) {
	output, err := RunGitCommand("remote", "get-url", remoteName)
	if err == nil {
		// Success path
		return output, nil
	}

	// Handle errors
	isNoSuchRemote := strings.Contains(err.Error(), "No such remote")

	// Check for the specific exit code 2 case
	isExitCode2 := false
	if exitErr, ok := err.(*exec.ExitError); ok { // Perform type assertion
		isExitCode2 = exitErr.ExitCode() == 2 // Check code if assertion worked
	}

	// Now check if either known "not found" condition is true
	if isNoSuchRemote || isExitCode2 {
		return "", fmt.Errorf("remote '%s' not found", remoteName)
	}

	// Otherwise, it's some other unexpected error
	return "", fmt.Errorf("failed to get URL for remote '%s': %w", remoteName, err)
}

// PushBranch pushes a local branch to a remote.
func PushBranch(branchName string, remoteName string, force bool) error {
	args := []string{"push"}
	if force {
		args = append(args, "--force")
		// Consider --force-with-lease later for more safety? Requires upstream info.
	}
	// Explicitly specify refspec to push local branch to remote branch of same name
	refspec := fmt.Sprintf("refs/heads/%s:refs/heads/%s", branchName, branchName)
	args = append(args, remoteName, refspec)

	// Push can output progress to stderr, use RunGitCommandInteractive for user feedback?
	// Or just use RunGitCommand and show stderr on error. Let's start simple.
	_, err := RunGitCommand(args...)
	if err != nil {
		return fmt.Errorf("failed to push branch '%s' to remote '%s': %w", branchName, remoteName, err)
	}
	return nil
}

// Regex to extract owner/repo from common Git URL formats
var repoUrlRegex = regexp.MustCompile(`(?::|/)([^/:]+)/([^/]+?)(?:\.git)?$`)

// ParseOwnerAndRepo extracts owner and repository name from a remote URL.
func ParseOwnerAndRepo(remoteUrl string) (owner string, repo string, err error) {
	// Handle SSH URLs like git@github.com:owner/repo.git
	if strings.HasPrefix(remoteUrl, "git@") {
		remoteUrl = strings.Replace(remoteUrl, ":", "/", 1)
	}

	// Use url.Parse for basic structure check, then regex for flexibility
	parsed, errParse := url.Parse(remoteUrl)
	if errParse != nil {
		return "", "", fmt.Errorf("failed to parse remote URL '%s': %w", remoteUrl, errParse)
	}

	matches := repoUrlRegex.FindStringSubmatch(parsed.Path) // Apply regex on path part
	if len(matches) != 3 {
		// Fallback: Try regex on the original string if path fails (e.g., SSH format after replace)
		matches = repoUrlRegex.FindStringSubmatch(remoteUrl)
		if len(matches) != 3 {
			return "", "", fmt.Errorf("could not extract owner/repo from URL: %s", remoteUrl)
		}
	}
	owner = matches[1]
	repo = matches[2]
	return owner, repo, nil
}

// FetchBranch updates a local branch from the default remote (origin).
func FetchBranch(branch string) error {
	// Assuming remote is 'origin'. Could make configurable later.
	remoteRef := fmt.Sprintf("origin/%s", branch)
	localRef := fmt.Sprintf("refs/heads/%s", branch)
	// Fetch directly into the local branch refspec
	_, err := RunGitCommand("fetch", "origin", fmt.Sprintf("%s:%s", remoteRef, localRef))
	// Handle cases where remote branch doesn't exist? Fetch might error.
	if err != nil {
		// Check for specific errors if needed, e.g., remote branch not found.
		return fmt.Errorf("failed to fetch '%s': %w", branch, err)
	}
	return nil
}
