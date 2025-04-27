package gitutils

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp" // For parsing owner/repo
	"strconv"
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

func RunGitCommandInteractive(args ...string) error {
	cmd := exec.Command("git", args...) // Don't add --no-pager here

	// Connect standard streams directly
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command and wait for it to finish
	err := cmd.Run()
	if err != nil {
		// Unlike RunGitCommand, we don't capture output, so just return the error.
		// The user will have seen any error messages directly in their terminal.
		return fmt.Errorf("interactive git command failed: %w", err)
	}
	return nil
}

// StageInteractively runs `git add -p`.
func StageInteractively() error {
	err := RunGitCommandInteractive("add", "-p")
	if err != nil {
		return fmt.Errorf("failed during git add -p: %w", err)
	}
	return nil
}

// HasStagedChanges checks if there are any changes staged in the index.
// Uses `git diff --cached --quiet`. Exits 0 if no changes, 1 if changes.
func HasStagedChanges() (bool, error) {
	// --quiet makes it exit 0 if no diff, 1 if diff.
	// --cached (or --staged) compares index to HEAD.
	err := RunGitCommandInteractive("diff", "--cached", "--quiet")

	if err == nil {
		return false, nil // Exit code 0 means no staged changes
	}

	// Check if the error is specifically an ExitError with code 1
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return true, nil // Exit code 1 means there *are* staged changes
	}

	// Any other error is unexpected
	return false, fmt.Errorf("failed to check for staged changes: %w", err)
}

// HasUncommittedChanges checks if the git working directory or index has changes.
func HasUncommittedChanges() (bool, error) {
	// Keep original implementation
	output, err := RunGitCommand("status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("failed to run git status: %w", err)
	}
	return output != "", nil
}

// StageAllChanges runs `git add .` to stage all changes in the working directory.
func StageAllChanges() error {
	// Keep original implementation
	_, err := RunGitCommand("add", ".")
	if err != nil {
		return fmt.Errorf("failed to run git add .: %w", err)
	}
	return nil
}

// CommitChanges commits staged changes with the given message.
func CommitChanges(message string) error {
	// Use -m to provide message directly
	_, err := RunGitCommand("commit", "-m", message)
	if err != nil {
		// Commit can fail for various reasons (e.g., hooks, empty commit)
		// The error from RunGitCommand should include stderr which is helpful
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	return nil
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

// GetCurrentCommit returns the full commit hash of HEAD.
func GetCurrentCommit() (string, error) {
	output, err := RunGitCommand("rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current commit hash: %w", err)
	}
	return output, nil
}

// BranchExists checks if a local branch with the given name exists.
func BranchExists(name string) (bool, error) {
	// `git rev-parse --verify` is a reliable way to check existence.
	// It exits with 0 if the ref exists, non-zero otherwise.
	ref := fmt.Sprintf("refs/heads/%s", name)
	_, err := RunGitCommand("rev-parse", "--verify", ref)
	if err == nil {
		return true, nil // Exit code 0 means success (it exists)
	}
	// Check if the error is the specific "ref not found" error
	if strings.Contains(err.Error(), "ref not found (exit status 1)") {
		return false, nil // It doesn't exist, but this is not an unexpected error
	}
	// Any other error is unexpected
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

// RebaseUpdateRefs performs `git rebase <base> --update-refs`.
// Assumes the caller has checked out the correct tip branch.
func RebaseUpdateRefs(baseBranch string) error {
	// Run non-interactively first to capture potential errors clearly
	_, err := RunGitCommand("rebase", baseBranch, "--update-refs")
	if err != nil {
		// Don't return a specific conflict error type here.
		// The caller should check IsRebaseInProgress() if err != nil.
		return fmt.Errorf("git rebase --update-refs failed: %w", err)
	}
	return nil
}

// IsRebaseInProgress checks if a rebase operation is currently paused.
func IsRebaseInProgress() bool {
	// Keep the existing implementation using os.Stat on .git/rebase-*
	repoRoot, err := GetRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not get repo root to check rebase status: %v\n", err)
		return false
	}
	gitDir, err := RunGitCommand("rev-parse", "--git-dir")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not get git dir to check rebase status: %v\n", err)
		return false
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(repoRoot, gitDir)
	}
	rebaseApplyPath := filepath.Join(gitDir, "rebase-apply")
	rebaseMergePath := filepath.Join(gitDir, "rebase-merge")
	_, errApply := os.Stat(rebaseApplyPath)
	_, errMerge := os.Stat(rebaseMergePath)
	return errApply == nil || errMerge == nil
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
