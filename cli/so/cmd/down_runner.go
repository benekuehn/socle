package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/benekuehn/socle/cli/so/internal/cmdutils"
	"github.com/benekuehn/socle/cli/so/internal/git"
)

type downCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
}

func (r *downCmdRunner) run() error {
	stackInfo, err := git.GetStackInfo()
	if err != nil {
		_, _ = fmt.Fprintf(r.stdout, "Error getting stack info: %s\n", err)
		return nil
	}
	r.logger.Debug("Retrieved stack info", "currentBranch", stackInfo.CurrentBranch, "fullStack", stackInfo.FullStack)

	// Use CurrentStack for navigation since it's always populated correctly
	// even when FullStack is nil (multiple stacks from base)
	stackToUse := stackInfo.CurrentStack
	if stackToUse == nil || len(stackToUse) == 0 {
		return fmt.Errorf("internal error: no current stack found for branch '%s'", stackInfo.CurrentBranch)
	}

	currentIndex := cmdutils.FindIndexInStack(stackInfo.CurrentBranch, stackToUse)
	if currentIndex == -1 {
		return fmt.Errorf("internal error: current branch '%s' not found in its current stack: %v", stackInfo.CurrentBranch, stackToUse)
	}
	r.logger.Debug("Current branch index in current stack", "index", currentIndex)

	if currentIndex == 0 {
		_, _ = fmt.Fprintf(r.stdout, "Already on the base branch '%s'. Cannot go down further.\n", stackInfo.CurrentBranch)
		return nil
	}

	parentBranch := stackToUse[currentIndex-1]
	r.logger.Debug("Checking out parent branch", "parent", parentBranch)

	if err := git.CheckoutBranch(parentBranch); err != nil {
		if strings.Contains(err.Error(), "Please commit your changes or stash them") {
			return fmt.Errorf("cannot checkout parent branch '%s': uncommitted changes detected in '%s'. Please commit or stash them first", parentBranch, stackInfo.CurrentBranch)
		}
		return fmt.Errorf("failed to checkout parent branch '%s': %w", parentBranch, err)
	}
	return nil
}
