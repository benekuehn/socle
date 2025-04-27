package gitutils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

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
