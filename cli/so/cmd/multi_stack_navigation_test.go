package cmd

import (
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Multi-stack navigation scenarios:
// Repository with two stacks from main:
//   Stack 0: main -> feature-a -> feature-b
//   Stack 1: main -> feature-x -> feature-y
// When on a non-base branch (e.g., feature-a, feature-b, feature-x, feature-y), commands should
// navigate linearly without emitting the prompt string "Multiple stacks available".
// When on base branch (main) we exercise hidden test flag to auto-select a stack instead of interactive prompt.
func TestMultiStackNavigation(t *testing.T) {
	repoPath, cleanup := setupRepoWithMultipleStacks(t)
	defer cleanup()

	// End state after setup is last created branch feature-y. Verify.
	cur, err := git.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "feature-y", cur)

	// Scenario 1: bottom inside second stack (feature-y -> feature-x).
	stdout, stderr, err := runSoCommandWithOutput(t, "down")
	require.NoError(t, err)
	assert.Empty(t, stderr)
	assert.NotContains(t, stdout, "Multiple stacks available")
	cur, err = git.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "feature-x", cur)

	// Scenario 2: down again to base (feature-x -> main)
	stdout, stderr, err = runSoCommandWithOutput(t, "down")
	require.NoError(t, err)
	assert.Empty(t, stderr)
	assert.NotContains(t, stdout, "Multiple stacks available") // Down never prompts
	cur, err = git.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "main", cur)

	// Helper to find index for a stack whose first child matches given branch

	// Scenario 3: Up from base selecting stack whose first child is feature-a.
	stdout, stderr, err = runSoCommandWithOutput(t, "up", "--test-select-stack-child=feature-a")
	require.NoError(t, err)
	assert.Empty(t, stderr)
	// Should not show prompt because auto selection bypassed
	assert.NotContains(t, stdout, "Multiple stacks available")
	cur, err = git.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "feature-a", cur)

	// Scenario 4: Up inside stack (feature-a -> feature-b) no prompt
	stdout, stderr, err = runSoCommandWithOutput(t, "up")
	require.NoError(t, err)
	assert.Empty(t, stderr)
	assert.NotContains(t, stdout, "Multiple stacks available")
	cur, err = git.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "feature-b", cur)

	// Scenario 5: top from feature-a (after moving back) selects tip without prompt
	// Move back to feature-a first
	testutils.RunCommand(t, repoPath, "git", "checkout", "feature-a")
	stdout, stderr, err = runSoCommandWithOutput(t, "top")
	require.NoError(t, err)
	assert.Empty(t, stderr)
	assert.NotContains(t, stdout, "Multiple stacks available")
	cur, err = git.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "feature-b", cur)

	// Scenario 6: bottom from feature-b moves to feature-a without prompt
	stdout, stderr, err = runSoCommandWithOutput(t, "bottom")
	require.NoError(t, err)
	assert.Empty(t, stderr)
	assert.NotContains(t, stdout, "Multiple stacks available")
	cur, err = git.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "feature-a", cur)

	// Scenario 7: top from base selecting stack whose first child is feature-x -> feature-y tip
	testutils.RunCommand(t, repoPath, "git", "checkout", "main")
	stdout, stderr, err = runSoCommandWithOutput(t, "top", "--test-select-stack-child=feature-x")
	require.NoError(t, err)
	assert.Empty(t, stderr)
	assert.NotContains(t, stdout, "Multiple stacks available")
	cur, err = git.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "feature-y", cur)

	// Scenario 8: bottom from base selecting stack whose first child is feature-a
	testutils.RunCommand(t, repoPath, "git", "checkout", "main")
	stdout, stderr, err = runSoCommandWithOutput(t, "bottom", "--test-select-stack-child=feature-a")
	require.NoError(t, err)
	assert.Empty(t, stderr)
	assert.NotContains(t, stdout, "Multiple stacks available")
	cur, err = git.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "feature-a", cur)
}
