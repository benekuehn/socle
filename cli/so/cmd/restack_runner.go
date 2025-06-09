package cmd

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/benekuehn/socle/cli/so/internal/git"
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
	if git.IsRebaseInProgress() {
		_, _ = fmt.Fprintln(r.stderr, ui.Colors.InfoStyle.Render("Git rebase already in progress."))
		_, _ = fmt.Fprintln(r.stderr, ui.Colors.InfoStyle.Render("Resolve conflicts and run 'git rebase --continue' or cancel with 'git rebase --abort'."))
		_, _ = fmt.Fprintln(r.stderr, ui.Colors.InfoStyle.Render("Once the Git rebase is finished, run 'so restack' again if needed."))
		cmd.SilenceUsage = true // Prevent usage printing on clean exit
		return nil              // Exit cleanly, user needs to act in Git
	}
	hasChanges, err := git.HasUncommittedChanges()
	if err != nil {
		return fmt.Errorf("failed to check working tree status: %w", err)
	}
	if hasChanges {
		return fmt.Errorf("uncommitted changes detected. Please commit or stash them before restacking")
	}

	// Get complete stack info in one call
	stackInfo, err := git.GetStackInfo()
	if err != nil {
		return err
	}

	// Extract the information we need
	stack := stackInfo.FullStack
	baseBranch := stackInfo.BaseBranch
	currentBranch := stackInfo.CurrentBranch

	r.logger.Debug("Identified stack for restacking", "stack", stack, "base", baseBranch)

	if len(stack) <= 1 {
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render("No branches to restack: current branch is a base branch."))
		return nil
	}

	// Defer returning to the original branch
	defer func() {
		// Only run if no rebase is currently in progress (i.e., we didn't exit due to conflict)
		if !git.IsRebaseInProgress() {
			if currentBranch != baseBranch {
				r.logger.Debug("Checking out original branch", "name", currentBranch)
				errCheckout := git.CheckoutBranch(currentBranch)
				if errCheckout != nil {
					_, _ = fmt.Fprintf(r.stderr, ui.Colors.WarningStyle.Render("Warning: Failed to checkout original branch '%s': %v\\n"), currentBranch, errCheckout)
				}
			}
		}
	}()

	// --- Fetch Base (with remote check) ---
	remoteName := "origin"
	shouldFetch := !r.noFetch
	if shouldFetch {
		_, errRemote := git.GetRemoteURL(remoteName)
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
		if err := git.FetchBranch(baseBranch, remoteName); err != nil {
			return fmt.Errorf("failed to fetch base branch '%s': %w.\\nUse --no-fetch to skip", baseBranch, err)
		}
		r.logger.Debug("Fetch complete.")
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
		parentOID, errPO := git.GetCurrentBranchCommit(parent)
		if errPO != nil {
			return fmt.Errorf("cannot get current commit of parent '%s': %w", parent, errPO)
		}

		// Optimization Check
		mergeBase, errMB := git.GetMergeBase(parent, branch)
		if errMB != nil {
			// If merge-base fails, maybe the branches have diverged significantly?
			// Warn and proceed with rebase attempt.
			_, _ = fmt.Fprintln(r.stdout, ui.Colors.WarningStyle.Render(fmt.Sprintf("  Warning: Could not find merge base between '%s' and '%s': %v. Attempting rebase anyway.", parent, branch, errMB)))
		} else if mergeBase == parentOID {
			r.logger.Debug("Branch is already based on current parent. Skipping rebase.", "branch", branch, "parent", parent)
			rebasedBranches = append(rebasedBranches, branch) // Add to list even if skipped, as it's confirmed correct
			continue                                          // Skip to next branch
		}

		// Checkout and Rebase
		r.logger.Debug("Checking out", "branch", branch)
		if err := git.CheckoutBranch(branch); err != nil {
			return fmt.Errorf("failed to checkout branch '%s' for rebase: %w", branch, err)
		}

		r.logger.Debug("Rebasing onto parent", "branch", branch, "parent", parent, "parentOID", parentOID[:7])
		err = git.RebaseCurrentBranchOnto(parentOID) // Rebase onto specific parent commit OID

		if err == nil {
			r.logger.Debug("Rebase step successful.")
			rebasedBranches = append(rebasedBranches, branch) // Track success
			continue                                          // Success, move to next branch
		}

		// Handle Rebase Failure
		if errors.Is(err, git.ErrRebaseConflict) {
			// CONFLICT Case
			_, _ = fmt.Fprintln(r.stderr, "")
			_, _ = fmt.Fprintln(r.stderr, ui.Colors.WarningStyle.Render("⚠️ Rebase paused due to conflicts."))
			_, _ = fmt.Fprintf(r.stderr, "Please resolve the conflicts in branch '%s' and then run:\n", branch)
			_, _ = fmt.Fprintln(r.stderr, "  1. Run 'git add <resolved-files...>'.")
			_, _ = fmt.Fprintln(r.stderr, "  2. Run 'git rebase --continue'.")
			_, _ = fmt.Fprintln(r.stderr, "   (To cancel, run 'git rebase --abort')")
			_, _ = fmt.Fprintln(r.stderr, "   Once the Git rebase is complete, run 'so restack' again.")

			cmd.SilenceUsage = true // Prevent usage printing
			return nil              // Exit cleanly, user needs to use Git
		}

		// Other Unexpected Rebase Failure
		return fmt.Errorf("unexpected error during rebase of '%s': %w", branch, err)
	}

	// --- Post-Success ---
	_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render("\n✓ Stack Rebase Completed Successfully\n"))

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
			_, _ = fmt.Fprintf(r.stderr, "Push prompt failed: %v. Skipping push.\n", err)
		}
		doPush = confirmPush
	}

	// Execute push if needed
	if doPush && len(rebasedBranches) > 0 {
		r.logger.Debug("Force Pushing Updated Branches", "remoteName", remoteName, "count", len(rebasedBranches))
		pushSuccessCount := 0
		for _, branch := range rebasedBranches {
			_, _ = fmt.Fprintf(r.stdout, "Pushing %s... ", branch)
			err := git.PushBranchWithLease(branch, remoteName) // Use force-with-lease
			if err != nil {
				_, _ = fmt.Fprintln(r.stdout, ui.Colors.FailureStyle.Render("Failed!"))
				// Log error but continue trying other branches? Or abort?
				_, _ = fmt.Fprintf(r.stderr, "  Error pushing %s: %v\n", branch, err)
				// Let's continue for now
			} else {
				_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render("Success."))
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
