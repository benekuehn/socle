package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/benekuehn/socle/cli/so/internal/cmdutils"
	"github.com/benekuehn/socle/cli/so/internal/git"
)

type upCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
}

func (r *upCmdRunner) run() error {
	stackInfo, err := git.GetStackInfo()
	if err != nil {
		_, _ = fmt.Fprintf(r.stdout, "Error getting stack info: %s\n", err)
		return nil
	}
	r.logger.Debug("Retrieved stack info", "currentBranch", stackInfo.CurrentBranch, "fullStack", stackInfo.FullStack)

	currentIndex := cmdutils.FindIndexInStack(stackInfo.CurrentBranch, stackInfo.FullStack)
	if currentIndex == -1 {
		return fmt.Errorf("internal error: current branch '%s' not found in its full stack: %v", stackInfo.CurrentBranch, stackInfo.FullStack)
	}
	r.logger.Debug("Current branch index in full stack", "index", currentIndex)

	if currentIndex == len(stackInfo.FullStack)-1 {
		_, _ = fmt.Fprintf(r.stdout, "Already on the top branch: '%s'.", stackInfo.CurrentBranch)
		return nil
	}

	childBranch := stackInfo.FullStack[currentIndex+1]
	r.logger.Debug("Checking out child branch", "child", childBranch)

	if err := git.CheckoutBranch(childBranch); err != nil {
		if strings.Contains(err.Error(), "Please commit your changes or stash them") {
			return fmt.Errorf("cannot checkout child branch '%s': uncommitted changes detected in '%s'. Please commit or stash them first", childBranch, stackInfo.CurrentBranch)
		}
		return fmt.Errorf("failed to checkout child branch '%s': %w", childBranch, err)
	}
	return nil
}
