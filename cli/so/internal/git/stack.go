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
// - FullStack is a linear ordered slice from BaseBranch to tip when the base has <=1 child lineage.
// - FullStack is set to nil ONLY when the currently checked out branch is a known base branch (main/master/develop)
//   that has >1 tracked child branches, i.e. multiple independent stacks originate from it.
//   In that case CurrentStack will contain just the base branch and navigation commands should prompt.
// - When the current branch is NOT the base but the base has multiple child stacks, FullStack is still nil.
//   CurrentStack then represents the lineage from the base to the current branch and navigation commands must
//   treat it as the active linear stack without prompting for stack selection.
// The navigation runners (up/top/bottom) implement this distinction; log command also follows these rules.

// GetStackInfo retrieves comprehensive information about the current branch stack.
// It returns all stack-related information in a single StackInfo struct.
func GetStackInfo() (*StackInfo, error) {
	// 1. Get Current Branch
	currentBranch, err := GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	// 2. Get all parent relationships at once
	parentMap, err := GetAllSocleParents()
	if err != nil {
		return nil, fmt.Errorf("failed to read tracking relationships: %w", err)
	}

	// Build the child map for later operations
	childMap := BuildChildMap(parentMap)

	// 3. Check if we are actually on a known base branch
	knownBases := map[string]bool{"main": true, "master": true, "develop": true}
	var baseBranch string
	var currentStack []string

	if knownBases[currentBranch] {
		baseBranch = currentBranch
		currentStack = []string{baseBranch} // Stack is just the base itself
	} else {
		// 4. Check if current branch is tracked
		baseConfigKey := fmt.Sprintf("branch.%s.socle-base", currentBranch)
		baseBranchNameFromConfig, errBase := GetGitConfig(baseConfigKey)
		isBaseNotFound := errors.Is(errBase, ErrConfigNotFound)

		if isBaseNotFound {
			return nil, fmt.Errorf("current branch '%s' is not tracked by socle (missing socle-base config) and is not a known base branch.\nRun 'so track' on this branch first", currentBranch)
		}
		if errBase != nil {
			return nil, fmt.Errorf("failed to read tracking base for '%s': %w", currentBranch, errBase)
		}

		baseBranch = baseBranchNameFromConfig

		// 5. Build the stack by walking up the parents using the parentMap
		currentStack = []string{currentBranch}
		currentInLoop := currentBranch

		for currentInLoop != baseBranch {
			parent, hasParent := parentMap[currentInLoop]
			if !hasParent {
				return nil, fmt.Errorf("tracking information broken: parent not found for branch '%s', which is not the base '%s'. Cannot determine stack", currentInLoop, baseBranch)
			}

			// Prepend the found parent to the stack slice
			currentStack = append([]string{parent}, currentStack...)

			// Check if the parent we just added is the base branch
			if parent == baseBranch {
				break // We've reached the base, stack is complete
			}

			// Move up for the next iteration
			currentInLoop = parent

			// Safety break to prevent infinite loops in case of cyclic metadata
			if len(currentStack) > 50 { // Arbitrary limit
				return nil, fmt.Errorf("stack trace exceeds 50 levels, assuming cycle or error in tracking metadata")
			}
		}
	}

	// 6. Determine the full ordered stack
	slog.Debug("Determining full ordered stack...")

	// Reconstruct the lineage from the base upwards
	fullStack := []string{baseBranch}
	current := baseBranch
	visited := make(map[string]bool)
	visited[current] = true

	for {
		children, found := childMap[current]
		if !found || len(children) == 0 {
			break // No more tracked children in this lineage
		}

		if len(children) > 1 {
			if knownBases[current] {
				// Base branch with multiple stacks.
				// If we are CURRENTLY on the base branch itself, we cannot provide a single linear FullStack.
				// If we are NOT on the base (i.e., navigating inside one lineage), we can still produce a FullStack
				// by using currentStack (base->...->currentBranch) and then extending downward from currentBranch.
				if currentBranch == current { // we are ON the base branch
					slog.Debug("Base branch with multiple stacks detected (on base)", "base", current, "children", children)
					return &StackInfo{
						CurrentBranch: currentBranch,
						BaseBranch:    baseBranch,
						CurrentStack:  currentStack,
						FullStack:     nil, // Signal that multiple stacks exist from base context
						ParentMap:     parentMap,
						ChildMap:      childMap,
					}, nil
				}
				// We are inside lineage: Build full stack using known path to currentBranch then descend.
				slog.Debug("Inside multi-stack lineage; reconstructing full lineage for current branch", "currentBranch", currentBranch)
				fullStack = append([]string{}, currentStack...) // base->...->currentBranch
				// Descend from currentBranch to tip
				walker := currentBranch
				for {
					childList, foundChild := childMap[walker]
					if !foundChild || len(childList) == 0 {
						break
					}
					if len(childList) > 1 {
						return nil, fmt.Errorf("non-base branch '%s' has multiple children %v, violating linear lineage assumption", walker, childList)
					}
					next := childList[0]
					// Avoid duplicates if currentBranch already included
					if next == walker {
						return nil, fmt.Errorf("cycle detected at branch '%s' while descending lineage", walker)
					}
					fullStack = append(fullStack, next)
					walker = next
					if len(fullStack) > 100 { // safety
						return nil, fmt.Errorf("stack reconstruction exceeded 100 branches (descending)")
					}
				}
				// Finished lineage reconstruction.
				break
			} else {
				// Non-base branch with multiple children - violates linear stack assumption
				return nil, fmt.Errorf("non-base branch '%s' has multiple children %v, which violates linear stack structure. Only base branches (%v) can have multiple children", current, children, []string{"main", "master", "develop"})
			}
		}
		nextChild := children[0]

		if visited[nextChild] {
			return nil, fmt.Errorf("cycle detected in stack tracking near branch '%s'", nextChild)
		}

		fullStack = append(fullStack, nextChild)
		visited[nextChild] = true
		current = nextChild

		if len(fullStack) > 100 { // Safety break
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

// GetAvailableStacksFromBase returns all available stacks that start from the given base branch
func GetAvailableStacksFromBase(baseBranch string) ([][]string, error) {
	parentMap, err := GetAllSocleParents()
	if err != nil {
		return nil, fmt.Errorf("failed to read tracking relationships: %w", err)
	}

	childMap := BuildChildMap(parentMap)
	children, found := childMap[baseBranch]
	if !found || len(children) == 0 {
		return nil, fmt.Errorf("no stacks found starting from base branch '%s'", baseBranch)
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
