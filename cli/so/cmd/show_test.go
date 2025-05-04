// cli/so/cmd/show_test.go
package cmd

import (
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShowCommand(t *testing.T) {
	t.Run("Show on base branch", func(t *testing.T) {
		_, cleanup := testutils.SetupGitRepo(t) // Only need main branch
		defer cleanup()

		// Action: Run 'so show' while on main
		stdout, stderr, err := runSoCommandWithOutput(t, "show")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr, "Stderr should be empty")
		assert.Contains(t, stdout, "Currently on the base branch 'main'.")
		assert.NotContains(t, stdout, "Current Stack Status:", "Should not print stack table")
	})

	t.Run("Show on untracked branch", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup: create branch but don't track
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature-untracked")

		// Action: Run 'so show'
		stdout, stderr, err := runSoCommandWithOutput(t, "show")

		// Assertions
		require.NoError(t, err) // Should exit cleanly with info message
		assert.Empty(t, stderr)
		assert.Contains(t, stdout, "Branch 'feature-untracked' is not currently tracked by socle.")
		assert.Contains(t, stdout, "Use 'so track' to associate it")
		assert.NotContains(t, stdout, "Current Stack Status:")
	})

	t.Run("Show simple tracked stack (up-to-date, no PR)", func(t *testing.T) {
		_, cleanup := setupRepoWithStack(t, []string{"main", "feature-a", "feature-b"})
		defer cleanup() // cd back happens here

		// Action: Run 'so show' (currently on feature-b from setup)
		stdout, _, err := runSoCommandWithOutput(t, "show")

		// Assertions
		require.NoError(t, err)
		// Expect warnings about GitHub client failing if GITHUB_TOKEN isn't set in test env
		// assert.Empty(t, stderr) // Might not be empty due to GH client warnings

		assert.Contains(t, stdout, "Current branch: feature-b (Stack base: main)")
		assert.Contains(t, stdout, "Current Stack Status:")
		assert.Contains(t, stdout, "main (base)")
		// Use regex or careful Contains because of color codes/variable spacing
		assert.Regexp(t, `->\s+feature-a\s+\(Up-to-date\)\s+\(PR: Not Submitted\)`, stdout)
		assert.Regexp(t, `->\s+feature-b\s+\(Up-to-date\)\s+\(PR: Not Submitted\)\s+\*`, stdout)
	})

	t.Run("Show stack needs restack (no PR)", func(t *testing.T) {
		repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a", "feature-b"})
		defer cleanup()

		// Setup: Add commit to main to make stack need restack
		testutils.RunCommand(t, repoPath, "git", "checkout", "main")
		writeFile(t, repoPath, "main_change.txt", "change")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "change main")
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature-b") // Checkout tip

		// Action: Run 'so show'
		stdout, _, err := runSoCommandWithOutput(t, "show")

		// Assertions
		require.NoError(t, err)
		// Expect warnings about GitHub client failing

		assert.Contains(t, stdout, "Current branch: feature-b (Stack base: main)")
		assert.Contains(t, stdout, "Current Stack Status:")
		assert.Contains(t, stdout, "main (base)")
		assert.Regexp(t, `->\s+feature-a\s+\(Needs Restack\)\s+\(PR: Not Submitted\)`, stdout)
		assert.Regexp(t, `->\s+feature-b\s+\(Needs Restack\)\s+\(PR: Not Submitted\)\s+\*`, stdout) // feature-b also needs restack because its parent does
	})

	t.Run("Show stack with PR config (no API access)", func(t *testing.T) {
		repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a"})
		defer cleanup()

		// Setup: Add PR config for feature-a
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-a.socle-pr-number", "123")

		// Action: Run 'so show'
		stdout, stderr, err := runSoCommandWithOutput(t, "show")

		// Assertions
		require.NoError(t, err)
		// Expect warning about GH client init failure in stderr
		assert.Contains(t, stderr, "Warning: Cannot fetch PR status:")

		assert.Contains(t, stdout, "Current branch: feature-a (Stack base: main)")
		assert.Regexp(t, `->\s+feature-a\s+\(Up-to-date\)\s+\(PR: Login/Setup Needed\)\s+\*`, stdout)
	})

	t.Run("Show stack with invalid PR config", func(t *testing.T) {
		repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a"})
		defer cleanup()

		// Setup: Add invalid PR config for feature-a
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-a.socle-pr-number", "not-a-number")

		// Action: Run 'so show'
		stdout, stderr, err := runSoCommandWithOutput(t, "show")

		// Assertions
		require.NoError(t, err)
		// Expect warning about parsing PR number in stderr
		assert.Contains(t, stderr, "Warning: Could not parse PR number 'not-a-number'")

		assert.Contains(t, stdout, "Current branch: feature-a (Stack base: main)")
		assert.Regexp(t, `->\s+feature-a\s+\(Up-to-date\)\s+\(PR: Invalid #\)\s+\*`, stdout)
	})

	// TODO: Tests with mocked GH client to verify different PR statuses display correctly
}
