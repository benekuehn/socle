package cmd

import (
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Using trackBranch and writeFile helpers defined in other _test.go files

func TestUpCommand(t *testing.T) {
	t.Run("Checkout child branch from middle branch", func(t *testing.T) {
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

		// Start on middle branch (feat-a)
		testutils.RunCommand(t, repoPath, "git", "checkout", "feat-a")

		// Action
		_, stderr, err := runSoCommandWithOutput(t, "up")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "feat-b", currentBranch)
	})

	t.Run("Checkout child branch from base branch", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup stack: main -> feat-a
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feat-a")
		writeFile(t, repoPath, "a.txt", "a")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "a")
		trackBranch(t, repoPath, "feat-a", "main", "main")

		// Start on base branch (main)
		testutils.RunCommand(t, repoPath, "git", "checkout", "main")

		// Action
		_, stderr, err := runSoCommandWithOutput(t, "up")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "feat-a", currentBranch)
	})

	t.Run("Already on top branch", func(t *testing.T) {
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
		testutils.RunCommand(t, repoPath, "git", "checkout", "feat-b")

		// Action
		stdout, stderr, err := runSoCommandWithOutput(t, "up")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		assert.Contains(t, stdout, "Already on the top branch: 'feat-b'.")
		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "feat-b", currentBranch)
	})

	t.Run("On untracked branch", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup untracked branch
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "untracked-feat")

		// Action
		stdout, stderr, err := runSoCommandWithOutput(t, "up")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		assert.Contains(t, stdout, "Error getting stack info: current branch 'untracked-feat' is not tracked by socle (missing socle-base config) and is not a known base branch.")
		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "untracked-feat", currentBranch)
	})
}
