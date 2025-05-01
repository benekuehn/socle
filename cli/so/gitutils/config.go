package gitutils

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// GetGitConfig retrieves a specific git config key's value.
// Returns an error containing "exit status 1" if the key doesn't exist.
func GetGitConfig(key string) (string, error) {
	// Assumes RunGitCommand exists and returns error wrapping *exec.ExitError on failure
	output, err := RunGitCommand("config", "--get", key)
	if err == nil {
		return output, nil // Success
	}

	// Check if the underlying error is ExitError code 1
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		// Key not found: Return our specific sentinel error, WRAPPING it.
		return "", fmt.Errorf("%w: %s", ErrConfigNotFound, key) // <-- Use %w
	}

	// Any other error during the git command
	return "", fmt.Errorf("failed to get git config '%s': %w", key, err) // <-- Use %w here too
}

// SetGitConfig sets (or adds) a git config key-value pair.
// Uses --add to avoid deleting other values if the key somehow exists multiple times,
// though for our usage, a simple set would likely be fine too.
func SetGitConfig(key, value string) error {
	// Using --local ensures we write to .git/config, not global or system
	_, err := RunGitCommand("config", "--local", "--add", key, value)
	return err
}

// UnsetGitConfig removes a git config key.
// Useful for cleanup or an 'untrack' command.
func UnsetGitConfig(key string) error {
	// Use --unset-all in case --add resulted in multiple entries (unlikely for us)
	_, err := RunGitCommand("config", "--local", "--unset-all", key)
	// Ignore exit code 5 which means the section or key was not found
	if err != nil && !strings.Contains(err.Error(), "exit status 5") {
		return err
	}
	return nil
}

func IsRerereEnabled() (bool, error) {
	output, err := RunGitCommand("config", "--get", "rerere.enabled")

	if err == nil {
		// Config key exists, check its value
		return strings.ToLower(strings.TrimSpace(output)) == "true", nil
	}

	// Check if the error was simply "key not found"
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		// Key not found means rerere is not explicitly enabled (or disabled)
		// via this specific key, treat as disabled.
		return false, nil
	}

	// Any other error is unexpected
	return false, fmt.Errorf("failed to check git config rerere.enabled: %w", err)
}

var ErrConfigNotFound = errors.New("git config key not found")

// GetAllSocleParents returns a map of childBranch -> parentBranch based on socle config.
func GetAllSocleParents() (map[string]string, error) {
	// Get all config keys matching the pattern using --get-regexp
	// Note: Output format is "key value\nkey value\n..."
	output, err := RunGitCommand("config", "--local", "--get-regexp", `^branch\.(.+)\.socle-parent$`)
	if err != nil {
		// It's okay if no keys are found (exit code 1 from --get-regexp)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return make(map[string]string), nil // No keys found, return empty map
		}
		// Other unexpected error
		return nil, fmt.Errorf("failed to get socle parent configs: %w", err)
	}

	parentMap := make(map[string]string)
	lines := strings.Split(output, "\n")
	// Regex to extract the branch name from the key
	keyRegex := regexp.MustCompile(`^branch\.(.+)\.socle-parent$`)

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2) // Split key and value
		if len(parts) != 2 {
			continue
		} // Skip malformed lines

		key := parts[0]
		value := parts[1]

		matches := keyRegex.FindStringSubmatch(key)
		if len(matches) == 2 {
			childBranch := matches[1]
			parentMap[childBranch] = value
		}
	}
	return parentMap, nil
}

// BuildChildMap creates a map of parent -> list of children.
func BuildChildMap(parentMap map[string]string) map[string][]string {
	childMap := make(map[string][]string)
	for child, parent := range parentMap {
		childMap[parent] = append(childMap[parent], child)
	}
	return childMap
}

// FindAllDescendants performs a DFS to find all descendants of a start node.
func FindAllDescendants(startNode string, childMap map[string][]string) []string {
	var descendants []string
	var visited = make(map[string]bool)
	var stack = []string{startNode} // Use slice as stack

	// Keep track of nodes added to descendants to avoid duplicates if graph has cycles (shouldn't happen)
	addedToDescendants := make(map[string]bool)

	for len(stack) > 0 {
		// Pop
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[node] {
			continue
		}
		visited[node] = true

		// Process children
		if children, ok := childMap[node]; ok {
			// Add children to stack (reverse order to process approx left-to-right in DFS)
			for i := len(children) - 1; i >= 0; i-- {
				child := children[i]
				if !visited[child] { // Add to stack only if not visited
					stack = append(stack, child)
				}
				// Add child to descendants list if not already added
				if !addedToDescendants[child] {
					descendants = append(descendants, child)
					addedToDescendants[child] = true
				}
			}
		}
	}
	// The result might not be perfectly ordered topologically if branches cross,
	// but it contains all descendants. Sorting might be needed depending on use case.
	// For submit, the order we process doesn't strictly matter as much as having the full list.
	return descendants
}
