package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
)

type topCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
}

func (r *topCmdRunner) run() error {
	// 1. Get current branch and its stack info (including base)
	currentBranch, currentStack, baseBranch, err := git.GetCurrentStackInfo()
	if err != nil {
		// Return the original error (e.g., not tracked, inconsistent, etc.)
		return err
	}
	r.logger.Debug("Current info", "branch", currentBranch, "base", baseBranch, "stackToCurrent", currentStack)

	// 2. Get the full ordered stack lineage
	orderedStack, _, err := git.GetFullStack(currentStack)
	if err != nil {
		return fmt.Errorf("failed to determine full stack: %w", err)
	}
	r.logger.Debug("Retrieved full ordered stack", "stack", orderedStack)

	// If GetCurrentStackInfo succeeded, orderedStack should have at least the base.
	if len(orderedStack) == 0 {
		return fmt.Errorf("internal error: full stack is unexpectedly empty after GetFullStack succeeded")
	}

	topBranch := orderedStack[len(orderedStack)-1]
	r.logger.Debug("Identified absolute top branch of lineage", "topBranch", topBranch)

	// 4. Handle different scenarios
	switch currentBranch {
	case baseBranch:
		// User is on the base branch. Inform them, don't move.
		// Use a slightly different message if the stack only contains the base.
		var msg string
		if len(orderedStack) <= 1 {
			msg = fmt.Sprintf("Currently on the base branch '%s', which is the only branch in the stack.", baseBranch)
		} else {
			msg = fmt.Sprintf("Currently on the base branch '%s'. Use 'git checkout %s' or 'so create' to move up or create branches.", baseBranch, topBranch) // Suggest top branch
		}
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render(msg))
		return nil
	case topBranch:
		// User is already on the top branch.
		msg := fmt.Sprintf("Already on the top branch: '%s'", currentBranch)
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render(msg))
		return nil
	default:
		// User is on a middle branch, checkout top branch.
		r.logger.Debug("Checking out top branch", "branch", topBranch)
		if err := git.CheckoutBranch(topBranch); err != nil {
			// Check if checkout failed because of uncommitted changes
			if strings.Contains(err.Error(), "Please commit your changes or stash them") {
				return fmt.Errorf("cannot checkout top branch '%s': uncommitted changes detected in '%s'. Please commit or stash them first", topBranch, currentBranch)
			}
			return fmt.Errorf("failed to checkout top branch '%s': %w", topBranch, err)
		}

		msg := fmt.Sprintf("Checked out top branch: '%s'", topBranch)
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render(msg))
		return nil
	}
}
