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

// baseMultiStackSelection encapsulates the logic for selecting a stack when on a base branch
// that has multiple independent stacks. It supports test-only overrides via index or first-child name.
// If no override flags are set it prompts the user interactively.
// Returns (targetBranch, handled, error). If handled is true the caller should stop further processing.
func baseMultiStackSelection(
	logger *slog.Logger,
	stdout io.Writer,
	stderr io.Writer,
	stdin io.Reader,
	baseBranch string,
	testIndex int,
	testChild string,
	purpose string, // e.g. "up", "top", "bottom" to decide which branch to pick in stack
) (string, bool, error) {
	availableStacks, err := git.GetAvailableStacksFromBase(baseBranch)
	if err != nil {
		return "", true, fmt.Errorf("failed to get available stacks from base '%s': %w", baseBranch, err)
	}
	if len(availableStacks) == 0 {
		_, _ = fmt.Fprintf(stdout, "No stacks found starting from base branch '%s'.\n", baseBranch)
		return "", true, nil
	}

	// Helper to pick branch from selected stack based on purpose
	pickBranch := func(stack []string) (string, error) {
		if len(stack) <= 1 { // stack slice always includes base at [0]
			return "", fmt.Errorf("selected stack has no branches beyond the base")
		}
		switch purpose {
		case "up":
			return stack[1], nil // first child
		case "bottom":
			return stack[1], nil // same as up but semantics differ
		case "top":
			return stack[len(stack)-1], nil
		default:
			return "", fmt.Errorf("unknown purpose '%s'", purpose)
		}
	}

	// Test child override
	if testChild != "" {
		resolved := -1
		for i, s := range availableStacks {
			if len(s) > 1 && s[1] == testChild { // first child match
				resolved = i
				break
			}
		}
		if resolved == -1 {
			return "", true, fmt.Errorf("test-select-stack-child '%s' not found among stacks", testChild)
		}
		branch, errPick := pickBranch(availableStacks[resolved])
		if errPick != nil {
			_, _ = fmt.Fprintln(stdout, errPick.Error())
			return "", true, nil
		}
		logger.Debug("Test auto-select stack by child", "branch", branch, "purpose", purpose)
		return branch, true, nil
	}

	// Test index override
	if testIndex >= 0 {
		if testIndex >= len(availableStacks) {
			return "", true, fmt.Errorf("test-select-stack-index %d out of range (have %d stacks)", testIndex, len(availableStacks))
		}
		branch, errPick := pickBranch(availableStacks[testIndex])
		if errPick != nil {
			_, _ = fmt.Fprintln(stdout, errPick.Error())
			return "", true, nil
		}
		logger.Debug("Test auto-select stack by index", "branch", branch, "purpose", purpose)
		return branch, true, nil
	}

	// Interactive prompt path
	logger.Debug("Multiple stacks detected from base branch, prompting for selection")
	options := make([]string, len(availableStacks))
	for i, stack := range availableStacks {
		if len(stack) > 1 {
			// Show contextual descriptor depending on purpose
			count := len(stack) - 1
			switch purpose {
			case "top":
				topBranch := stack[len(stack)-1]
				options[i] = fmt.Sprintf("%s (top of stack with %d branches)", topBranch, count)
			default: // up/bottom both show first child
				options[i] = fmt.Sprintf("%s (stack with %d branches)", stack[1], count)
			}
		} else {
			options[i] = fmt.Sprintf("Stack %d", i+1)
		}
	}
	msg := fmt.Sprintf("Multiple stacks available from '%s'. Select a stack:", baseBranch)
	if purpose == "top" {
		msg = fmt.Sprintf("Multiple stacks available from '%s'. Select a stack to go to the top of:", baseBranch)
	}
	var selectedOption string
	prompt := &survey.Select{Message: msg, Options: options}
	err = survey.AskOne(prompt, &selectedOption, survey.WithStdio(stdin.(*os.File), stderr.(*os.File), stderr.(*os.File)))
	if err != nil {
		return "", true, ui.HandleSurveyInterrupt(err, "Navigation cancelled.")
	}
	selectedIndex := -1
	for i, opt := range options {
		if opt == selectedOption {
			selectedIndex = i
			break
		}
	}
	if selectedIndex == -1 {
		return "", true, fmt.Errorf("internal error: could not find selected option")
	}
	branch, errPick := pickBranch(availableStacks[selectedIndex])
	if errPick != nil {
		_, _ = fmt.Fprintln(stdout, errPick.Error())
		return "", true, nil
	}
	return branch, true, nil
}

// navigateLinear moves along a linear stack (either FullStack or CurrentStack) depending on direction purpose.
// purpose: up/top/bottom. Returns handled error semantics like above.
func navigateLinear(
	logger *slog.Logger,
	stdout io.Writer,
	currentBranch string,
	stack []string,
	purpose string,
) (string, bool, error) {
	if stack == nil || len(stack) == 0 {
		return "", true, fmt.Errorf("internal error: empty stack for branch '%s'", currentBranch)
	}
	idx := cmdutils.FindIndexInStack(currentBranch, stack)
	if idx == -1 {
		return "", true, fmt.Errorf("internal error: branch '%s' not found in stack %v", currentBranch, stack)
	}
	switch purpose {
	case "up":
		if idx == len(stack)-1 {
			_, _ = fmt.Fprintf(stdout, "Already on the top branch: '%s'.\n", currentBranch)
			return "", true, nil
		}
		return stack[idx+1], true, nil
	case "top":
		if idx == len(stack)-1 {
			_, _ = fmt.Fprintf(stdout, "Already on the top branch: '%s'\n", currentBranch)
			return "", true, nil
		}
		return stack[len(stack)-1], true, nil
	case "bottom":
		if len(stack) <= 1 {
			_, _ = fmt.Fprintf(stdout, "Already on the base branch '%s', which is the only branch in the stack.\n", currentBranch)
			return "", true, nil
		}
		if idx == 1 {
			_, _ = fmt.Fprintf(stdout, "Already on the bottom branch: '%s'\n", currentBranch)
			return "", true, nil
		}
		return stack[1], true, nil
	}
	return "", true, fmt.Errorf("unknown navigation purpose '%s'", purpose)
}

// checkoutBranch wraps git.CheckoutBranch with common error message logic.
func checkoutBranch(target string, current string) error {
	if err := git.CheckoutBranch(target); err != nil {
		if strings.Contains(err.Error(), "Please commit your changes or stash them") {
			return fmt.Errorf("cannot checkout branch '%s': uncommitted changes detected in '%s'. Please commit or stash them first", target, current)
		}
		return fmt.Errorf("failed to checkout branch '%s': %w", target, err)
	}
	return nil
}
