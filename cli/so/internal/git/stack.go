package git

import (
	"errors"
	"fmt"
	"log/slog"
)

// StackInfo holds all information about a branch stack
type StackInfo struct {
	// The currently checked out branch name
	CurrentBranch string
	// The base branch of the stack
	BaseBranch string
	// Branches from base to current, inclusive
	CurrentStack []string
	// All branches from base to tip, inclusive
	FullStack []string
	// Map of child branch -> parent branch
	ParentMap map[string]string
	// Map of parent branch -> child branches
	ChildMap map[string][]string
}

// Invariants / Semantics:
// - FullStack is a linear ordered slice from BaseBranch to tip for the active lineage.
// - FullStack is nil when CurrentBranch itself has multiple tracked children, because there is no
//   single implicit "up/top/bottom" target without stack selection.
// - CurrentStack always contains the lineage from BaseBranch to CurrentBranch.

// GetStackInfo retrieves comprehensive information about the current branch stack.
// It returns all stack-related information in a single StackInfo struct.
func GetStackInfo() (*StackInfo, error) {
	currentBranch, err := GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	parentMap, err := GetAllSocleParents()
	if err != nil {
		return nil, fmt.Errorf("failed to read tracking relationships: %w", err)
	}
	childMap := BuildChildMap(parentMap)

	knownBases := map[string]bool{"main": true, "master": true, "develop": true}
	baseBranch := ""

	if knownBases[currentBranch] {
		baseBranch = currentBranch
	} else {
		baseConfigKey := fmt.Sprintf("branch.%s.socle-base", currentBranch)
		baseBranch, err = GetGitConfig(baseConfigKey)
		if errors.Is(err, ErrConfigNotFound) {
			return nil, fmt.Errorf("current branch '%s' is not tracked by socle (missing socle-base config) and is not a known base branch.\nRun 'so track' on this branch first", currentBranch)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tracking base for '%s': %w", currentBranch, err)
		}
	}

	currentStack := []string{currentBranch}
	for walker := currentBranch; walker != baseBranch; {
		parent, ok := parentMap[walker]
		if !ok {
			return nil, fmt.Errorf("tracking information broken: parent not found for branch '%s', which is not the base '%s'. Cannot determine stack", walker, baseBranch)
		}
		currentStack = append([]string{parent}, currentStack...)
		walker = parent
		if len(currentStack) > 100 {
			return nil, fmt.Errorf("stack trace exceeds 100 levels, assuming cycle or error in tracking metadata")
		}
	}

	if children := childMap[currentBranch]; len(children) > 1 {
		slog.Debug("Current branch has multiple children; full stack is ambiguous", "current", currentBranch, "children", children)
		return &StackInfo{
			CurrentBranch: currentBranch,
			BaseBranch:    baseBranch,
			CurrentStack:  currentStack,
			FullStack:     nil,
			ParentMap:     parentMap,
			ChildMap:      childMap,
		}, nil
	}

	fullStack := append([]string{}, currentStack...)
	for {
		last := fullStack[len(fullStack)-1]
		children := childMap[last]
		if len(children) == 0 {
			break
		}
		if len(children) > 1 {
			return nil, fmt.Errorf("branch '%s' has multiple children %v; checkout one child branch to continue", last, children)
		}
		next := children[0]
		fullStack = append(fullStack, next)
		if len(fullStack) > 100 {
			return nil, fmt.Errorf("stack reconstruction exceeded 100 branches, aborting")
		}
	}

	slog.Debug("Full ordered stack identified:", "stack", fullStack)
	return &StackInfo{
		CurrentBranch: currentBranch,
		BaseBranch:    baseBranch,
		CurrentStack:  currentStack,
		FullStack:     fullStack,
		ParentMap:     parentMap,
		ChildMap:      childMap,
	}, nil
}

// GetAvailableStacksFromBase returns all available stacks that start from the given branch.
func GetAvailableStacksFromBase(baseBranch string) ([][]string, error) {
	parentMap, err := GetAllSocleParents()
	if err != nil {
		return nil, fmt.Errorf("failed to read tracking relationships: %w", err)
	}

	childMap := BuildChildMap(parentMap)
	children, found := childMap[baseBranch]
	if !found || len(children) == 0 {
		return nil, fmt.Errorf("no stacks found starting from branch '%s'", baseBranch)
	}

	var stacks [][]string
	for _, child := range children {
		stack, err := buildLinearStackFromChild(baseBranch, child, childMap, make(map[string]bool))
		if err != nil {
			slog.Warn("Failed to build stack from child", "base", baseBranch, "child", child, "error", err)
			continue
		}
		stacks = append(stacks, stack)
	}

	return stacks, nil
}

// buildLinearStackFromChild builds a complete linear stack starting from base->child
func buildLinearStackFromChild(baseBranch, child string, childMap map[string][]string, visited map[string]bool) ([]string, error) {
	if visited[child] {
		return nil, fmt.Errorf("cycle detected in stack near branch '%s'", child)
	}

	stack := []string{baseBranch, child}
	current := child
	visited[child] = true

	for {
		children, found := childMap[current]
		if !found || len(children) == 0 {
			break // End of stack
		}

		if len(children) > 1 {
			return nil, fmt.Errorf("non-base branch '%s' has multiple children %v, violating linear stack structure", current, children)
		}

		nextChild := children[0]
		if visited[nextChild] {
			return nil, fmt.Errorf("cycle detected in stack near branch '%s'", nextChild)
		}

		stack = append(stack, nextChild)
		visited[nextChild] = true
		current = nextChild

		if len(stack) > 100 { // Safety break
			return nil, fmt.Errorf("stack reconstruction exceeded 100 branches")
		}
	}

	return stack, nil
}

// IsKnownBaseBranch checks if a branch is a known base branch
func IsKnownBaseBranch(branchName string) bool {
	knownBases := map[string]bool{"main": true, "master": true, "develop": true}
	return knownBases[branchName]
}
