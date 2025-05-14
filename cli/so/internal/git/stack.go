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
			slog.Warn("Multiple tracked children found, using the first one for stack order", "parent", current, "children", children)
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
