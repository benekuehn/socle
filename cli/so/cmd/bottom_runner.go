package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
)

type bottomCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
}

func (r *bottomCmdRunner) run() error {
	// 1. Get current branch and stack info
	currentBranch, currentStack, _, err := git.GetCurrentStackInfo()
	if err != nil {
		// Handle untracked error specifically for a potentially better message?
		if strings.Contains(err.Error(), "not tracked by socle") {
			// Or just return err as GetCurrentStackInfo gives a good message?
			// Let's return a slightly different error for testing purposes, following convention.
			return fmt.Errorf("not on a tracked branch stack")
		}
		return err // Other errors (not git repo, inconsistent tracking)
	}
	r.logger.Debug("Current info", "branch", currentBranch, "stackToCurrent", currentStack)

	// 2. Get the full ordered stack
	orderedStack, _, err := git.GetFullStack(currentStack)
	if err != nil {
		return fmt.Errorf("failed to determine full stack: %w", err)
	}
	r.logger.Debug("Retrieved full ordered stack", "stack", orderedStack)

	// 3. Identify the bottom branch (branch after base)
	if len(orderedStack) <= 1 {
		// Only the base branch exists in the stack.
		msg := fmt.Sprintf("Already on the base branch '%s', which is the only branch in the stack.", currentBranch)
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render(msg))
		return nil
	}

	// Bottom branch is the one at index 1 (after base at index 0)
	bottomBranch := orderedStack[1]
	r.logger.Debug("Identified bottom branch", "bottomBranch", bottomBranch)

	// 4. Handle different scenarios
	if currentBranch == bottomBranch {
		msg := fmt.Sprintf("Already on the bottom branch: '%s'", currentBranch)
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render(msg))
		return nil
	}

	// Checkout the bottom branch (works if current is base or higher up)
	r.logger.Debug("Checking out bottom branch", "branch", bottomBranch)
	if err := git.CheckoutBranch(bottomBranch); err != nil {
		// Check for uncommitted changes error
		if strings.Contains(err.Error(), "Please commit your changes or stash them") {
			return fmt.Errorf("cannot checkout bottom branch '%s': uncommitted changes detected in '%s'. Please commit or stash them first", bottomBranch, currentBranch)
		}
		return fmt.Errorf("failed to checkout bottom branch '%s': %w", bottomBranch, err)
	}

	msg := fmt.Sprintf("Checked out bottom branch: '%s'", bottomBranch)
	_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render(msg))
	return nil
}
