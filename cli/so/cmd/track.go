package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/benekuehn/socle/cli/so/gitutils"
	"github.com/benekuehn/socle/cli/so/internal/ui"
	"github.com/spf13/cobra"
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
		prompt := &survey.Select{
			Message: fmt.Sprintf("Select the parent branch for '%s':", currentBranch),
			Options: potentialParents,
			// Suggest common bases first? Could sort potentialParents here.
		}
		err = survey.AskOne(prompt, &selectedParent, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
		if err != nil {
			// Handle potential ctrl+c from prompt etc.
			if err.Error() == "interrupt" {
				fmt.Println("Track command cancelled.")
				os.Exit(0) // Clean exit on interrupt
			}
			return fmt.Errorf("failed to get parent selection: %w", err)
		}

		// 5. Determine and store base branch
		selectedBase := ""
		if knownBases[selectedParent] {
			// Parent is a known base branch
			selectedBase = selectedParent
		} else {
			// Parent is another feature branch, check if IT is tracked
			parentBaseKey := fmt.Sprintf("branch.%s.socle-base", selectedParent)
			inheritedBase, err := gitutils.GetGitConfig(parentBaseKey)
			if err == nil && inheritedBase != "" {
				// Parent is tracked, inherit its base
				selectedBase = inheritedBase
				fmt.Printf("Parent '%s' is tracked with base '%s'. Inheriting base.\n", selectedParent, selectedBase)
			} else if err != nil && !strings.Contains(err.Error(), "exit status 1") {
				// Handle unexpected errors from git config
				return fmt.Errorf("failed to check tracking status for parent branch '%s': %w", selectedParent, err)
			} else {
				// Parent is not a known base AND not tracked. This is ambiguous.
				// For now, let's default to main, but ideally prompt or error.
				// Option A: Error out
				// return fmt.Errorf("parent branch '%s' is not a known base branch (main, develop) and is not tracked itself. Please track '%s' first", selectedParent, selectedParent)

				// Option B: Default to main (less safe but simpler for now)
				fmt.Printf("Warning: Parent branch '%s' is not tracked. Assuming stack base is 'main'.\n", selectedParent)
				fmt.Println("Consider tracking the parent branch first for more accurate stack definitions.")
				selectedBase = "main" // Default assumption

				// Option C: Prompt explicitly (requires another survey call)
				// basePrompt := &survey.Input{ Message: fmt.Sprintf("Parent '%s' is not tracked. What is the ultimate base branch for this stack?", selectedParent), Default: "main" }
				// survey.AskOne(basePrompt, &selectedBase, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
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

func init() {
	AddCommand(trackCmd)
}
