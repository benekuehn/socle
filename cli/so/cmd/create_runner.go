package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
)

type createCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader // Needed for survey prompts

	// Config flags
	createMessage string
	branchNameArg string // Optional branch name from args[0]

	// --- TESTING FLAGS ---
	testBranchName      string
	testStageChoice     string
	testAddPResultEmpty bool
}

func (r *createCmdRunner) run() error {
	// 1. Get current branch info
	parentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	parentCommit, err := git.GetCurrentCommit()
	if err != nil {
		return fmt.Errorf("failed to get current commit hash: %w", err)
	}

	// 2. Check if parent branch is tracked
	parentParentKey := fmt.Sprintf("branch.%s.socle-parent", parentBranch)
	parentBaseKey := fmt.Sprintf("branch.%s.socle-base", parentBranch)
	_, errParent := git.GetGitConfig(parentParentKey)
	parentBase, errBase := git.GetGitConfig(parentBaseKey)

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

	// 2.5. Validate linear stack constraint: non-base branches can only have one child
	if !isParentBase {
		// Check if parent already has children
		parentMap, err := git.GetAllSocleParents()
		if err != nil {
			return fmt.Errorf("failed to check existing branch relationships: %w", err)
		}
		childMap := git.BuildChildMap(parentMap)

		if existingChildren, hasChildren := childMap[parentBranch]; hasChildren && len(existingChildren) > 0 {
			return fmt.Errorf("non-base branch '%s' already has child branch(es): %v. Only base branches can have multiple children. Use 'so up' to navigate to the existing child or create a new stack from the base branch", parentBranch, existingChildren)
		}
	}

	// 3. Determine new branch name
	newBranchName := ""
	if r.testBranchName != "" {
		r.logger.Debug("Using branch name from test flag", "testBranchName", r.testBranchName)
		newBranchName = r.testBranchName
	} else if r.branchNameArg != "" {
		newBranchName = r.branchNameArg
	} else {
		prompt := &survey.Input{Message: "Enter name for the new branch:"}
		surveyOpts := survey.WithStdio(r.stdin.(*os.File), r.stderr.(*os.File), r.stderr.(*os.File))
		err := survey.AskOne(prompt, &newBranchName, survey.WithValidator(survey.Required), surveyOpts)
		if err != nil {
			return ui.HandleSurveyInterrupt(err, "Create cancelled.")
		}
	}

	// 4. Validate new branch name
	if err := git.IsValidBranchName(newBranchName); err != nil {
		return fmt.Errorf("invalid branch name '%s': %w", newBranchName, err)
	}
	exists, err := git.BranchExists(newBranchName)
	if err != nil {
		return fmt.Errorf("failed to check if branch '%s' exists: %w", newBranchName, err)
	}
	if exists {
		return fmt.Errorf("branch '%s' already exists", newBranchName)
	}

	// 5. Check for uncommitted changes
	hasChanges, err := git.HasUncommittedChanges()
	if err != nil {
		return fmt.Errorf("failed to check working tree status: %w", err)
	}

	// 6. Determine commit message if needed
	commitMsg := ""
	if hasChanges {
		if r.createMessage != "" {
			commitMsg = r.createMessage
		} else {
			prompt := &survey.Input{Message: "Enter commit message for current changes:"}
			surveyOpts := survey.WithStdio(r.stdin.(*os.File), r.stderr.(*os.File), r.stderr.(*os.File))
			err := survey.AskOne(prompt, &commitMsg, survey.WithValidator(survey.Required), surveyOpts)
			if err != nil {
				return ui.HandleSurveyInterrupt(err, "Create cancelled.")
			}
		}
	}

	// --- Action Sequence ---

	r.logger.Debug("Creating branch...", "newBranchName", newBranchName, "parentBranch", parentBranch)
	// 1. Create branch
	if err := git.CreateBranch(newBranchName, parentCommit); err != nil {
		return fmt.Errorf("failed to create branch '%s': %w", newBranchName, err)
	}

	// Defer cleanup in case subsequent steps fail
	cleanupNeeded := true
	defer func() {
		if cleanupNeeded {
			_, _ = fmt.Fprintf(r.stderr, "Cleaning up branch '%s' due to error...\\n", newBranchName)
			// Best effort cleanup: try to switch back and delete created branch
			_ = git.CheckoutBranch(parentBranch)
			_ = git.BranchDelete(newBranchName)
		}
	}()

	// 2. Checkout new branch
	r.logger.Debug("Checking out", "newBranchName", newBranchName)
	if err := git.CheckoutBranch(newBranchName); err != nil {
		return fmt.Errorf("failed to checkout new branch '%s': %w", newBranchName, err)
	}

	// 3. Stage and Commit (if needed)
	commitOccurred := false // Track if we actually commit
	if commitMsg != "" {    // commitMsg will only be non-empty if hasChanges was true
		stageChoice := ""
		if r.testStageChoice != "" {
			r.logger.Debug("Using stage choice from test flag", "testStageChoice", r.testStageChoice)
			stageChoice = r.testStageChoice
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

			surveyOpts := survey.WithStdio(r.stdin.(*os.File), r.stderr.(*os.File), r.stderr.(*os.File))
			err := survey.AskOne(prompt, &stageChoice, surveyOpts)
			if err != nil {
				return ui.HandleSurveyInterrupt(err, "Create cancelled.")
			}
		}

		stagedSomething := false
		switch stageChoice {
		case "Stage all changes (`git add .`)", "add-all":
			r.logger.Debug("Staging all changes...")
			if err := git.StageAllChanges(); err != nil {
				return fmt.Errorf("failed to stage all changes: %w", err)
			}
			stagedSomething = true
		case "Stage interactively (`git add -p`)", "add-p":
			if r.testAddPResultEmpty {
				// Simulate user running add -p but staging nothing
				r.logger.Debug("Simulating 'git add -p' with no changes staged (via test flag)")
				stagedSomething = false
			} else {
				r.logger.Info("Starting interactive staging (`git add -p`)...")
				r.logger.Debug("Calling git.StageInteractively")
				if err := git.StageInteractively(); err != nil {
					return fmt.Errorf("interactive staging failed: %w", err)
				}
				r.logger.Debug("Interactive staging finished, checking if changes were staged")
				haveStaged, errCheck := git.HasStagedChanges()
				if errCheck != nil {
					return fmt.Errorf("failed to check staged changes after interactive add: %w", errCheck)
				}
				if !haveStaged {
					r.logger.Warn("No changes were staged during interactive add.")
				}
				stagedSomething = haveStaged
			}
		case "Cancel", "cancel":
			r.logger.Debug("Operation cancelled during staging.")
			return nil // Let defer cleanup
		default:
			return fmt.Errorf("internal error: unexpected staging choice")
		}

		// Only commit if something was potentially staged
		if stagedSomething {
			// Verify again if anything is staged *before* committing
			haveStaged, errCheck := git.HasStagedChanges()
			if errCheck != nil {
				return fmt.Errorf("failed to verify staged changes before commit: %w", errCheck)
			}

			if haveStaged {
				r.logger.Debug("Committing staged changes", "message", commitMsg)
				if err := git.CommitChanges(commitMsg); err != nil {
					_, _ = fmt.Fprint(r.stderr, ui.Colors.FailureStyle.Render("Commit failed (hooks?). Aborting.\\n"))
					return fmt.Errorf("failed to commit changes: %w", err)
				}
				r.logger.Debug("Changes committed successfully..")
				commitOccurred = true
			} else {
				// This can happen if `git add .` staged nothing (e.g. only .gitignore changes)
				// or if the user exited `git add -p` without staging anything.
				_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render("No changes were staged, skipping commit."))
			}
		}
	}

	// 4. Update metadata
	r.logger.Debug("Updating socle tracking information...")
	newParentKey := fmt.Sprintf("branch.%s.socle-parent", newBranchName)
	newBaseKey := fmt.Sprintf("branch.%s.socle-base", newBranchName)

	if err := git.SetGitConfig(newParentKey, parentBranch); err != nil {
		return fmt.Errorf("failed to set socle-parent config for '%s': %w", newBranchName, err)
	}
	if err := git.SetGitConfig(newBaseKey, parentBase); err != nil {
		_ = git.UnsetGitConfig(newParentKey) // Attempt cleanup
		return fmt.Errorf("failed to set socle-base config for '%s': %w", newBranchName, err)
	}

	// Success! Prevent cleanup
	cleanupNeeded = false
	finalMessage := fmt.Sprintf("✓ Created and tracked branch '%s' on top of '%s'.", newBranchName, parentBranch)
	if commitOccurred {
		finalMessage += " Changes committed."
	} else if hasChanges && !commitOccurred {
		// User started with changes, went through staging, but nothing ended up staged/committed
		finalMessage += " Uncommitted changes remain."
	}
	_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render(finalMessage))

	return nil
}
