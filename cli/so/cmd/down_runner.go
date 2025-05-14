package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/benekuehn/socle/cli/so/internal/cmdutils"
	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
)

type downCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
}

// findIndexInStack finds the position of branch in the full stack
func findIndexInStack(branch string, stack []string) int {
	for i, name := range stack {
		if name == branch {
			return i
		}
	}
	return -1 // Not found
}

func (r *downCmdRunner) run() error {
	// 1. Get current branch first (best effort, for error handling)
	currentBranch, _ := git.GetCurrentBranch()

	// 2. Get complete stack info in one call
	stackInfo, err := git.GetStackInfo()

	// 3. Let HandleStartupError manage potential error classification
	handled, processedErr := cmdutils.HandleStartupError(err, currentBranch, r.stdout, r.stderr)
	if processedErr != nil {
		// For down command tests, specific handling for base branch and untracked branch scenarios
		if strings.Contains(processedErr.Error(), "not tracked by socle") {
			// For "untracked branch" test case
			return processedErr // Don't print anything to stdout
		}
		return processedErr
	}
	if handled {
		return nil
	}

	// 4. If we reach here and have a nil stackInfo, it must be a base branch
	if stackInfo == nil || len(stackInfo.FullStack) <= 1 {
		// We're on a base branch - match exact test string
		_, _ = fmt.Fprintf(r.stdout, "Already on the base branch '%s'. Cannot go down further.\n", currentBranch)
		return nil
	}

	// Find current position in stack
	currentIndex := findIndexInStack(stackInfo.CurrentBranch, stackInfo.FullStack)
	if currentIndex == -1 {
		return fmt.Errorf("internal error: current branch '%s' not found in its stack", stackInfo.CurrentBranch)
	}

	// Check if we're already at the base
	if currentIndex == 0 {
		msg := fmt.Sprintf("Already on the base branch: '%s'", stackInfo.CurrentBranch)
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render(msg))
		return nil
	}

	// Target is one level down in the stack (toward base)
	parentBranch := stackInfo.FullStack[currentIndex-1]
	r.logger.Debug("Moving down to parent branch", "parent", parentBranch)

	// Checkout the parent branch
	if err := git.CheckoutBranch(parentBranch); err != nil {
		// Check for uncommitted changes error
		if strings.Contains(err.Error(), "Please commit your changes or stash them") {
			return fmt.Errorf("cannot checkout parent branch '%s': uncommitted changes detected in '%s'. Please commit or stash them first", parentBranch, stackInfo.CurrentBranch)
		}
		return fmt.Errorf("failed to checkout parent branch '%s': %w", parentBranch, err)
	}

	msg := fmt.Sprintf("Checked out parent branch: '%s'", parentBranch)
	_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render(msg))
	return nil
}
