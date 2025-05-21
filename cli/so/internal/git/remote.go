package git

import (
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strings"

	"errors"

	"github.com/benekuehn/socle/cli/so/internal/ui"
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

// FetchBranch updates the remote-tracking branch for a given local branch
// from the specified remote (e.g., fetch 'origin' to update 'origin/master').
func FetchBranch(branchName string, remoteName string) error {
	fmt.Printf("  (Running: git fetch %s)\n", remoteName)
	_, err := RunGitCommand("fetch", remoteName)
	if err != nil {
		return fmt.Errorf("failed to fetch remote '%s': %w", remoteName, err)
	}

	// Optionally, after fetching, explicitly update the *local* branch
	// to match the newly fetched remote-tracking branch. This makes sure
	// the subsequent rebase uses the absolute latest code.
	// Use merge --ff-only which is safe (only fast-forwards).
	// Need to checkout the branch first.
	currentBranch, cbErr := GetCurrentBranch()
	if cbErr != nil {
		return fmt.Errorf("fetch successful, but failed to get current branch to restore later: %w", cbErr)
	}
	if currentBranch != branchName {
		fmt.Printf("  Checking out '%s' to update from remote...\n", branchName)
		errCheckout := CheckoutBranch(branchName)
		if errCheckout != nil {
			return fmt.Errorf("fetch successful, but failed to checkout '%s' to update: %w", branchName, errCheckout)
		}
		// Defer switching back
		defer func() {
			fmt.Printf("  Switching back to %s...\n", currentBranch)
			_ = CheckoutBranch(currentBranch) // Ignore error on cleanup
		}()
	}

	remoteTrackingBranch := fmt.Sprintf("%s/%s", remoteName, branchName)
	fmt.Printf("  Attempting fast-forward merge for '%s' from '%s'...\n", branchName, remoteTrackingBranch)
	_, errMerge := RunGitCommand("merge", "--ff-only", remoteTrackingBranch)
	if errMerge != nil {
		// If ff-only fails, it means local branch has diverged or remote tracking branch wasn't updated correctly.
		// Should we warn or error? Let's warn and proceed, the rebase will likely fail anyway if needed.
		fmt.Println(ui.Colors.WarningStyle.Render(fmt.Sprintf("  Warning: Could not fast-forward local '%s'. It may have diverged from '%s'. Rebase will use local version.", branchName, remoteTrackingBranch)))
	} else {
		fmt.Println(ui.Colors.SuccessStyle.Render(fmt.Sprintf("  Local branch '%s' updated.", branchName)))
	}

	return nil
}

// PushBranchWithLease pushes a local branch to a remote using --force-with-lease.
// This is safer than --force as it checks if the remote ref hasn't changed unexpectedly.
func PushBranchWithLease(branchName string, remoteName string) error {
	args := []string{"push", "--force-with-lease"}

	// Explicitly specify refspec for clarity and safety
	refspec := fmt.Sprintf("refs/heads/%s:refs/heads/%s", branchName, branchName)
	args = append(args, remoteName, refspec)

	// Push can output progress to stderr, RunGitCommand handles capturing stderr on error.
	_, err := RunGitCommand(args...)
	if err != nil {
		// Push failures (especially with --force-with-lease) often have informative
		// messages in stderr, which RunGitCommand includes in the error.
		return fmt.Errorf("failed to push branch '%s' with lease to remote '%s': %w", branchName, remoteName, err)
	}
	return nil
}

// FetchAll fetches all branches from the specified remote
func FetchAll(remoteName string) error {
	_, err := RunGitCommand("fetch", remoteName)
	if err != nil {
		return fmt.Errorf("failed to fetch from remote '%s': %w", remoteName, err)
	}
	return nil
}

// DeleteBranch deletes a local branch
func DeleteBranch(branchName string) error {
	_, err := RunGitCommand("branch", "-D", branchName)
	if err != nil {
		return fmt.Errorf("failed to delete branch '%s': %w", branchName, err)
	}
	return nil
}

// ErrNotFastForward is returned when a branch cannot be fast-forwarded
var ErrNotFastForward = errors.New("branch cannot be fast-forwarded")

// FastForwardBranch attempts to fast-forward a branch to match its remote tracking branch
func FastForwardBranch(branchName, remoteName string) error {
	// First check if we can fast-forward
	_, err := RunGitCommand("merge-base", "--is-ancestor", branchName, fmt.Sprintf("%s/%s", remoteName, branchName))
	if err != nil {
		return ErrNotFastForward
	}

	// If we get here, we can fast-forward
	_, err = RunGitCommand("checkout", branchName)
	if err != nil {
		return fmt.Errorf("failed to checkout branch '%s': %w", branchName, err)
	}

	_, err = RunGitCommand("merge", "--ff-only", fmt.Sprintf("%s/%s", remoteName, branchName))
	if err != nil {
		return fmt.Errorf("failed to fast-forward branch '%s': %w", branchName, err)
	}

	return nil
}

// ForceUpdateBranch forces a branch to match its remote tracking branch
func ForceUpdateBranch(branchName, remoteName string) error {
	_, err := RunGitCommand("checkout", branchName)
	if err != nil {
		return fmt.Errorf("failed to checkout branch '%s': %w", branchName, err)
	}

	_, err = RunGitCommand("reset", "--hard", fmt.Sprintf("%s/%s", remoteName, branchName))
	if err != nil {
		return fmt.Errorf("failed to force update branch '%s': %w", branchName, err)
	}

	return nil
}

// SetBranchParent updates the tracking information for a branch to point to a new parent.
func SetBranchParent(branchName, parentName string) error {
	cmd := exec.Command("git", "branch", "--set-upstream-to="+parentName, branchName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set parent for branch '%s' to '%s': %w", branchName, parentName, err)
	}
	return nil
}

// SwitchBranch switches to the specified branch
func SwitchBranch(branch string) error {
	cmd := exec.Command("git", "switch", branch)
	return cmd.Run()
}

// UpdateBranchParent updates both the tracking and parent configuration for a branch
func UpdateBranchParent(branchName, parentName string) error {
	// Update tracking information
	if err := SetBranchParent(branchName, parentName); err != nil {
		return fmt.Errorf("failed to set tracking for branch '%s' to '%s': %w", branchName, parentName, err)
	}

	// Update parent configuration
	parentConfigKey := fmt.Sprintf("branch.%s.socle-parent", branchName)
	cmd := exec.Command("git", "config", parentConfigKey, parentName)
	if err := cmd.Run(); err != nil {
		// If setting parent fails, try to restore tracking
		_ = SetBranchParent(branchName, parentName)
		return fmt.Errorf("failed to set parent configuration for branch '%s' to '%s': %w", branchName, parentName, err)
	}

	return nil
}
