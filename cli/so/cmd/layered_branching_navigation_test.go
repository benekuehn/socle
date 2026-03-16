package cmd

import (
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLayeredBranchingNavigation(t *testing.T) {
	repoPath, cleanup := setupRepoWithLayeredBranching(t)
	defer cleanup()

	testutils.RunCommand(t, repoPath, "git", "checkout", "feature-a")

	stdout, stderr, err := runSoCommandWithOutput(t, "up", "--test-select-stack-child=feature-c")
	require.NoError(t, err)
	assert.Empty(t, stderr)
	assert.NotContains(t, stdout, "Multiple stacks available")
	cur, err := git.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "feature-c", cur)

	testutils.RunCommand(t, repoPath, "git", "checkout", "feature-a")
	stdout, stderr, err = runSoCommandWithOutput(t, "top", "--test-select-stack-child=feature-b")
	require.NoError(t, err)
	assert.Empty(t, stderr)
	cur, err = git.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "feature-b", cur)

	testutils.RunCommand(t, repoPath, "git", "checkout", "feature-a")
	stdout, stderr, err = runSoCommandWithOutput(t, "bottom", "--test-select-stack-child=feature-c")
	require.NoError(t, err)
	assert.Empty(t, stderr)
	cur, err = git.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "feature-c", cur)
}
