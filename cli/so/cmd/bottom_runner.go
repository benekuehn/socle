package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/benekuehn/socle/cli/so/internal/cmdutils"
	"github.com/benekuehn/socle/cli/so/internal/git"
)

type bottomCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
}

func (r *bottomCmdRunner) run() error {
	stackInfo, err := git.GetStackInfo()
	if err != nil {
		_, _ = fmt.Fprintf(r.stdout, "Error getting stack info: %s\n", err)
		return nil
	}
	r.logger.Debug("Retrieved stack info", "currentBranch", stackInfo.CurrentBranch, "fullStack", stackInfo.FullStack)

	if len(stackInfo.FullStack) <= 1 {
		_, _ = fmt.Fprintf(r.stdout, "Already on the base branch '%s', which is the only branch in the stack.", stackInfo.CurrentBranch)
		return nil
	}

	currentIndex := cmdutils.FindIndexInStack(stackInfo.CurrentBranch, stackInfo.FullStack)
	if currentIndex == -1 {
		return fmt.Errorf("internal error: current branch '%s' not found in its full stack: %v", stackInfo.CurrentBranch, stackInfo.FullStack)
	}
	r.logger.Debug("Current branch index in full stack", "index", currentIndex)

	if currentIndex == 1 {
		_, _ = fmt.Fprintf(r.stdout, "Already on the bottom branch: '%s'\n", stackInfo.CurrentBranch)
		return nil
	}

	bottomBranch := stackInfo.FullStack[1]
	r.logger.Debug("Checking out bottom branch", "bottom", bottomBranch)

	if err := git.CheckoutBranch(bottomBranch); err != nil {
		if strings.Contains(err.Error(), "Please commit your changes or stash them") {
			return fmt.Errorf("cannot checkout bottom branch '%s': uncommitted changes detected in '%s'. Please commit or stash them first", bottomBranch, stackInfo.CurrentBranch)
		}
		return fmt.Errorf("failed to checkout bottom branch '%s': %w", bottomBranch, err)
	}
	return nil
}
