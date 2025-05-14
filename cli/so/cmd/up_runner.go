package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
)

type upCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
}

// findIndex finds the index of a string in a slice, returns -1 if not found.
func findIndex(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

func (r *upCmdRunner) run() error {
	// Get the complete stack info in one call
	stackInfo, err := git.GetStackInfo()
	if err != nil {
		// Let GetStackInfo handle untracked errors etc.
		return err
	}
	r.logger.Debug("Retrieved stack info", "currentBranch", stackInfo.CurrentBranch, "fullStack", stackInfo.FullStack)

	// Find the current branch's index in the full ordered stack
	currentIndex := findIndex(stackInfo.FullStack, stackInfo.CurrentBranch)
	if currentIndex == -1 {
		// Should not happen if GetStackInfo is correct
		return fmt.Errorf("internal error: current branch '%s' not found in its full stack: %v", stackInfo.CurrentBranch, stackInfo.FullStack)
	}
	r.logger.Debug("Current branch index in full stack", "index", currentIndex)

	// Check if we can go down (towards top/tip)
	if currentIndex == len(stackInfo.FullStack)-1 {
		// Current branch is the top-most branch
		msg := fmt.Sprintf("Already on the top branch: '%s'.", stackInfo.CurrentBranch)
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render(msg))
		return nil
	}

	// Identify and checkout the child branch (next in the ordered stack)
	childIndex := currentIndex + 1
	if childIndex < 0 || childIndex >= len(stackInfo.FullStack) {
		// Should be impossible given the top branch check above, but safeguard
		return fmt.Errorf("internal error: calculated invalid child index %d for stack %v", childIndex, stackInfo.FullStack)
	}
	childBranch := stackInfo.FullStack[childIndex]
	r.logger.Debug("Identified child branch", "childBranch", childBranch)

	// Checkout child
	r.logger.Debug("Checking out child branch", "branch", childBranch)
	if err := git.CheckoutBranch(childBranch); err != nil {
		// Check for uncommitted changes error
		if strings.Contains(err.Error(), "Please commit your changes or stash them") {
			return fmt.Errorf("cannot checkout child branch '%s': uncommitted changes detected in '%s'. Please commit or stash them first", childBranch, stackInfo.CurrentBranch)
		}
		return fmt.Errorf("failed to checkout child branch '%s': %w", childBranch, err)
	}

	msg := fmt.Sprintf("Checked out child branch: '%s'", childBranch)
	_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render(msg))
	return nil
}
