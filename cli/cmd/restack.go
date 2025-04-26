package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/benekuehn/so/gitutils"
	"github.com/benekuehn/so/internal/ui"
	"github.com/spf13/cobra"
)

// Flag variables
var (
	restackAbort   bool
	restackNoFetch bool
)

var restackCmd = &cobra.Command{
	Use:   "restack",
	Short: "Rebase the current stack onto the latest base branch using --update-refs",
	Long: `Updates the current stack of branches based on the latest version of the base branch.
Uses 'git rebase --update-refs' (requires Git >= 2.38) to rebase the whole stack.

If conflicts occur, the process stops, and you will be instructed to use standard
Git commands to resolve the rebase:
1. Resolve the conflicts (check 'git status').
2. Run 'git add .' for resolved files.
3. Run 'git rebase --continue'.
Repeat until the rebase is complete.

Use 'git rebase --abort' to cancel a conflicted rebase operation.
Use 'socle restack --abort' ONLY if you need to clear socle state from a PREVIOUS, FAILED version.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// --- Git Version Check ---
		major, minor, err := gitutils.GetGitVersion()
		if err != nil {
			return fmt.Errorf("failed to determine Git version: %w", err)
		}
		if major < 2 || (major == 2 && minor < 38) {
			return fmt.Errorf("socle restack requires Git version 2.38 or later for --update-refs support (you have %d.%d)", major, minor)
		}

		if restackAbort {
			if gitutils.IsRebaseInProgress() {
				fmt.Fprintln(os.Stderr, ui.Colors.WarningStyle.Render("Warning: A git rebase operation is currently in progress."))
				fmt.Fprintln(os.Stderr, ui.Colors.WarningStyle.Render("Run 'git rebase --abort' manually."))
			}
			fmt.Println("Restack aborted (no socle state to clear with this version). Use 'git rebase --abort' if needed.")
			return nil
		}

		// --- Pre-Checks ---
		if gitutils.IsRebaseInProgress() {
			return fmt.Errorf("a git rebase operation is already in progress.\nPlease resolve or abort it ('git rebase --continue' or 'git rebase --abort') before starting a new restack")
		}
		hasChanges, err := gitutils.HasUncommittedChanges()
		if err != nil {
			return fmt.Errorf("failed to check working tree status: %w", err)
		}
		if hasChanges {
			return fmt.Errorf("uncommitted changes detected. Please commit or stash them before restacking")
		}

		// --- Get Stack Info ---
		originalBranch, err := gitutils.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("cannot get current branch: %w", err)
		}

		_, stack, baseBranch, err := getCurrentStackInfo()
		if err != nil {
			return err
		}
		if len(stack) <= 1 {
			fmt.Println("Current branch is the base or directly on the base. Nothing to restack.")
			return nil
		}
		tipBranch := stack[len(stack)-1]

		// --- Defer Checkout Back ---
		defer func() {
			fmt.Printf("\nChecking out original branch '%s'...\n", originalBranch)
			errCheckout := gitutils.CheckoutBranch(originalBranch)
			if errCheckout != nil {
				fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("Warning: Failed to checkout original branch '%s': %v\n"), originalBranch, errCheckout)
			}
		}()

		// --- Fetch Base ---
		if !restackNoFetch {
			fmt.Printf("Fetching latest '%s' from origin...\n", baseBranch)
			if err := gitutils.FetchBranch(baseBranch); err != nil {
				return fmt.Errorf("failed to fetch base branch '%s': %w.\nUse --no-fetch to skip.", baseBranch, err)
			}
			fmt.Println(ui.Colors.SuccessStyle.Render("Fetch complete."))
		} else {
			fmt.Println("Skipping fetch for base branch (--no-fetch).")
		}

		// --- Execute Rebase ---
		fmt.Printf("Checking out tip branch '%s' to start rebase...\n", tipBranch)
		if err := gitutils.CheckoutBranch(tipBranch); err != nil {
			return fmt.Errorf("failed to checkout tip branch '%s': %w", tipBranch, err)
		}

		fmt.Printf("Running: git rebase %s --update-refs\n", baseBranch)
		err = gitutils.RebaseUpdateRefs(baseBranch)

		// --- Handle Result ---
		if err == nil {
			fmt.Println(ui.Colors.SuccessStyle.Render("\nRestack completed successfully!"))
			fmt.Println("Updated branches in stack:")
			for _, branchName := range stack {
				fmt.Printf("  - %s\n", branchName)
			}
			return nil // Success
		}

		if gitutils.IsRebaseInProgress() {
			fmt.Fprint(os.Stderr, ui.Colors.FailureStyle.Render("\nCONFLICT detected during rebase.\n"))
			fmt.Fprintln(os.Stderr, ui.Colors.FailureStyle.Render("Please resolve the conflicts using standard Git commands:"))
			fmt.Fprintln(os.Stderr, ui.Colors.FailureStyle.Render("  1. Fix conflicts (see 'git status')."))
			fmt.Fprintln(os.Stderr, ui.Colors.FailureStyle.Render("  2. Run 'git add .' for resolved files."))
			fmt.Fprintln(os.Stderr, ui.Colors.FailureStyle.Render("  3. Run 'git rebase --continue'."))
			fmt.Fprintln(os.Stderr, ui.Colors.FailureStyle.Render("Repeat steps 1-3 until the rebase completes."))
			fmt.Fprintln(os.Stderr, ui.Colors.FailureStyle.Render("Alternatively, run 'git rebase --abort' to cancel."))
			return fmt.Errorf("restack halted due to conflict. Resolve manually with Git")
		} else {
			// Rebase failed for a different reason
			fmt.Fprint(os.Stderr, ui.Colors.FailureStyle.Render("\nRebase command failed unexpectedly.\n"))
			// The error 'err' from RebaseUpdateRefs contains Git's output.
			return err
		}
	},
}

// Helper to get stack info (similar to show, but returns data)
func getCurrentStackInfo() (currentBranch string, stack []string, baseBranch string, err error) {
	currentBranch, err = gitutils.GetCurrentBranch()
	if err != nil {
		err = fmt.Errorf("failed to get current branch: %w", err)
		return
	}

	parentConfigKey := fmt.Sprintf("branch.%s.socle-parent", currentBranch)
	baseConfigKey := fmt.Sprintf("branch.%s.socle-base", currentBranch)
	_, errParent := gitutils.GetGitConfig(parentConfigKey)
	baseBranch, errBase := gitutils.GetGitConfig(baseConfigKey)

	isUntracked := false
	if errors.Is(errParent, os.ErrNotExist) || errors.Is(errBase, os.ErrNotExist) { // Assuming GetGitConfig returns os.ErrNotExist
		isUntracked = true
	} else if errParent != nil {
		err = fmt.Errorf("failed to read tracking parent for '%s': %w", currentBranch, errParent)
		return
	} else if errBase != nil {
		err = fmt.Errorf("failed to read tracking base for '%s': %w", currentBranch, errBase)
		return
	}

	knownBases := map[string]bool{"main": true, "master": true, "develop": true} // TODO: Configurable
	if isUntracked {
		if knownBases[currentBranch] {
			// On a base branch
			baseBranch = currentBranch
			stack = []string{baseBranch} // Stack is just the base
			err = nil                    // Not an error state
			return
		}
		err = fmt.Errorf("current branch '%s' is not tracked by socle. Run 'socle track' first", currentBranch)
		return
	}

	// Build stack by walking up parents (same logic as show)
	stack = []string{currentBranch}
	parent := ""             // Initialize parent
	current := currentBranch // Variable to track the branch whose parent we are looking for

	for {
		// Read parent config for the 'current' branch in the walk-up
		currentParentKey := fmt.Sprintf("branch.%s.socle-parent", current)
		parent, err = gitutils.GetGitConfig(currentParentKey)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) { // Reached an untracked branch before base?
				err = fmt.Errorf("tracking information broken at branch '%s' (parent not found). Cannot determine stack", current)
				return
			}
			err = fmt.Errorf("failed to read tracking parent for '%s': %w", current, err)
			return
		}

		if parent == baseBranch {
			break // Reached base
		}
		stack = append([]string{parent}, stack...) // Prepend parent
		current = parent                           // Move up for the next iteration

		if len(stack) > 50 { // Safety break
			err = fmt.Errorf("stack trace exceeds 50 levels, assuming cycle or error")
			return
		}
	}
	stack = append([]string{baseBranch}, stack...) // Prepend the final base branch

	return currentBranch, stack, baseBranch, nil
}

func init() {
	restackCmd.Flags().BoolVar(&restackAbort, "abort", false, "Abort an in-progress *git* rebase (clears potential old socle state)")
	restackCmd.Flags().BoolVar(&restackNoFetch, "no-fetch", false, "Skip fetching the remote base branch")
}
