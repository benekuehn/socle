package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/benekuehn/socle/cli/so/internal/cmdutils"
	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
)

type bottomCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader
}

func (r *bottomCmdRunner) run() error {
	stackInfo, err := git.GetStackInfo()
	if err != nil {
		_, _ = fmt.Fprintf(r.stdout, "Error getting stack info: %s\n", err)
		return nil
	}
	r.logger.Debug("Retrieved stack info", "currentBranch", stackInfo.CurrentBranch, "fullStack", stackInfo.FullStack)

	// CASE 1: On base in multi-stack environment -> selection
	if stackInfo.FullStack == nil && stackInfo.CurrentBranch == stackInfo.BaseBranch {
		if target, handled, selErr := cmdutils.ResolveTestStackSelection(stackInfo.CurrentBranch, cmdutils.PurposeBottom, testSelectStackIndexBottom, testSelectStackChildBottom); handled {
			if selErr != nil {
				return selErr
			}
			if target == "" {
				return nil
			}
			return checkoutBranch(target, stackInfo.CurrentBranch)
		}
		branch, _, errSel := r.promptSelectStack(stackInfo.CurrentBranch, cmdutils.PurposeBottom)
		if errSel != nil {
			return errSel
		}
		if branch == "" {
			return nil
		}
		return checkoutBranch(branch, stackInfo.CurrentBranch)
	}

	// CASE 2: Inside lineage (multi-stack env) FullStack nil -> use CurrentStack
	if stackInfo.FullStack == nil {
		branch, msg, navErr := cmdutils.ComputeLinearTarget(stackInfo.CurrentBranch, stackInfo.CurrentStack, cmdutils.PurposeBottom)
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

	// CASE 3: Standard linear stack
	branch, msg, navErr := cmdutils.ComputeLinearTarget(stackInfo.CurrentBranch, stackInfo.FullStack, cmdutils.PurposeBottom)
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

// promptSelectStack provides interactive stack selection using shared utilities.
func (r *bottomCmdRunner) promptSelectStack(baseBranch string, purpose cmdutils.NavigationPurpose) (string, bool, error) {
	options, stacks, err := cmdutils.BuildStackSelectionOptions(baseBranch, purpose)
	if err != nil {
		return "", true, err
	}
	if len(stacks) == 0 {
		_, _ = fmt.Fprintf(r.stdout, "No stacks found starting from base branch '%s'.\n", baseBranch)
		return "", true, nil
	}
	var selectedOption string
	prompt := &survey.Select{Message: fmt.Sprintf("Multiple stacks available from '%s'. Select a stack:", baseBranch), Options: options}
	err = survey.AskOne(prompt, &selectedOption, survey.WithStdio(r.stdin.(*os.File), r.stderr.(*os.File), r.stderr.(*os.File)))
	if err != nil {
		return "", true, ui.HandleSurveyInterrupt(err, "Navigation cancelled.")
	}
	idx := -1
	for i, opt := range options {
		if opt == selectedOption {
			idx = i
			break
		}
	}
	if idx == -1 {
		return "", true, fmt.Errorf("internal error: could not find selected option")
	}
	branch, pickErr := cmdutils.PickBranchFromStack(stacks[idx], purpose)
	if pickErr != nil {
		_, _ = fmt.Fprintln(r.stdout, pickErr.Error())
		return "", true, nil
	}
	return branch, true, nil
}
