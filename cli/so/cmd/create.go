package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/benekuehn/socle/cli/so/gitutils"
	"github.com/benekuehn/socle/cli/so/internal/ui"
	"github.com/spf13/cobra"
)

// Flag variables
var createMessage string

// --- ADD TESTING FLAGS ---
var (
	testBranchName      string // Bypass branch name prompt
	testStageChoice     string // Bypass stage choice prompt ("add-all", "add-p", "cancel")
	testAddPResultEmpty bool   // Simulate 'git add -p' staging nothing
)

var createCmd = &cobra.Command{
	Use:   "create [branch-name]",
	Short: "Create the next branch in the stack, optionally committing current changes",
	Long: `Creates a new branch stacked on top of the current branch.

If a [branch-name] is not provided, you will be prompted for one.

If there are uncommitted changes in the working directory:
  - They will be staged and committed onto the *new* branch.
  - You must provide a commit message via the -m flag, or you will be prompted.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Get current branch info
		parentBranch, err := gitutils.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
		parentCommit, err := gitutils.GetCurrentCommit()
		if err != nil {
			return fmt.Errorf("failed to get current commit hash: %w", err)
		}

		// 2. Check if parent branch is tracked
		parentParentKey := fmt.Sprintf("branch.%s.socle-parent", parentBranch)
		parentBaseKey := fmt.Sprintf("branch.%s.socle-base", parentBranch)
		_, errParent := gitutils.GetGitConfig(parentParentKey)
		parentBase, errBase := gitutils.GetGitConfig(parentBaseKey)

		// Check if parent is tracked (needs both keys, essentially)
		// Allow creating off a known base branch directly
		knownBases := map[string]bool{"main": true, "master": true, "develop": true} // TODO: Configurable
		isParentBase := knownBases[parentBranch]
		isParentTracked := (errParent == nil && errBase == nil) || isParentBase

		if !isParentTracked {
			// Provide specific guidance
			if isParentBase {
				// If creating off a base, implicitly determine base
				parentBase = parentBranch
			} else {
				return fmt.Errorf("current branch '%s' is not tracked by socle and is not a known base branch.\nRun 'so track' on this branch first before creating a child branch", parentBranch)
			}
		} else if isParentBase {
			// Set parentBase explicitly if we are on a base branch
			parentBase = parentBranch
		}
		// If parent was tracked but not a base, parentBase already holds the inherited base

		if parentBase == "" {
			return fmt.Errorf("internal error: could not determine base branch for parent '%s'", parentBranch)
		}

		// 3. Determine new branch name
		newBranchName := ""
		if testBranchName != "" { // Check test flag first
			slog.Debug("Using branch name from test flag", "testBranchName", testBranchName)
			newBranchName = testBranchName
		} else if len(args) > 0 {
			newBranchName = args[0]
		} else {
			prompt := &survey.Input{Message: "Enter name for the new branch:"}
			err := survey.AskOne(prompt, &newBranchName, survey.WithValidator(survey.Required), survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
			if err != nil {
				return handleSurveyInterrupt(err, "Create cancelled.")
			}
		}

		// 4. Validate new branch name
		if err := gitutils.IsValidBranchName(newBranchName); err != nil {
			return fmt.Errorf("invalid branch name '%s': %w", newBranchName, err)
		}
		exists, err := gitutils.BranchExists(newBranchName)
		if err != nil {
			return fmt.Errorf("failed to check if branch '%s' exists: %w", newBranchName, err)
		}
		if exists {
			return fmt.Errorf("branch '%s' already exists", newBranchName)
		}

		// 5. Check for uncommitted changes
		hasChanges, err := gitutils.HasUncommittedChanges()
		if err != nil {
			return fmt.Errorf("failed to check working tree status: %w", err)
		}

		// 6. Determine commit message if needed
		commitMsg := ""
		if hasChanges {
			if createMessage != "" {
				commitMsg = createMessage
			} else {
				// Prompt for commit message
				prompt := &survey.Input{Message: "Enter commit message for current changes:"}
				err := survey.AskOne(prompt, &commitMsg, survey.WithValidator(survey.Required), survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
				if err != nil {
					return handleSurveyInterrupt(err, "Create cancelled.")
				}
			}
		}

		// --- Action Sequence ---

		slog.Debug("Creating branch '%s' from '%s'...\n", newBranchName, parentBranch)
		// 1. Create branch
		if err := gitutils.CreateBranch(newBranchName, parentCommit); err != nil {
			return fmt.Errorf("failed to create branch '%s': %w", newBranchName, err)
		}

		// Defer cleanup in case subsequent steps fail
		cleanupNeeded := true
		defer func() {
			if cleanupNeeded {
				fmt.Fprintf(os.Stderr, "Cleaning up branch '%s' due to error...\n", newBranchName)
				// Best effort cleanup: try to switch back and delete
				_ = gitutils.CheckoutBranch(parentBranch) // Switch back first
				_ = gitutils.BranchDelete(newBranchName)  // Delete the failed branch
			}
		}()

		// 2. Checkout new branch
		slog.Debug("Checking out", "newBranchName", newBranchName)
		if err := gitutils.CheckoutBranch(newBranchName); err != nil {
			return fmt.Errorf("failed to checkout new branch '%s': %w", newBranchName, err)
		}

		// 3. Stage and Commit (if needed)
		commitOccurred := false // Track if we actually commit
		if commitMsg != "" {    // commitMsg will only be non-empty if hasChanges was true
			stageChoice := ""
			if testStageChoice != "" {
				slog.Debug("Using stage choice from test flag", "testStageChoice", testStageChoice)
				stageChoice = testStageChoice
			} else {
				prompt := &survey.Select{
					Message: "You have uncommitted changes. How would you like to stage them?",
					Options: []string{
						"Stage all changes (`git add .`)",
						"Stage interactively (`git add -p`)",
						"Cancel",
					},
					Default: "Stage all changes (`git add .`)",
				}
				err := survey.AskOne(prompt, &stageChoice, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
				if err != nil {
					return handleSurveyInterrupt(err, "Create cancelled.")
				}
			}

			stagedSomething := false
			switch stageChoice {
			case "Stage all changes (`git add .`)", "add-all":
				slog.Debug("Staging all changes...")
				if err := gitutils.StageAllChanges(); err != nil {
					return fmt.Errorf("failed to stage all changes: %w", err)
				}
				stagedSomething = true // Assume git add . stages something if there were changes initially
			case "Stage interactively (`git add -p`)", "add-p":
				if testAddPResultEmpty {
					// Simulate user running add -p but staging nothing
					slog.Debug("Simulating 'git add -p' with no changes staged (via test flag)")
					stagedSomething = false
				} else {
					slog.Info("Starting interactive staging (`git add -p`)...")
					slog.Debug("Calling gitutils.StageInteractively")
					if err := gitutils.StageInteractively(); err != nil {
						return fmt.Errorf("interactive staging failed: %w", err)
					}
					slog.Debug("Interactive staging finished, checking if changes were staged")
					haveStaged, errCheck := gitutils.HasStagedChanges()
					if errCheck != nil {
						return fmt.Errorf("failed to check staged changes after interactive add: %w", errCheck)
					}
					if !haveStaged {
						slog.Warn("No changes were staged during interactive add.")
					}
					stagedSomething = haveStaged
				}
			case "Cancel", "cancel":
				slog.Debug("Operation cancelled during staging.")
				return nil // Let defer cleanup
			default:
				return fmt.Errorf("internal error: unexpected staging choice")
			}

			// Only commit if something was potentially staged
			if stagedSomething {
				// Verify again if anything is staged *before* committing
				haveStaged, errCheck := gitutils.HasStagedChanges()
				if errCheck != nil {
					return fmt.Errorf("failed to verify staged changes before commit: %w", errCheck)
				}

				if haveStaged {
					slog.Debug("Committing staged changes", "message", commitMsg)
					if err := gitutils.CommitChanges(commitMsg); err != nil {
						fmt.Fprint(os.Stderr, ui.Colors.FailureStyle.Render("Commit failed (hooks?). Aborting.\n"))
						return fmt.Errorf("failed to commit changes: %w", err)
					}
					fmt.Println(ui.Colors.SuccessStyle.Render("Changes committed successfully."))
					commitOccurred = true
				} else {
					// This can happen if `git add .` staged nothing (e.g. only .gitignore changes)
					// or if the user exited `git add -p` without staging anything.
					fmt.Println(ui.Colors.InfoStyle.Render("No changes were staged, skipping commit."))
				}
			}
		} // End of commitMsg != "" block

		// 4. Update metadata
		slog.Debug("Updating socle tracking information...")
		newParentKey := fmt.Sprintf("branch.%s.socle-parent", newBranchName)
		newBaseKey := fmt.Sprintf("branch.%s.socle-base", newBranchName)

		if err := gitutils.SetGitConfig(newParentKey, parentBranch); err != nil {
			return fmt.Errorf("failed to set socle-parent config for '%s': %w", newBranchName, err)
		}
		if err := gitutils.SetGitConfig(newBaseKey, parentBase); err != nil {
			_ = gitutils.UnsetGitConfig(newParentKey) // Attempt cleanup
			return fmt.Errorf("failed to set socle-base config for '%s': %w", newBranchName, err)
		}

		// Success! Prevent cleanup
		cleanupNeeded = false
		finalMessage := fmt.Sprintf("Successfully created and tracked branch '%s' on top of '%s'.", newBranchName, parentBranch)
		if commitOccurred {
			finalMessage += " Changes committed."
		} else if hasChanges && !commitOccurred {
			finalMessage += " No changes were committed."
		}

		fmt.Println(ui.Colors.SuccessStyle.Render(finalMessage))
		return nil
	},
}

func init() {
	AddCommand(createCmd)
	createCmd.Flags().StringVarP(&createMessage, "message", "m", "", "Commit message to use for uncommitted changes")
	createCmd.Flags().StringVar(&testBranchName, "test-branch-name", "", "Branch name to use (testing only)")
	createCmd.Flags().StringVar(&testStageChoice, "test-stage-choice", "", "Staging choice [add-all|add-p|cancel] (testing only)")
	createCmd.Flags().BoolVar(&testAddPResultEmpty, "test-addp-empty", false, "Simulate 'git add -p' staging nothing (testing only)")
	_ = createCmd.Flags().MarkHidden("test-branch-name")
	_ = createCmd.Flags().MarkHidden("test-stage-choice")
	_ = createCmd.Flags().MarkHidden("test-addp-empty")
}

// Helper function to handle survey interrupt (Ctrl+C) gracefully
func handleSurveyInterrupt(err error, message string) error {
	if err.Error() == "interrupt" {
		fmt.Println(message)
		os.Exit(0) // Clean exit
	}
	return fmt.Errorf("prompt failed: %w", err)
}
