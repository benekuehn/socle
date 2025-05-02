// cli/so/cmd/restack.go
package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2" // For push prompt
	"github.com/benekuehn/socle/cli/so/gitutils"
	"github.com/benekuehn/socle/cli/so/internal/ui"
	"github.com/spf13/cobra"
)

// Flag variables
var (
	restackNoFetch   bool
	restackForcePush bool // Explicit flag to force push without prompt
	restackNoPush    bool // Explicit flag to disable push entirely
)

var restackCmd = &cobra.Command{
	Use:   "restack",
	Short: "Rebase the current stack onto the latest base branch",
	Long: `Updates the current stack by rebasing each branch sequentially onto its updated parent.
Handles remote 'origin' automatically.

Process:
1. Checks for clean state & existing Git rebase.
2. Fetches the base branch from 'origin' (unless --no-fetch).
3. Rebases each branch in the stack onto the latest commit of its parent.
   - Skips branches that are already up-to-date.
4. If conflicts occur:
   - Stops and instructs you to use standard Git commands (status, add, rebase --continue / --abort).
   - Run 'so restack' again after resolving or aborting the Git rebase.
5. If successful:
   - Prompts to force-push updated branches to 'origin' (use --force-push or --no-push to skip prompt).`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// --- Pre-Checks ---
		if gitutils.IsRebaseInProgress() {
			// Changed wording: Guide user on what to do *now*
			fmt.Fprintln(os.Stderr, ui.Colors.InfoStyle.Render("Git rebase already in progress."))
			fmt.Fprintln(os.Stderr, ui.Colors.InfoStyle.Render("Resolve conflicts and run 'git rebase --continue' or cancel with 'git rebase --abort'."))
			fmt.Fprintln(os.Stderr, ui.Colors.InfoStyle.Render("Once the Git rebase is finished, run 'so restack' again if needed."))
			cmd.SilenceUsage = true // Prevent usage printing on clean exit
			return nil              // Exit cleanly, user needs to act in Git
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
		_, stack, baseBranch, err := getCurrentStackInfo() // Assumes this helper exists and is correct
		if err != nil {
			return err
		}
		if len(stack) <= 1 {
			fmt.Println("Current branch is the base or directly on base. Nothing to restack.")
			return nil
		}

		// --- Defer Checkout Back ---
		defer func() {
			// Only run if no rebase is currently in progress (i.e., we didn't exit due to conflict)
			if !gitutils.IsRebaseInProgress() {
				currentBranchAfter, _ := gitutils.GetCurrentBranch()
				if currentBranchAfter != originalBranch {
					fmt.Printf("\nChecking out original branch '%s'...\n", originalBranch)
					errCheckout := gitutils.CheckoutBranch(originalBranch)
					if errCheckout != nil {
						fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("Warning: Failed to checkout original branch '%s': %v\n"), originalBranch, errCheckout)
					}
				}
			}
		}()

		// --- Fetch Base (with remote check) ---
		remoteName := "origin"
		shouldFetch := !restackNoFetch
		if shouldFetch {
			_, errRemote := gitutils.GetRemoteURL(remoteName)
			if errRemote != nil {
				if strings.Contains(errRemote.Error(), "not found") {
					fmt.Printf("Remote '%s' not found. Skipping fetch.\n", remoteName)
					shouldFetch = false
				} else {
					return fmt.Errorf("failed to check remote '%s': %w", remoteName, errRemote)
				}
			}
		}
		if shouldFetch {
			fmt.Printf("Fetching latest '%s' from %s...\n", baseBranch, remoteName)
			// Pass remote name to FetchBranch if it needs it
			if err := gitutils.FetchBranch(baseBranch, remoteName); err != nil {
				return fmt.Errorf("failed to fetch base branch '%s': %w.\nUse --no-fetch to skip.", baseBranch, err)
			}
			fmt.Println(ui.Colors.SuccessStyle.Render("Fetch complete."))
		} else if restackNoFetch {
			fmt.Println("Skipping fetch (--no-fetch).")
		}

		// --- Iterative Rebase Loop ---
		fmt.Println("\n--- Starting Stack Rebase ---")
		rebasedBranches := []string{} // Keep track of branches we actually rebased/checked

		for i := 1; i < len(stack); i++ {
			branch := stack[i]
			parent := stack[i-1]

			fmt.Printf("\n[%d/%d] Processing branch: %s (parent: %s)\n", i, len(stack)-1, ui.Colors.UserInputStyle.Render(branch), ui.Colors.UserInputStyle.Render(parent))

			// Get current OIDs
			parentOID, errPO := gitutils.GetCurrentBranchCommit(parent)
			if errPO != nil {
				return fmt.Errorf("cannot get current commit of parent '%s': %w", parent, errPO)
			}

			// Optimization Check
			mergeBase, errMB := gitutils.GetMergeBase(parent, branch)
			if errMB != nil {
				// If merge-base fails, maybe the branches have diverged significantly?
				// Warn and proceed with rebase attempt.
				fmt.Println(ui.Colors.WarningStyle.Render(fmt.Sprintf("  Warning: Could not find merge base between '%s' and '%s': %v. Attempting rebase anyway.", parent, branch, errMB)))
			} else if mergeBase == parentOID {
				fmt.Println(ui.Colors.InfoStyle.Render(fmt.Sprintf("  Branch '%s' is already based on current '%s'. Skipping rebase.", branch, parent)))
				rebasedBranches = append(rebasedBranches, branch) // Add to list even if skipped, as it's confirmed correct
				continue                                          // Skip to next branch
			}

			// Checkout and Rebase
			fmt.Printf("  Checking out '%s'...\n", branch)
			if err := gitutils.CheckoutBranch(branch); err != nil {
				return fmt.Errorf("failed to checkout branch '%s' for rebase: %w", branch, err)
			}

			fmt.Printf("  Rebasing onto '%s' (%s)...\n", parent, parentOID[:7])
			err = gitutils.RebaseCurrentBranchOnto(parentOID) // Rebase onto specific parent commit OID

			if err == nil {
				fmt.Println(ui.Colors.SuccessStyle.Render("  Rebase step successful."))
				rebasedBranches = append(rebasedBranches, branch) // Track success
				continue                                          // Success, move to next branch
			}

			// Handle Rebase Failure
			if errors.Is(err, gitutils.ErrRebaseConflict) {
				// CONFLICT Case
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, ui.Colors.WarningStyle.Render("⚠️ Rebase paused due to conflicts."))
				fmt.Fprintln(os.Stderr, "   Resolve conflicts manually:")
				fmt.Fprintln(os.Stderr, "     1. Edit conflicted files (see 'git status').")
				fmt.Fprintln(os.Stderr, "     2. Run 'git add <resolved-files...>'.")
				fmt.Fprintln(os.Stderr, "     3. Run 'git rebase --continue'.")
				fmt.Fprintln(os.Stderr, "   (To cancel, run 'git rebase --abort')")
				fmt.Fprintln(os.Stderr, "   Once the Git rebase is complete, run 'so restack' again.")

				rerereEnabled, errCheck := gitutils.IsRerereEnabled()
				if errCheck != nil {
					// Don't fail the process for this, just warn
					fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("   Warning: Could not check git rerere config: %v\n"), errCheck)
				} else if !rerereEnabled {
					fmt.Fprintln(os.Stderr, "") // Extra space before tip
					fmt.Fprintln(os.Stderr, ui.Colors.InfoStyle.Render(
						"   Tip: Enable 'git rerere' globally ('git config --global rerere.enabled true')"))
					fmt.Fprintln(os.Stderr, ui.Colors.InfoStyle.Render(
						"        to potentially automate resolving similar conflicts in the future."))
				}

				cmd.SilenceUsage = true // Prevent usage printing
				return nil              // Exit cleanly, user needs to use Git
			}

			// Other Unexpected Rebase Failure
			return fmt.Errorf("unexpected error during rebase of '%s': %w", branch, err)

		} // End of loop

		// --- Post-Success ---
		fmt.Println(ui.Colors.SuccessStyle.Render("\n--- Stack Rebase Completed Successfully ---"))

		// Determine if push is desired
		doPush := false
		if restackForcePush {
			doPush = true
			fmt.Println("Force pushing specified via --force-push flag.")
		} else if restackNoPush {
			doPush = false
			fmt.Println("Pushing disabled via --no-push flag.")
		} else {
			// Prompt user if flags don't decide
			confirmPush := false
			prompt := &survey.Confirm{
				Message: fmt.Sprintf("Force push %d successfully rebased/checked branches to remote '%s'?", len(rebasedBranches), remoteName),
				Default: false, // Default to NO for safety
			}
			err := survey.AskOne(prompt, &confirmPush, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
			if err != nil { /* Handle prompt error or interrupt if desired */
				fmt.Fprintf(os.Stderr, "Push prompt failed: %v. Skipping push.\n", err)
			}
			doPush = confirmPush
		}

		// Execute push if needed
		if doPush && len(rebasedBranches) > 0 {
			fmt.Printf("\n--- Force Pushing Updated Branches to '%s' ---\n", remoteName)
			pushSuccessCount := 0
			for _, branch := range rebasedBranches {
				fmt.Printf("Pushing %s... ", branch)
				err := gitutils.PushBranchWithLease(branch, remoteName) // Use force-with-lease
				if err != nil {
					fmt.Println(ui.Colors.FailureStyle.Render("Failed!"))
					// Log error but continue trying other branches? Or abort?
					fmt.Fprintf(os.Stderr, "  Error pushing %s: %v\n", branch, err)
					// Let's continue for now
				} else {
					fmt.Println(ui.Colors.SuccessStyle.Render("Success."))
					pushSuccessCount++
				}
			}
			fmt.Printf("Finished pushing %d branches.\n", pushSuccessCount)
		} else if len(rebasedBranches) == 0 {
			fmt.Println("No branches were identified as needing rebase or push.")
		} else {
			fmt.Println("Skipping push.")
		}

		return nil // Overall success
	},
}

// Helper to get stack info (base, ordered stack list, current)
func getCurrentStackInfo() (currentBranch string, stack []string, baseBranch string, err error) {
	// 1. Get Current Branch
	currentBranch, err = gitutils.GetCurrentBranch()
	if err != nil {
		err = fmt.Errorf("failed to get current branch: %w", err)
		return // Return zero values for others
	}

	// 2. Check if current branch is tracked and get its base
	parentConfigKey := fmt.Sprintf("branch.%s.socle-parent", currentBranch)
	baseConfigKey := fmt.Sprintf("branch.%s.socle-base", currentBranch)

	_, errParent := gitutils.GetGitConfig(parentConfigKey)
	baseBranch, errBase := gitutils.GetGitConfig(baseConfigKey)

	// Check for specific "not found" errors from GetGitConfig
	// (Assuming GetGitConfig now returns an error containing "not found" text)
	isParentNotFound := errParent != nil && strings.Contains(errParent.Error(), "not found")
	isBaseNotFound := errBase != nil && strings.Contains(errBase.Error(), "not found")

	isUntracked := isParentNotFound || isBaseNotFound

	// Handle other unexpected errors during config reading
	if errParent != nil && !isParentNotFound {
		err = fmt.Errorf("failed to read tracking parent for '%s': %w", currentBranch, errParent)
		return
	}
	if errBase != nil && !isBaseNotFound {
		err = fmt.Errorf("failed to read tracking base for '%s': %w", currentBranch, errBase)
		return
	}

	// Check if we are actually on a known base branch
	// TODO: Make base branches configurable instead of hardcoded map
	knownBases := map[string]bool{"main": true, "master": true, "develop": true}
	if knownBases[currentBranch] {
		baseBranch = currentBranch
		stack = []string{baseBranch} // Stack is just the base itself
		err = nil                    // Not an error state
		return
	}

	// If it's not a known base branch AND tracking info is missing
	if isUntracked {
		err = fmt.Errorf("current branch '%s' is not tracked by socle and is not a known base branch.\nRun 'so track' on this branch first", currentBranch)
		return
	}

	// If we reach here, the branch is tracked and is not the base branch itself.
	// BaseBranch variable holds the correct base name from config.

	// 3. Build the stack by walking up the parents
	stack = []string{currentBranch} // Start with the current branch
	current := currentBranch        // Start the walk from the current branch

	for {
		// Get the parent of the 'current' branch in the walk-up
		currentParentKey := fmt.Sprintf("branch.%s.socle-parent", current)
		parent, parentErr := gitutils.GetGitConfig(currentParentKey)

		if parentErr != nil {
			// If we can't find the parent config for an intermediate branch, the tracking is broken
			if strings.Contains(parentErr.Error(), "not found") {
				err = fmt.Errorf("tracking information broken: parent branch config key '%s' not found for branch '%s'. Cannot determine stack", currentParentKey, current)
			} else {
				err = fmt.Errorf("failed to read tracking parent for intermediate branch '%s': %w", current, parentErr)
			}
			return // Return empty stack/base and the error
		}

		// Prepend the found parent to the stack slice
		stack = append([]string{parent}, stack...)

		// Check if the parent we just added is the base branch
		if parent == baseBranch {
			break // We've reached the base, stack is complete
		}

		// Move up for the next iteration
		current = parent

		// Safety break to prevent infinite loops in case of cyclic metadata
		if len(stack) > 50 { // Arbitrary limit
			err = fmt.Errorf("stack trace exceeds 50 levels, assuming cycle or error in tracking metadata")
			return // Return empty stack/base and the error
		}
	} // End of for loop

	// Stack is now built correctly from base to currentBranch
	return currentBranch, stack, baseBranch, nil // Success
}

func init() {
	AddCommand(restackCmd)
	restackCmd.Flags().BoolVar(&restackNoFetch, "no-fetch", false, "Skip fetching the remote base branch")
	restackCmd.Flags().BoolVar(&restackForcePush, "force-push", false, "Force push rebased branches without prompting")
	restackCmd.Flags().BoolVar(&restackNoPush, "no-push", false, "Do not push branches after successful rebase")
	// Flags that decide push behavior are mutually exclusive
	restackCmd.MarkFlagsMutuallyExclusive("force-push", "no-push")
}
