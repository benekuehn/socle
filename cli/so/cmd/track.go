package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/benekuehn/socle/cli/so/gitutils"
	"github.com/benekuehn/socle/cli/so/internal/ui"
	"github.com/spf13/cobra"
)

var (
	testSelectedParent string // Flag to bypass parent prompt in tests
	testAssumeBase     string // Flag to bypass base determination/warning in tests
)

var trackCmd = &cobra.Command{
	Use:   "track",
	Short: "Start tracking the current branch as part of a stack",
	Long: `Associates the current branch with a parent branch to define its position
within a stack. This allows 'socle show' to display the specific stack you are on.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Get current branch
		currentBranch, err := gitutils.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
		// Basic check: Don't track base branches like main/master/develop
		// TODO: Make base branches configurable
		if currentBranch == "main" || currentBranch == "master" || currentBranch == "develop" {
			return fmt.Errorf("cannot track a base branch ('%s') itself", currentBranch)
		}

		// 2. Check if already tracked
		parentConfigKey := fmt.Sprintf("branch.%s.socle-parent", currentBranch)
		existingParent, errGetParent := gitutils.GetGitConfig(parentConfigKey)
		// We expect an error if the key doesn't exist, that's the non-tracked case.
		// If err is nil or a different error occurs, handle it.
		if errGetParent == nil && existingParent != "" {
			baseConfigKey := fmt.Sprintf("branch.%s.socle-base", currentBranch)
			existingBase, _ := gitutils.GetGitConfig(baseConfigKey) // Ignore error here
			fmt.Printf("Branch '%s' is already tracked.\n", currentBranch)
			fmt.Printf("  Parent: %s\n", existingParent)
			fmt.Printf("  Base:   %s\n", existingBase)
			return nil // Not an error state, just info.
		} else if errGetParent != nil && !errors.Is(errGetParent, gitutils.ErrConfigNotFound) {
			// Handle unexpected errors from git config
			return fmt.Errorf("failed to check tracking status for branch '%s': %w", currentBranch, err)
		}

		// 3. Get potential parent branches
		allBranches, err := gitutils.GetLocalBranches()
		if err != nil {
			return fmt.Errorf("failed to list local branches: %w", err)
		}

		potentialParents := []string{}
		knownBases := map[string]bool{"main": true, "master": true, "develop": true} // TODO: Configurable
		for _, b := range allBranches {
			if b != currentBranch { // Can't be its own parent
				potentialParents = append(potentialParents, b)
			}
		}

		if len(potentialParents) == 0 {
			return fmt.Errorf("no other local branches found to select as a parent")
		}

		// 4. Prompt user to select parent
		selectedParent := ""
		if testSelectedParent != "" {
			slog.Debug("Using parent branch from test flag", "testParent", testSelectedParent)
			// Validate the test flag value exists as a branch
			found := false
			for _, p := range potentialParents {
				if p == testSelectedParent {
					found = true
					break
				}
			}
			if !found && !knownBases[testSelectedParent] { // Allow test parent to be a known base too
				return fmt.Errorf("invalid test parent '%s': not found in potential parents %v or known bases", testSelectedParent, potentialParents)
			}
			selectedParent = testSelectedParent
		} else {
			// User prompt (Keep survey for actual use)
			slog.Debug("Prompting user for parent branch")
			prompt := &survey.Select{Message: fmt.Sprintf("Select the parent branch for '%s':", currentBranch), Options: potentialParents}
			err := survey.AskOne(prompt, &selectedParent, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
			if err != nil {
				return ui.HandleSurveyInterrupt(err, "Track command cancelled.")
			}
			slog.Debug("Parent selected via prompt", "selectedParent", selectedParent)
		}

		// 5. Determine and store base branch
		selectedBase := ""
		if knownBases[selectedParent] {
			// Parent is a known base branch
			selectedBase = selectedParent
		} else {
			// Parent is another feature branch, check if IT is tracked
			parentBaseKey := fmt.Sprintf("branch.%s.socle-base", selectedParent)
			inheritedBase, errGetBase := gitutils.GetGitConfig(parentBaseKey)
			if errGetBase == nil && inheritedBase != "" {
				// Parent is tracked, inherit its base
				selectedBase = inheritedBase
				slog.Debug("Inheriting base from tracked parent", "base", selectedBase, "parent", selectedParent)
			} else if errors.Is(errGetBase, gitutils.ErrConfigNotFound) {
				// Parent is untracked. Use test flag or default/warn.
				slog.Debug("Selected parent is not tracked", "parent", selectedParent)
				if testAssumeBase != "" {
					slog.Debug("Using base from test flag", "testBase", testAssumeBase)
					selectedBase = testAssumeBase
				} else {
					// User feedback - keep fmt.Println for warning
					fmt.Println(ui.Colors.WarningStyle.Render(fmt.Sprintf(
						"Warning: Parent branch '%s' is not tracked. Assuming stack base is '%s'.", selectedParent, defaultBaseBranch))) // Use a constant/variable for default base
					fmt.Println(ui.Colors.WarningStyle.Render("Consider tracking the parent branch first for more accurate stack definitions."))
					selectedBase = defaultBaseBranch // Default assumption
				}
			} else {
				// Unexpected error reading parent's base config
				return fmt.Errorf("failed to check tracking base for parent branch '%s': %w", selectedParent, errGetBase)
			}
		}

		if selectedBase == "" {
			// Should not happen if logic above is correct, but as a safeguard
			return fmt.Errorf("could not determine base branch for the stack")
		}

		// 6. Store metadata in git config
		fmt.Printf("Tracking branch '%s' with parent '%s' and base '%s'.\n", currentBranch, selectedParent, selectedBase)

		err = gitutils.SetGitConfig(parentConfigKey, selectedParent)
		if err != nil {
			return fmt.Errorf("failed to set socle-parent config: %w", err)
		}

		baseConfigKey := fmt.Sprintf("branch.%s.socle-base", currentBranch)
		err = gitutils.SetGitConfig(baseConfigKey, selectedBase)
		if err != nil {
			// Attempt to clean up the parent key if setting base fails
			_ = gitutils.UnsetGitConfig(parentConfigKey) // Ignore error during cleanup
			return fmt.Errorf("failed to set socle-base config: %w", err)
		}

		fmt.Println(ui.Colors.SuccessStyle.Render("Branch tracking information saved successfully."))
		return nil
	},
}

const defaultBaseBranch = "main"

func init() {
	AddCommand(trackCmd)
	trackCmd.Flags().StringVar(&testSelectedParent, "test-parent", "", "Parent branch to select (for testing only)")
	trackCmd.Flags().StringVar(&testAssumeBase, "test-base", "", "Base branch to assume if parent is untracked (for testing only)")
	_ = trackCmd.Flags().MarkHidden("test-parent")
	_ = trackCmd.Flags().MarkHidden("test-base")
}
