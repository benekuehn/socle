package cmd

import (
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Using trackBranch and writeFile helpers defined in other _test.go files

func TestDownCommand(t *testing.T) {
	t.Run("Checkout parent branch from middle branch", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup stack: main -> feat-a -> feat-b
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feat-a")
		writeFile(t, repoPath, "a.txt", "a")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "a")
		trackBranch(t, repoPath, "feat-a", "main", "main")

		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feat-b")
		writeFile(t, repoPath, "b.txt", "b")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "b")
		trackBranch(t, repoPath, "feat-b", "feat-a", "main")

		// Start on top branch (feat-b)

		// Action
		stdout, stderr, err := runSoCommandWithOutput(t, "down")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		assert.Contains(t, stdout, "Checked out parent branch: 'feat-a'")
		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "feat-a", currentBranch)
	})

	t.Run("Checkout base branch from bottom branch", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup stack: main -> feat-a
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feat-a")
		writeFile(t, repoPath, "a.txt", "a")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "a")
		trackBranch(t, repoPath, "feat-a", "main", "main")

		// Start on bottom branch (feat-a)

		// Action
		stdout, stderr, err := runSoCommandWithOutput(t, "down")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		assert.Contains(t, stdout, "Checked out parent branch: 'main'")
		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "main", currentBranch)
	})

	t.Run("Already on base branch", func(t *testing.T) {
		_ /* repoPath */, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Start on main

		// Action
		stdout, stderr, err := runSoCommandWithOutput(t, "down")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		assert.Contains(t, stdout, "Already on the base branch 'main'. Cannot go down further.")
		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "main", currentBranch)
	})

	t.Run("On untracked branch", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup untracked branch
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "untracked-feat")

		// Action
		stdout, _ /* stderr */, err := runSoCommandWithOutput(t, "down")

		// Assertions
		require.Error(t, err)
		// Expect the specific error from GetCurrentStackInfo
		assert.Contains(t, err.Error(), "not tracked by socle")
		assert.Empty(t, stdout)
		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "untracked-feat", currentBranch)
	})
}
