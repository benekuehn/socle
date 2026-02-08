package cmd

import (
	"fmt"
	"io"
	"log/slog"

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

	// Always use CurrentStack (even when multiple stacks originate at base) for downward navigation.
	if len(stackInfo.CurrentStack) == 0 {
		return fmt.Errorf("internal error: no current stack found for branch '%s'", stackInfo.CurrentBranch)
	}

	branch, msg, navErr := cmdutils.ComputeLinearTarget(stackInfo.CurrentBranch, stackInfo.CurrentStack, cmdutils.PurposeDown)
	if navErr != nil {
		return navErr
	}
	if branch == "" {
		if msg != "" {
			_, _ = fmt.Fprintf(r.stdout, "%s\n", msg)
		}
		return nil
	}
	return checkoutBranch(branch, stackInfo.CurrentBranch)
}
