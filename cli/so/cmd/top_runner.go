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

type topCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
}

func (r *topCmdRunner) run() error {
	// 1. Get current branch first (best effort, for error handling)
	currentBranch, _ := git.GetCurrentBranch()

	// 2. Get complete stack info in one call
	stackInfo, err := git.GetStackInfo()

	// 3. Handle startup errors (like untracked branch)
	_, processedErr := cmdutils.HandleStartupError(err, currentBranch, r.stdout, r.stderr)
	if processedErr != nil {
		// For untracked branch, we want to return the error directly
		if strings.Contains(processedErr.Error(), "not tracked by socle") {
			return processedErr // Don't print to stdout
		}
		return processedErr
	}

	// 4. Check if we're on a base branch
	if isBase := (stackInfo != nil && stackInfo.CurrentBranch == stackInfo.BaseBranch); isBase {
		// On base branch - check for children
		parentMap, _ := git.GetAllSocleParents()
		childMap := git.BuildChildMap(parentMap)

		if children, hasChildren := childMap[currentBranch]; hasChildren && len(children) > 0 {
			// There is at least one child - show the message with checkout suggestion
			msg := fmt.Sprintf("Currently on the base branch '%s'. Use 'git checkout %s'", currentBranch, children[0])
			_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render(msg))
			return nil
		}
	}

	// 5. Check if the stack is trivial (only base or empty)
	if stackInfo == nil || len(stackInfo.FullStack) <= 1 {
		msg := fmt.Sprintf("Currently on the base branch '%s', which is the only branch in the stack.", currentBranch)
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render(msg))
		return nil
	}

	// 6. Top branch is the last one in the ordered stack
	topBranch := stackInfo.FullStack[len(stackInfo.FullStack)-1]
	r.logger.Debug("Identified top branch", "topBranch", topBranch)

	// 7. Handle different scenarios
	if stackInfo.CurrentBranch == topBranch {
		msg := fmt.Sprintf("Already on the top branch: '%s'", stackInfo.CurrentBranch)
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render(msg))
		return nil
	}

	// 8. Checkout the top branch (works if current is base or anywhere in the stack)
	r.logger.Debug("Checking out top branch", "branch", topBranch)
	if err := git.CheckoutBranch(topBranch); err != nil {
		// Check for uncommitted changes error
		if strings.Contains(err.Error(), "Please commit your changes or stash them") {
			return fmt.Errorf("cannot checkout top branch '%s': uncommitted changes detected in '%s'. Please commit or stash them first", topBranch, stackInfo.CurrentBranch)
		}
		return fmt.Errorf("failed to checkout top branch '%s': %w", topBranch, err)
	}

	msg := fmt.Sprintf("Checked out top branch: '%s'", topBranch)
	_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render(msg))
	return nil
}
