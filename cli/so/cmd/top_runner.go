package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/benekuehn/socle/cli/so/internal/cmdutils"
	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
)

type topCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader
}

func (r *topCmdRunner) run() error {
	stackInfo, err := git.GetStackInfo()
	if err != nil {
		_, _ = fmt.Fprintf(r.stdout, "Error getting stack info: %s\n", err)
		return nil
	}
	r.logger.Debug("Retrieved stack info", "currentBranch", stackInfo.CurrentBranch, "fullStack", stackInfo.FullStack)

	// Handle case where we're on a base branch with multiple stacks
	if stackInfo.FullStack == nil {
		r.logger.Debug("Multiple stacks detected from base branch, prompting for selection")
		return r.handleMultipleStackSelection(stackInfo)
	}

	currentIndex := cmdutils.FindIndexInStack(stackInfo.CurrentBranch, stackInfo.FullStack)
	if currentIndex == -1 {
		return fmt.Errorf("internal error: current branch '%s' not found in its full stack: %v", stackInfo.CurrentBranch, stackInfo.FullStack)
	}
	r.logger.Debug("Current branch index in full stack", "index", currentIndex)

	if currentIndex == len(stackInfo.FullStack)-1 {
		_, _ = fmt.Fprintf(r.stdout, "Already on the top branch: '%s'\n", stackInfo.CurrentBranch)
		return nil
	}

	topBranch := stackInfo.FullStack[len(stackInfo.FullStack)-1]
	r.logger.Debug("Identified top branch", "topBranch", topBranch)

	if err := git.CheckoutBranch(topBranch); err != nil {
		if strings.Contains(err.Error(), "Please commit your changes or stash them") {
			return fmt.Errorf("cannot checkout top branch '%s': uncommitted changes detected in '%s'. Please commit or stash them first", topBranch, stackInfo.CurrentBranch)
		}
		return fmt.Errorf("failed to checkout top branch '%s': %w", topBranch, err)
	}
	return nil
}

func (r *topCmdRunner) handleMultipleStackSelection(stackInfo *git.StackInfo) error {
	// Get available stacks from the current base
	availableStacks, err := git.GetAvailableStacksFromBase(stackInfo.CurrentBranch)
	if err != nil {
		return fmt.Errorf("failed to get available stacks from base '%s': %w", stackInfo.CurrentBranch, err)
	}

	if len(availableStacks) == 0 {
		_, _ = fmt.Fprintf(r.stdout, "No stacks found starting from base branch '%s'.\n", stackInfo.CurrentBranch)
		return nil
	}

	// Build options for selection
	options := make([]string, len(availableStacks))
	for i, stack := range availableStacks {
		// Show the top branch of each stack as the option
		if len(stack) > 1 {
			topBranch := stack[len(stack)-1]
			options[i] = fmt.Sprintf("%s (top of stack with %d branches)", topBranch, len(stack)-1)
		} else {
			options[i] = fmt.Sprintf("Stack %d (no branches)", i+1)
		}
	}

	// Prompt user for selection
	var selectedOption string
	prompt := &survey.Select{
		Message: fmt.Sprintf("Multiple stacks available from '%s'. Select a stack to go to the top of:", stackInfo.CurrentBranch),
		Options: options,
	}
	
	err = survey.AskOne(prompt, &selectedOption, survey.WithStdio(r.stdin.(*os.File), r.stderr.(*os.File), r.stderr.(*os.File)))
	if err != nil {
		return ui.HandleSurveyInterrupt(err, "Navigation cancelled.")
	}

	// Find selected stack index
	selectedIndex := -1
	for i, option := range options {
		if option == selectedOption {
			selectedIndex = i
			break
		}
	}

	if selectedIndex == -1 {
		return fmt.Errorf("internal error: could not find selected option")
	}

	// Navigate to top branch in selected stack
	selectedStack := availableStacks[selectedIndex]
	if len(selectedStack) <= 1 {
		_, _ = fmt.Fprintf(r.stdout, "Selected stack has no branches beyond the base.\n")
		return nil
	}

	targetBranch := selectedStack[len(selectedStack)-1] // Top branch of the stack
	r.logger.Debug("Checking out selected stack top branch", "branch", targetBranch, "stack", selectedStack)

	if err := git.CheckoutBranch(targetBranch); err != nil {
		if strings.Contains(err.Error(), "Please commit your changes or stash them") {
			return fmt.Errorf("cannot checkout branch '%s': uncommitted changes detected in '%s'. Please commit or stash them first", targetBranch, stackInfo.CurrentBranch)
		}
		return fmt.Errorf("failed to checkout branch '%s': %w", targetBranch, err)
	}

	return nil
}
