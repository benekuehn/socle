package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
)

type downCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
}

// findIndex utility (can be shared if moved to a util package later)
func findIndexDown(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

func (r *downCmdRunner) run() error {
	// 1. Get current branch and its stack info (up to current)
	currentBranch, currentStack, baseBranch, err := git.GetCurrentStackInfo()
	if err != nil {
		// Let GetCurrentStackInfo provide the detailed error (untracked, etc.)
		return err
	}
	r.logger.Debug("Current info", "branch", currentBranch, "base", baseBranch, "stackToCurrent", currentStack)

	// 2. Find the current branch's index in the stack provided by GetCurrentStackInfo
	currentIndex := findIndexDown(currentStack, currentBranch)
	if currentIndex == -1 {
		// This shouldn't happen if GetCurrentStackInfo succeeded
		return fmt.Errorf("internal error: current branch '%s' not found in its own stack: %v", currentBranch, currentStack)
	}
	r.logger.Debug("Current branch index", "index", currentIndex)

	// 3. Check if we can go down (towards base)
	if currentIndex == 0 {
		// Current branch is the base branch
		msg := fmt.Sprintf("Already on the base branch '%s'. Cannot go down further.", currentBranch)
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render(msg))
		return nil
	}

	// 4. Identify and checkout the parent branch (which is index - 1 in this stack)
	parentIndex := currentIndex - 1
	if parentIndex < 0 { // Simplified check as currentIndex > 0 from above
		// Should be impossible given the currentIndex > 0 check above, but safeguard
		return fmt.Errorf("internal error: calculated invalid parent index %d for stack %v", parentIndex, currentStack)
	}
	parentBranch := currentStack[parentIndex]
	r.logger.Debug("Identified parent branch", "parentBranch", parentBranch)

	// Checkout parent
	r.logger.Debug("Checking out parent branch", "branch", parentBranch)
	if err := git.CheckoutBranch(parentBranch); err != nil {
		// Check for uncommitted changes error
		if strings.Contains(err.Error(), "Please commit your changes or stash them") {
			return fmt.Errorf("cannot checkout parent branch '%s': uncommitted changes detected in '%s'. Please commit or stash them first", parentBranch, currentBranch)
		}
		return fmt.Errorf("failed to checkout parent branch '%s': %w", parentBranch, err)
	}

	msg := fmt.Sprintf("Checked out parent branch: '%s'", parentBranch)
	_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render(msg))
	return nil
}
