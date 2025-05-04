package git

import (
	"fmt"
	"log/slog"
	"strings"
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

	_, errParent := GetGitConfig(parentConfigKey)
	baseBranch, errBase := GetGitConfig(baseConfigKey)

	// Check for specific "not found" errors from GetGitConfig
	// (Assuming GetGitConfig returns an error containing "not found" text)
	isParentNotFound := errParent != nil && strings.Contains(errParent.Error(), "not found")
	isBaseNotFound := errBase != nil && strings.Contains(errBase.Error(), "not found")

	isUntracked := isParentNotFound || isBaseNotFound

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

	// If it's not a known base branch AND tracking info is missing
	if isUntracked {
		err = fmt.Errorf("current branch '%s' is not tracked by socle and is not a known base branch.\nRun 'so track' on this branch first", currentBranch)
		return
	}

	// If we reach here, the branch is tracked and is not the base branch itself.
	// BaseBranch variable holds the correct base name from config.

	// 3. Build the stack by walking up the parents
	stack = []string{currentBranch} // Start with the current branch
	current := currentBranch        // Start the walk from the current branch

	for {
		// Get the parent of the 'current' branch in the walk-up
		currentParentKey := fmt.Sprintf("branch.%s.socle-parent", current)
		parent, parentErr := GetGitConfig(currentParentKey)

		if parentErr != nil {
			// If we can't find the parent config for an intermediate branch, the tracking is broken
			if strings.Contains(parentErr.Error(), "not found") {
				err = fmt.Errorf("tracking information broken: parent branch config key '%s' not found for branch '%s'. Cannot determine stack", currentParentKey, current)
			} else {
				err = fmt.Errorf("failed to read tracking parent for intermediate branch '%s': %w", current, parentErr)
			}
			return // Return empty stack/base and the error
		}

		// Prepend the found parent to the stack slice
		stack = append([]string{parent}, stack...)

		// Check if the parent we just added is the base branch
		if parent == baseBranch {
			break // We've reached the base, stack is complete
		}

		// Move up for the next iteration
		current = parent

		// Safety break to prevent infinite loops in case of cyclic metadata
		if len(stack) > 50 { // Arbitrary limit
			err = fmt.Errorf("stack trace exceeds 50 levels, assuming cycle or error in tracking metadata")
			return // Return empty stack/base and the error
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
