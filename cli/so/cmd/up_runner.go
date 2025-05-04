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
	// 1. Get current branch and stack info
	currentBranch, currentStack, _, err := git.GetCurrentStackInfo()
	if err != nil {
		// Let GetCurrentStackInfo handle untracked errors etc.
		return err
	}
	r.logger.Debug("Current info", "branch", currentBranch, "stackToCurrent", currentStack)

	// 2. Get the full ordered stack
	orderedStack, _, err := git.GetFullStack(currentStack)
	if err != nil {
		return fmt.Errorf("failed to determine full stack: %w", err)
	}
	r.logger.Debug("Retrieved full ordered stack", "stack", orderedStack)

	// 3. Find the current branch's index in the full ordered stack
	currentIndex := findIndex(orderedStack, currentBranch)
	if currentIndex == -1 {
		// Should not happen if GetFullStack is correct
		return fmt.Errorf("internal error: current branch '%s' not found in its full stack: %v", currentBranch, orderedStack)
	}
	r.logger.Debug("Current branch index in full stack", "index", currentIndex)

	// 4. Check if we can go down (towards top/tip)
	if currentIndex == len(orderedStack)-1 {
		// Current branch is the top-most branch
		msg := fmt.Sprintf("Already on the top branch: '%s'.", currentBranch)
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render(msg))
		return nil
	}

	// 5. Identify and checkout the child branch (next in the ordered stack)
	childIndex := currentIndex + 1
	if childIndex < 0 || childIndex >= len(orderedStack) {
		// Should be impossible given the top branch check above, but safeguard
		return fmt.Errorf("internal error: calculated invalid child index %d for stack %v", childIndex, orderedStack)
	}
	childBranch := orderedStack[childIndex]
	r.logger.Debug("Identified child branch", "childBranch", childBranch)

	// Checkout child
	r.logger.Debug("Checking out child branch", "branch", childBranch)
	if err := git.CheckoutBranch(childBranch); err != nil {
		// Check for uncommitted changes error
		if strings.Contains(err.Error(), "Please commit your changes or stash them") {
			return fmt.Errorf("cannot checkout child branch '%s': uncommitted changes detected in '%s'. Please commit or stash them first", childBranch, currentBranch)
		}
		return fmt.Errorf("failed to checkout child branch '%s': %w", childBranch, err)
	}

	msg := fmt.Sprintf("Checked out child branch: '%s'", childBranch)
	_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render(msg))
	return nil
}
