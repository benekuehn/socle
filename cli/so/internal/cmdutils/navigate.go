package cmdutils

import (
	"fmt"
	"github.com/benekuehn/socle/cli/so/internal/git"
)

// FindIndexInStack finds the index of a branch in a stack
func FindIndexInStack(branch string, stack []string) int {
	for i, name := range stack {
		if name == branch {
			return i
		}
	}
	return -1 // Not found
}

// NavigationPurpose describes the intent for determining a target branch.
type NavigationPurpose string

const (
	PurposeUp     NavigationPurpose = "up"
	PurposeTop    NavigationPurpose = "top"
	PurposeBottom NavigationPurpose = "bottom"
	PurposeDown   NavigationPurpose = "down"
)

// ErrMultipleStacksBase is returned when attempting to auto-compute a navigation target
// while on a base branch that has multiple child stacks (requires user selection).
var ErrMultipleStacksBase = fmt.Errorf("multiple stacks originate from base; selection required")

// ComputeLinearTarget determines the next target branch for up/top/bottom navigation within a linear stack.
// Returns targetBranch (empty if already at destination), and a message to show when already at boundary.
func ComputeLinearTarget(currentBranch string, stack []string, purpose NavigationPurpose) (targetBranch string, alreadyMsg string, err error) {
	if stack == nil || len(stack) == 0 {
		return "", "", fmt.Errorf("empty stack for branch '%s'", currentBranch)
	}
	idx := FindIndexInStack(currentBranch, stack)
	if idx == -1 {
		return "", "", fmt.Errorf("branch '%s' not found in stack %v", currentBranch, stack)
	}
	switch purpose {
	case PurposeUp:
		if idx == len(stack)-1 {
			return "", fmt.Sprintf("Already on the top branch: '%s'.", currentBranch), nil
		}
		return stack[idx+1], "", nil
	case PurposeDown:
		if idx == 0 { // base
			return "", fmt.Sprintf("Already on the base branch '%s'. Cannot go down further.", currentBranch), nil
		}
		return stack[idx-1], "", nil
	case PurposeTop:
		if idx == len(stack)-1 {
			return "", fmt.Sprintf("Already on the top branch: '%s'", currentBranch), nil
		}
		return stack[len(stack)-1], "", nil
	case PurposeBottom:
		if len(stack) <= 1 { // base only
			return "", fmt.Sprintf("Already on the base branch '%s', which is the only branch in the stack.", currentBranch), nil
		}
		if idx == 1 { // already at bottom (first after base)
			return "", fmt.Sprintf("Already on the bottom branch: '%s'", currentBranch), nil
		}
		return stack[1], "", nil
	default:
		return "", "", fmt.Errorf("unknown navigation purpose '%s'", purpose)
	}
}

// PickBranchFromStack selects the appropriate branch from a full stack (base included at index 0)
// according to the navigation purpose (used after a user or test chooses a stack).
func PickBranchFromStack(stack []string, purpose NavigationPurpose) (string, error) {
	if len(stack) <= 1 {
		return "", fmt.Errorf("selected stack has no branches beyond the base")
	}
	switch purpose {
	case PurposeUp, PurposeBottom:
		return stack[1], nil // first child
	case PurposeTop:
		return stack[len(stack)-1], nil
	default:
		return "", fmt.Errorf("unknown navigation purpose '%s'", purpose)
	}
}

// ResolveTestStackSelection resolves test-only selection overrides.
// Returns targetBranch (maybe empty), handled indicates whether caller should stop further processing.
func ResolveTestStackSelection(baseBranch string, purpose NavigationPurpose, testIndex int, testChild string) (targetBranch string, handled bool, err error) {
	if testIndex < 0 && testChild == "" {
		return "", false, nil // nothing to do
	}
	stacks, err := git.GetAvailableStacksFromBase(baseBranch)
	if err != nil {
		return "", true, fmt.Errorf("failed to get available stacks from base '%s': %w", baseBranch, err)
	}
	if len(stacks) == 0 {
		return "", true, nil // no stacks to choose
	}
	if testChild != "" {
		for _, s := range stacks {
			if len(s) > 1 && s[1] == testChild {
				b, errPick := PickBranchFromStack(s, purpose)
				return b, true, errPick
			}
		}
		return "", true, fmt.Errorf("test-select-stack-child '%s' not found among stacks", testChild)
	}
	if testIndex >= 0 {
		if testIndex >= len(stacks) {
			return "", true, fmt.Errorf("test-select-stack-index %d out of range (have %d stacks)", testIndex, len(stacks))
		}
		b, errPick := PickBranchFromStack(stacks[testIndex], purpose)
		return b, true, errPick
	}
	return "", true, nil
}

// BuildStackSelectionOptions builds human-readable options for interactive selection along with stacks.
func BuildStackSelectionOptions(baseBranch string, purpose NavigationPurpose) (options []string, stacks [][]string, err error) {
	stacks, err = git.GetAvailableStacksFromBase(baseBranch)
	if err != nil {
		return nil, nil, err
	}
	options = make([]string, len(stacks))
	for i, s := range stacks {
		if len(s) > 1 {
			count := len(s) - 1
			switch purpose {
			case PurposeTop:
				topBranch := s[len(s)-1]
				options[i] = fmt.Sprintf("%s (top of stack with %d branches)", topBranch, count)
			default:
				options[i] = fmt.Sprintf("%s (stack with %d branches)", s[1], count)
			}
		} else {
			options[i] = fmt.Sprintf("Stack %d", i+1)
		}
	}
	return options, stacks, nil
}
