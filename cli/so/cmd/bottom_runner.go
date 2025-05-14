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
	// Get complete stack info in one call
	stackInfo, err := git.GetStackInfo()
	if err != nil {
		// Handle untracked error specifically for a potentially better message?
		if strings.Contains(err.Error(), "not tracked by socle") {
			// Or just return err as GetStackInfo gives a good message?
			// Let's return a slightly different error for testing purposes, following convention.
			return fmt.Errorf("not on a tracked branch stack")
		}
		return err // Other errors (not git repo, inconsistent tracking)
	}
	r.logger.Debug("Retrieved stack info", "currentBranch", stackInfo.CurrentBranch, "fullStack", stackInfo.FullStack)

	// Identify the bottom branch (branch after base)
	if len(stackInfo.FullStack) <= 1 {
		// Only the base branch exists in the stack.
		msg := fmt.Sprintf("Already on the base branch '%s', which is the only branch in the stack.", stackInfo.CurrentBranch)
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render(msg))
		return nil
	}

	// Bottom branch is the one at index 1 (after base at index 0)
	bottomBranch := stackInfo.FullStack[1]
	r.logger.Debug("Identified bottom branch", "bottomBranch", bottomBranch)

	// Handle different scenarios
	if stackInfo.CurrentBranch == bottomBranch {
		msg := fmt.Sprintf("Already on the bottom branch: '%s'", stackInfo.CurrentBranch)
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render(msg))
		return nil
	}

	// Checkout the bottom branch (works if current is base or higher up)
	r.logger.Debug("Checking out bottom branch", "branch", bottomBranch)
	if err := git.CheckoutBranch(bottomBranch); err != nil {
		// Check for uncommitted changes error
		if strings.Contains(err.Error(), "Please commit your changes or stash them") {
			return fmt.Errorf("cannot checkout bottom branch '%s': uncommitted changes detected in '%s'. Please commit or stash them first", bottomBranch, stackInfo.CurrentBranch)
		}
		return fmt.Errorf("failed to checkout bottom branch '%s': %w", bottomBranch, err)
	}

	msg := fmt.Sprintf("Checked out bottom branch: '%s'", bottomBranch)
	_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render(msg))
	return nil
}
