package git

import (
	"errors"
	"fmt"
	"log/slog"
)

// GetCurrentStackInfo determines the current branch, its base branch, and the full stack of branches
// by walking up the socle parent tracking information stored in Git config.
// It returns the current branch name, a slice of branch names from base to current (inclusive),
// the determined base branch name, and any error encountered.
func GetCurrentStackInfo() (currentBranch string, stack []string, baseBranch string, err error) {
	// 1. Get Current Branch
	currentBranch, err = GetCurrentBranch()
	if err != nil {
		err = fmt.Errorf("failed to get current branch: %w", err)
		return // Return zero values for others
	}

	// 2. Check if current branch is tracked and get its base
	parentConfigKey := fmt.Sprintf("branch.%s.socle-parent", currentBranch)
	baseConfigKey := fmt.Sprintf("branch.%s.socle-base", currentBranch)

	// errParent is used to check if parent config exists, actual parent name is read in the loop if needed.
	_, errParent := GetGitConfig(parentConfigKey)
	baseBranchNameFromConfig, errBase := GetGitConfig(baseConfigKey)

	// Check for specific "not found" errors from GetGitConfig
	isParentNotFound := errors.Is(errParent, ErrConfigNotFound)
	isBaseNotFound := errors.Is(errBase, ErrConfigNotFound)

	// isUntracked := isParentNotFound || isBaseNotFound // This variable is no longer used with the refined logic below.

	// Handle other unexpected errors during config reading
	if errParent != nil && !isParentNotFound {
		err = fmt.Errorf("failed to read tracking parent for '%s': %w", currentBranch, errParent)
		return
	}
	if errBase != nil && !isBaseNotFound {
		err = fmt.Errorf("failed to read tracking base for '%s': %w", currentBranch, errBase)
		return
	}

	// Check if we are actually on a known base branch
	// TODO: Make base branches configurable instead of hardcoded map
	knownBases := map[string]bool{"main": true, "master": true, "develop": true}
	if knownBases[currentBranch] {
		baseBranch = currentBranch
		stack = []string{baseBranch} // Stack is just the base itself
		err = nil                    // Not an error state
		return
	}

	// If it's not a known base branch AND tracking info is missing (specifically base, parent might be missing if it IS the base)
	if isBaseNotFound { // If socle-base is not defined, it must be untracked or a base branch itself (which was checked above)
		err = fmt.Errorf("current branch '%s' is not tracked by socle (missing socle-base config) and is not a known base branch.\nRun 'so track' on this branch first", currentBranch)
		return
	}
	// If base is found, use it. isUntracked check earlier was a bit broad.
	baseBranch = baseBranchNameFromConfig

	// If we reach here, the branch is tracked (has a socle-base) and is not a known base branch itself.
	// baseBranch variable holds the correct base name from config.

	// 3. Build the stack by walking up the parents
	stack = []string{currentBranch} // Start with the current branch
	currentInLoop := currentBranch  // Start the walk from the current branch

	for {
		// If currentInLoop is already the baseBranch, we stop. This happens if currentBranch's parent is the base.
		if currentInLoop == baseBranch {
			break
		}

		// Get the parent of the 'currentInLoop' branch in the walk-up
		currentParentKey := fmt.Sprintf("branch.%s.socle-parent", currentInLoop)
		parent, parentErr := GetGitConfig(currentParentKey)

		if parentErr != nil {
			// If we can't find the parent config for an intermediate branch (that is not the baseBranch itself),
			// the tracking is broken or this branch is unexpectedly the one just above base.
			if errors.Is(parentErr, ErrConfigNotFound) {
				// This implies currentInLoop is the branch directly on top of baseBranch, and baseBranch has no socle-parent itself.
				// Or, tracking is broken for currentInLoop.
				// If parent is indeed the baseBranch, the loop condition `currentInLoop == baseBranch` should handle it.
				// If we expect a parent because currentInLoop is not baseBranch, then this is an error.
				err = fmt.Errorf("tracking information broken: parent branch config key '%s' not found for branch '%s', which is not the base '%s'. Cannot determine stack", currentParentKey, currentInLoop, baseBranch)
			} else {
				err = fmt.Errorf("failed to read tracking parent for intermediate branch '%s': %w", currentInLoop, parentErr)
			}
			return // Return with error
		}

		// Prepend the found parent to the stack slice
		stack = append([]string{parent}, stack...)

		// Check if the parent we just added is the base branch
		if parent == baseBranch {
			break // We've reached the base, stack is complete
		}

		// Move up for the next iteration
		currentInLoop = parent

		// Safety break to prevent infinite loops in case of cyclic metadata
		if len(stack) > 50 { // Arbitrary limit
			err = fmt.Errorf("stack trace exceeds 50 levels, assuming cycle or error in tracking metadata")
			return // Return with error
		}
	} // End of for loop

	// Stack is now built correctly from base to currentBranch
	return currentBranch, stack, baseBranch, nil // Success
}

// GetFullStack determines the complete stack of branches related to the current stack,
// ordered from the base branch up to the highest tracked descendant branch.
// It takes the current stack (from base to the currently checked-out branch) as input.
// It returns the full ordered stack (base -> ... -> top) and the map of all parent relationships.
func GetFullStack(currentStack []string) (orderedStack []string, allParents map[string]string, err error) {
	slog.Debug("Determining full ordered stack...")
	allParents, err = GetAllSocleParents()
	if err != nil {
		err = fmt.Errorf("failed to read all tracking relationships: %w", err)
		return
	}

	if len(currentStack) == 0 {
		slog.Debug("Current stack is empty.")
		err = fmt.Errorf("internal error: GetFullStack called with empty currentStack")
		return
	}
	baseBranch := currentStack[0]

	// Reconstruct the lineage from the base upwards
	childMap := BuildChildMap(allParents)
	orderedStack = []string{baseBranch}
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
			err = fmt.Errorf("cycle detected in stack tracking near branch '%s'", nextChild)
			return // Return partial stack and error
		}

		orderedStack = append(orderedStack, nextChild)
		visited[nextChild] = true
		current = nextChild

		if len(orderedStack) > 100 { // Safety break
			err = fmt.Errorf("stack reconstruction exceeded 100 branches, aborting")
			return // Return partial stack and error
		}
	}

	slog.Debug("Full ordered stack identified:", "stack", orderedStack)
	return orderedStack, allParents, nil
}
