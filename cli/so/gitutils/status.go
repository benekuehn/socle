package gitutils

import (
	"errors"
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
	// Use RunGitCommand as no interaction is needed, just the exit code.
	_, err := RunGitCommand("diff", "--cached", "--quiet") // Use non-interactive runner

	if err == nil {
		return false, nil // Exit code 0 means no staged changes
	}

	// Check if the error indicates exit code 1 (which means changes *were* found)
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) { // Use errors.As to properly unwrap
		if exitErr.ExitCode() == 1 {
			// Exit code 1 from "git diff --quiet" means "differences found".
			// This is success for our check, so return true and nil error.
			return true, nil
		}
	}

	// Any other error (or different exit code) is unexpected
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

// ErrRebaseConflict indicates a git rebase operation stopped due to conflicts.
var ErrRebaseConflict = errors.New("rebase conflict detected")

// RebaseCurrentBranchOnto performs `git rebase <newBaseOID>` on the currently checked-out branch.
// It specifically checks for conflicts upon failure using IsRebaseInProgress.
func RebaseCurrentBranchOnto(newBaseOID string) error {
	// Run the simple rebase command for the current branch
	// Pass the specific commit hash as the <newbase>
	_, err := RunGitCommand("rebase", newBaseOID)

	if err == nil {
		return nil // Success
	}

	// Check if the failure left us in a conflict state
	if IsRebaseInProgress() {
		// Return the specific sentinel error
		return ErrRebaseConflict
	}

	// Otherwise, it was some other unexpected rebase error.
	// The error 'err' from RunGitCommand already contains context (stderr, exit code).
	return fmt.Errorf("git rebase onto '%s' failed: %w", newBaseOID, err)
}

// HasDiff checks if there are differences between two refs (e.g., parent..branch).
// Uses `git diff --quiet <ref1>..<ref2>`. Exits 0 if no changes, 1 if changes.
func HasDiff(ref1, ref2 string) (bool, error) {
	diffRange := fmt.Sprintf("%s..%s", ref1, ref2)
	// --quiet makes it exit 0 if no diff, 1 if diff.
	_, err := RunGitCommand("diff", "--quiet", diffRange)

	if err == nil {
		return false, nil // Exit code 0 means no differences
	}

	// Check if the error indicates exit code 1 (which means differences *were* found)
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() == 1 {
			// Exit code 1 from "git diff --quiet" means "differences found".
			return true, nil // Report true (diff exists), nil error
		}
	}

	// Any other error (or different exit code) is unexpected
	return false, fmt.Errorf("failed to check diff for range '%s': %w", diffRange, err)
}

func NeedsRestack(parentBranchName, childBranchName string) (needsRestack bool, err error) {
	// 1. Get the current commit hash (OID) of the parent branch
	parentOID, err := GetCurrentBranchCommit(parentBranchName)
	if err != nil {
		// If we can't get the parent's commit, we can't determine status reliably
		return false, fmt.Errorf("failed to get current commit for parent '%s': %w", parentBranchName, err)
	}
	if parentOID == "" { // Should not happen if GetCurrentBranchCommit works, but safeguard
		return false, fmt.Errorf("internal error: got empty commit OID for parent '%s'", parentBranchName)
	}

	// 2. Get the merge-base between the parent and the child branch
	mergeBase, err := GetMergeBase(parentBranchName, childBranchName)
	if err != nil {
		// If merge-base fails (e.g., unrelated histories, though unlikely in a stack),
		// consider it as needing restack? Or return error?
		// Let's return an error for now, as it indicates a broken state.
		// The error from GetMergeBase should be informative.
		return false, fmt.Errorf("failed to find merge base between '%s' and '%s': %w", parentBranchName, childBranchName, err)
	}
	if mergeBase == "" { // Should not happen if GetMergeBase works, but safeguard
		return false, fmt.Errorf("internal error: got empty merge base between '%s' and '%s'", parentBranchName, childBranchName)
	}

	// 3. Compare the merge-base with the parent's current OID
	// If the merge-base is *not* the same as the parent's current tip,
	// it means the parent has moved forward since the child was based on it,
	// OR the child was based on something else entirely. Either way, a rebase is needed.
	needsRestack = (mergeBase != parentOID)

	return needsRestack, nil
}
