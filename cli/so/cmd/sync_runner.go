package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"sync"

	"github.com/AlecAivazis/survey/v2"
	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
	"github.com/spf13/cobra"
)

type syncCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader // For prompts

	// Config flags
	doRestack bool
	noFetch   bool
}

func (r *syncCmdRunner) run(cmd *cobra.Command) error {
	// --- Pre-Checks ---
	if git.IsRebaseInProgress() {
		_, _ = fmt.Fprintln(r.stderr, ui.Colors.InfoStyle.Render("Git rebase already in progress."))
		_, _ = fmt.Fprintln(r.stderr, ui.Colors.InfoStyle.Render("Resolve conflicts and run 'git rebase --continue' or cancel with 'git rebase --abort'."))
		_, _ = fmt.Fprintln(r.stderr, ui.Colors.InfoStyle.Render("Once the Git rebase is finished, run 'so sync' again if needed."))
		cmd.SilenceUsage = true // Prevent usage printing on clean exit
		return nil              // Exit cleanly, user needs to act in Git
	}

	hasChanges, err := git.HasUncommittedChanges()
	if err != nil {
		return fmt.Errorf("failed to check working tree status: %w", err)
	}
	if hasChanges {
		return fmt.Errorf("uncommitted changes detected. Please commit or stash them before syncing")
	}

	// --- Setup GitHub Client ---
	remoteName := "origin"
	remoteURL, err := git.GetRemoteURL(remoteName)
	if err != nil {
		return fmt.Errorf("cannot get remote URL for '%s': %w", remoteName, err)
	}

	owner, repoName, err := git.ParseOwnerAndRepo(remoteURL)
	if err != nil {
		return fmt.Errorf("cannot parse owner/repo from remote '%s' URL '%s': %w", remoteName, remoteURL, err)
	}

	ghClient, err := gh.NewClient(context.Background(), owner, repoName)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// --- Fetch All Branches ---
	if !r.noFetch {
		_, _ = fmt.Fprintln(r.stdout, "Fetching all branches from remote...")
		if err := git.FetchAll(remoteName); err != nil {
			return fmt.Errorf("failed to fetch from remote '%s': %w", remoteName, err)
		}
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render("Fetch complete."))
	} else {
		_, _ = fmt.Fprintln(r.stdout, "Skipping fetch (--no-fetch).")
	}

	// --- Get Stack Info ---
	stackInfo, err := git.GetStackInfo()
	if err != nil {
		return fmt.Errorf("failed to get stack info: %w", err)
	}

	// --- Check PR Statuses and Clean Up ---
	_, _ = fmt.Fprintln(r.stdout, "\nChecking PR statuses...")

	// Process branches in parallel
	var wg sync.WaitGroup
	results := make(map[string]struct {
		prNumber int
		status   string
	})
	var mu sync.Mutex

	for i := 1; i < len(stackInfo.FullStack); i++ {
		branch := stackInfo.FullStack[i]
		prNumber, err := git.GetStoredPRNumber(branch)
		if err != nil || prNumber == 0 {
			continue // Skip branches without PRs
		}

		wg.Add(1)
		go func(branchName string, prNum int) {
			defer wg.Done()

			status, _, err := ghClient.GetPullRequestStatus(prNum)
			if err != nil {
				_, _ = fmt.Fprintf(r.stderr, ui.Colors.WarningStyle.Render("  Warning: Could not get status for PR #%d (branch '%s'): %v\n"), prNum, branchName, err)
				return
			}

			if status == gh.PRStatusMerged || status == gh.PRStatusClosed {
				mu.Lock()
				results[branchName] = struct {
					prNumber int
					status   string
				}{prNum, status}
				mu.Unlock()
			}
		}(branch, prNumber)
	}

	// Wait for all checks to complete
	wg.Wait()

	// Process results in order
	branchesToDelete := make([]string, 0, len(results))
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	for i := 1; i < len(stackInfo.FullStack); i++ {
		branch := stackInfo.FullStack[i]
		if result, ok := results[branch]; ok {
			// Include the current branch in branches to delete
			branchesToDelete = append(branchesToDelete, branch)
			_, _ = fmt.Fprintf(r.stdout, "  Found %s PR #%d for branch '%s'\n", result.status, result.prNumber, branch)
		}
	}

	// --- Prompt to Delete Branches ---
	if len(branchesToDelete) > 0 {
		_, _ = fmt.Fprintf(r.stdout, "\nThe following branches have merged or closed PRs:\n")
		for _, branch := range branchesToDelete {
			_, _ = fmt.Fprintf(r.stdout, "  - %s\n", branch)
		}

		confirm := false
		prompt := &survey.Confirm{
			Message: "Delete these " + strconv.Itoa(len(branchesToDelete)) + " branches?",
		}
		if err := survey.AskOne(prompt, &confirm); err != nil {
			return fmt.Errorf("failed to get user confirmation: %w", err)
		}

		if confirm {
			// Get initial stack info before any modifications
			initialStackInfo, err := git.GetStackInfo()
			if err != nil {
				return fmt.Errorf("failed to get initial stack info: %w", err)
			}

			// Create a map of branch -> new parent for all branches that need updating
			// This ensures we have all the updates ready before making any changes
			branchUpdates := make(map[string]string)
			for _, branch := range branchesToDelete {
				// Get the parent of the branch to be deleted
				parentConfigKey := fmt.Sprintf("branch.%s.socle-parent", branch)
				deletedBranchParent, err := git.GetGitConfig(parentConfigKey)
				if err != nil {
					return fmt.Errorf("failed to get parent for branch '%s': %w", branch, err)
				}

				// Find all branches that were tracking this branch
				for _, currentBranch := range initialStackInfo.FullStack {
					if currentBranch == branch || currentBranch == initialStackInfo.BaseBranch {
						continue
					}
					parent, ok := initialStackInfo.ParentMap[currentBranch]
					if !ok {
						continue
					}
					if parent == branch {
						// This branch needs to be updated to track the deleted branch's parent
						branchUpdates[currentBranch] = deletedBranchParent
					}
				}
			}

			// Apply all tracking updates first
			for branch, newParent := range branchUpdates {
				if err := git.UpdateBranchParent(branch, newParent); err != nil {
					return fmt.Errorf("failed to update parent for branch '%s' to '%s': %w", branch, newParent, err)
				}
				_, _ = fmt.Fprintf(r.stdout, "  Updated tracking for branch '%s' to track '%s'\n", branch, newParent)
			}

			// Now that all tracking is updated, delete the branches
			for _, branch := range branchesToDelete {
				// If this is the current branch, switch to main first
				if branch == currentBranch {
					if err := git.SwitchBranch(initialStackInfo.BaseBranch); err != nil {
						return fmt.Errorf("failed to switch to base branch before deleting current branch: %w", err)
					}
					_, _ = fmt.Fprintf(r.stdout, "  Switched to base branch '%s'\n", initialStackInfo.BaseBranch)
				}

				_, _ = fmt.Fprintf(r.stdout, "Deleting branch %s... ", branch)
				if err := git.DeleteBranch(branch); err != nil {
					_, _ = fmt.Fprintln(r.stdout, ui.Colors.FailureStyle.Render("Failed"))
					return fmt.Errorf("failed to delete branch '%s': %w", branch, err)
				}
				_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render("Success"))
			}
		}
	}

	// --- Update Trunk ---
	baseBranch := stackInfo.BaseBranch
	_, _ = fmt.Fprintf(r.stdout, "\nUpdating trunk branch '%s'...\n", baseBranch)

	// Try fast-forward first
	if err := git.FastForwardBranch(baseBranch, remoteName); err != nil {
		if errors.Is(err, git.ErrNotFastForward) {
			// Not fast-forwardable, need to force update
			_, _ = fmt.Fprintln(r.stdout, ui.Colors.WarningStyle.Render("  Trunk cannot be fast-forwarded. Force updating..."))
			if err := git.ForceUpdateBranch(baseBranch, remoteName); err != nil {
				return fmt.Errorf("failed to force update trunk: %w", err)
			}
			_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render("  Trunk force updated."))
		} else {
			return fmt.Errorf("failed to update trunk: %w", err)
		}
	} else {
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render("  Trunk fast-forwarded."))
	}

	// --- Restack if Enabled ---
	if r.doRestack {
		_, _ = fmt.Fprintln(r.stdout, "\nRestacking branches...")
		restackRunner := &restackCmdRunner{
			logger:  r.logger,
			stdout:  r.stdout,
			stderr:  r.stderr,
			stdin:   r.stdin,
			noFetch: true, // We already fetched
			noPush:  true, // Don't push during sync
		}
		if err := restackRunner.run(cmd); err != nil {
			return fmt.Errorf("failed during restack: %w", err)
		}
	}

	_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render("\nSync completed successfully."))
	return nil
}
