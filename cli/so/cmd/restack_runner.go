package cmd

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/benekuehn/socle/cli/so/gitutils"
	"github.com/benekuehn/socle/cli/so/internal/ui"
	"github.com/spf13/cobra"
)

type restackCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader // For push prompt

	// Config flags
	noFetch   bool
	forcePush bool
	noPush    bool
}

func (r *restackCmdRunner) run(cmd *cobra.Command) error {
	// --- Pre-Checks ---
	if gitutils.IsRebaseInProgress() {
		fmt.Fprintln(r.stderr, ui.Colors.InfoStyle.Render("Git rebase already in progress."))
		fmt.Fprintln(r.stderr, ui.Colors.InfoStyle.Render("Resolve conflicts and run 'git rebase --continue' or cancel with 'git rebase --abort'."))
		fmt.Fprintln(r.stderr, ui.Colors.InfoStyle.Render("Once the Git rebase is finished, run 'so restack' again if needed."))
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
	_, stack, baseBranch, err := gitutils.GetCurrentStackInfo()
	if err != nil {
		return err
	}
	if len(stack) <= 1 {
		fmt.Fprintln(r.stdout, "Current branch is the base or directly on base. Nothing to restack.")
		return nil
	}

	// --- Defer Checkout Back ---
	defer func() {
		// Only run if no rebase is currently in progress (i.e., we didn't exit due to conflict)
		if !gitutils.IsRebaseInProgress() {
			currentBranchAfter, _ := gitutils.GetCurrentBranch()
			if currentBranchAfter != originalBranch {
				r.logger.Debug("Checking out original branch", "name", originalBranch)
				errCheckout := gitutils.CheckoutBranch(originalBranch)
				if errCheckout != nil {
					fmt.Fprintf(r.stderr, ui.Colors.WarningStyle.Render("Warning: Failed to checkout original branch '%s': %v\n"), originalBranch, errCheckout)
				}
			}
		}
	}()

	// --- Fetch Base (with remote check) ---
	remoteName := "origin"
	shouldFetch := !r.noFetch
	if shouldFetch {
		_, errRemote := gitutils.GetRemoteURL(remoteName)
		if errRemote != nil {
			if strings.Contains(errRemote.Error(), "not found") {
				r.logger.Debug("Remote not found. Skipping fetch.", "remoteName", remoteName)
				shouldFetch = false
			} else {
				return fmt.Errorf("failed to check remote '%s': %w", remoteName, errRemote)
			}
		}
	}
	if shouldFetch {
		r.logger.Debug("Fetching latest", "baseBranch", baseBranch, "remoteName", remoteName)
		// Pass remote name to FetchBranch if it needs it
		if err := gitutils.FetchBranch(baseBranch, remoteName); err != nil {
			return fmt.Errorf("failed to fetch base branch '%s': %w.\nUse --no-fetch to skip", baseBranch, err)
		}
		fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render("Fetch complete."))
	} else if r.noFetch {
		r.logger.Debug("Skipping fetch (--no-fetch).")
	}

	// --- Iterative Rebase Loop ---
	r.logger.Debug("\n--- Starting Stack Rebase ---")
	rebasedBranches := []string{} // Keep track of branches we actually rebased/checked

	for i := 1; i < len(stack); i++ {
		branch := stack[i]
		parent := stack[i-1]

		r.logger.Debug("Processing branch", "index", i, "total", len(stack)-1, "branch", branch, "parent", parent)

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
			fmt.Fprintln(r.stdout, ui.Colors.WarningStyle.Render(fmt.Sprintf("  Warning: Could not find merge base between '%s' and '%s': %v. Attempting rebase anyway.", parent, branch, errMB)))
		} else if mergeBase == parentOID {
			r.logger.Debug("Branch is already based on current parent. Skipping rebase.", "branch", branch, "parent", parent)
			rebasedBranches = append(rebasedBranches, branch) // Add to list even if skipped, as it's confirmed correct
			continue                                          // Skip to next branch
		}

		// Checkout and Rebase
		r.logger.Debug("Checking out", "branch", branch)
		if err := gitutils.CheckoutBranch(branch); err != nil {
			return fmt.Errorf("failed to checkout branch '%s' for rebase: %w", branch, err)
		}

		r.logger.Debug("Rebasing onto parent", "branch", branch, "parent", parent, "parentOID", parentOID[:7])
		err = gitutils.RebaseCurrentBranchOnto(parentOID) // Rebase onto specific parent commit OID

		if err == nil {
			r.logger.Debug("Rebase step successful.")
			rebasedBranches = append(rebasedBranches, branch) // Track success
			continue                                          // Success, move to next branch
		}

		// Handle Rebase Failure
		if errors.Is(err, gitutils.ErrRebaseConflict) {
			// CONFLICT Case
			fmt.Fprintln(r.stderr, "")
			fmt.Fprintln(r.stderr, ui.Colors.WarningStyle.Render("⚠️ Rebase paused due to conflicts."))
			fmt.Fprintln(r.stderr, "   Resolve conflicts manually:")
			fmt.Fprintln(r.stderr, "     1. Edit conflicted files (see 'git status').")
			fmt.Fprintln(r.stderr, "     2. Run 'git add <resolved-files...>'.")
			fmt.Fprintln(r.stderr, "     3. Run 'git rebase --continue'.")
			fmt.Fprintln(r.stderr, "   (To cancel, run 'git rebase --abort')")
			fmt.Fprintln(r.stderr, "   Once the Git rebase is complete, run 'so restack' again.")

			rerereEnabled, errCheck := gitutils.IsRerereEnabled()
			if errCheck != nil {
				// Don't fail the process for this, just warn
				fmt.Fprintf(r.stderr, ui.Colors.WarningStyle.Render("   Warning: Could not check git rerere config: %v\n"), errCheck)
			} else if !rerereEnabled {
				fmt.Fprintln(r.stderr, "") // Extra space before tip
				fmt.Fprintln(r.stderr, ui.Colors.InfoStyle.Render(
					"   Tip: Enable 'git rerere' globally ('git config --global rerere.enabled true')"))
				fmt.Fprintln(r.stderr, ui.Colors.InfoStyle.Render(
					"        to potentially automate resolving similar conflicts in the future."))
			}

			cmd.SilenceUsage = true // Prevent usage printing
			return nil              // Exit cleanly, user needs to use Git
		}

		// Other Unexpected Rebase Failure
		return fmt.Errorf("unexpected error during rebase of '%s': %w", branch, err)

	}

	// --- Post-Success ---
	fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render("\n--- Stack Rebase Completed Successfully ---"))

	// Determine if push is desired
	doPush := false
	if r.forcePush {
		doPush = true
		r.logger.Debug("Force pushing specified via --force-push flag.")
	} else if r.noPush {
		doPush = false
		r.logger.Debug("Pushing disabled via --no-push flag.")
	} else {
		// Prompt user if flags don't decide
		confirmPush := false
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("Force push %d successfully rebased/checked branches to remote '%s'?", len(rebasedBranches), remoteName),
			Default: false, // Default to NO for safety
		}

		surveyOpts := survey.WithStdio(r.stdin.(*os.File), r.stderr.(*os.File), r.stderr.(*os.File))
		err := survey.AskOne(prompt, &confirmPush, surveyOpts)
		if err != nil {
			if err.Error() == "interrupt" {
				return ui.HandleSurveyInterrupt(err, "Push cancelled.")
			}
			fmt.Fprintf(r.stderr, "Push prompt failed: %v. Skipping push.\n", err)
		}
		doPush = confirmPush
	}

	// Execute push if needed
	if doPush && len(rebasedBranches) > 0 {
		r.logger.Debug("Force Pushing Updated Branches", "remoteName", remoteName, "count", len(rebasedBranches))
		pushSuccessCount := 0
		for _, branch := range rebasedBranches {
			fmt.Fprintf(r.stdout, "Pushing %s... ", branch)
			err := gitutils.PushBranchWithLease(branch, remoteName) // Use force-with-lease
			if err != nil {
				fmt.Fprintln(r.stdout, ui.Colors.FailureStyle.Render("Failed!"))
				// Log error but continue trying other branches? Or abort?
				fmt.Fprintf(r.stderr, "  Error pushing %s: %v\n", branch, err)
				// Let's continue for now
			} else {
				fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render("Success."))
				pushSuccessCount++
			}
		}
		r.logger.Debug("Finished pushing branches.", "successCount", pushSuccessCount)
	} else if len(rebasedBranches) == 0 {
		r.logger.Debug("No branches were identified as needing rebase or push.")
	} else {
		r.logger.Debug("Skipping push.")
	}

	return nil
}
