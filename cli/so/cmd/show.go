package cmd

import (
	"fmt"
	"strings"

	"github.com/benekuehn/socle/cli/so/gitutils"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Display the current tracked stack of branches",
	Long: `Shows the sequence of tracked branches leading from the stack's base
branch to the current branch, based on metadata set by 'socle track'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Get Current Branch
		currentBranch, err := gitutils.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		// 2. Check if current branch is tracked and get its base
		parentConfigKey := fmt.Sprintf("branch.%s.socle-parent", currentBranch)
		baseConfigKey := fmt.Sprintf("branch.%s.socle-base", currentBranch)

		currentBranchParent, errParent := gitutils.GetGitConfig(parentConfigKey)
		currentBranchBase, errBase := gitutils.GetGitConfig(baseConfigKey)

		// Handle errors specifically for non-tracking
		isUntracked := false
		if errParent != nil && strings.Contains(errParent.Error(), "exit status 1") {
			isUntracked = true
		}
		if errBase != nil && strings.Contains(errBase.Error(), "exit status 1") {
			isUntracked = true
		}

		// Handle other unexpected errors
		if errParent != nil && !strings.Contains(errParent.Error(), "exit status 1") {
			return fmt.Errorf("failed to read tracking parent for '%s': %w", currentBranch, errParent)
		}
		if errBase != nil && !strings.Contains(errBase.Error(), "exit status 1") {
			return fmt.Errorf("failed to read tracking base for '%s': %w", currentBranch, errBase)
		}

		if isUntracked || currentBranchParent == "" || currentBranchBase == "" {
			// Check if it's actually a known base branch
			knownBases := map[string]bool{"main": true, "master": true, "develop": true} // TODO: Configurable
			if knownBases[currentBranch] {
				fmt.Printf("Currently on the base branch '%s'.\n", currentBranch)
				// Optionally, list stacks starting from here? (Advanced)
				// For now, just indicate we're on the base.
			} else {
				fmt.Printf("Branch '%s' is not currently tracked by socle.\n", currentBranch)
				fmt.Println("Use 'socle track' to associate it with a parent branch and start a stack.")
			}
			return nil // Not an error, just informational exit.
		}

		fmt.Printf("Current branch: %s (Stack base: %s)\n", currentBranch, currentBranchBase)

		// 3. Build the stack by walking up the parents
		stack := []string{currentBranch}
		parent := currentBranchParent

		for {
			if parent == currentBranchBase {
				break // We've reached the base
			}

			// Prepend parent to stack
			stack = append([]string{parent}, stack...)

			// Find the next parent
			nextParentKey := fmt.Sprintf("branch.%s.socle-parent", parent)
			nextParent, err := gitutils.GetGitConfig(nextParentKey)
			if err != nil || nextParent == "" {
				// Error or untracked parent - stack definition is broken or incomplete
				fmt.Printf("\nWarning: Tracking information incomplete or broken at branch '%s'.\n", parent)
				fmt.Printf("Its parent ('%s') could not be determined from git config.\n", nextParentKey)
				fmt.Println("Stack trace might be incomplete.")
				break // Stop tracing here
			}

			parent = nextParent

			// Safety break to prevent infinite loops in case of cyclic metadata
			if len(stack) > 50 { // Arbitrary limit
				fmt.Println("\nWarning: Stack trace exceeds 50 levels. Assuming cycle or error.")
				break
			}
		}

		// 4. Print the stack
		fmt.Println("\nCurrent Stack:")
		fmt.Printf("  %s (base)\n", currentBranchBase)
		for _, branchName := range stack {
			marker := ""
			if branchName == currentBranch {
				marker = " *" // Mark the current branch
			}
			fmt.Printf("  -> %s%s\n", branchName, marker)
		}

		return nil
	},
}

func init() {
	AddCommand(showCmd)
}
