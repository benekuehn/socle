package cmd

import (
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Using trackBranch helper defined in top_test.go (assuming same package)

func TestBottomCommand(t *testing.T) {
	t.Run("Checkout bottom branch from top branch", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup stack: main -> feat-a -> feat-b
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feat-a")
		writeFile(t, repoPath, "a.txt", "a")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "feat: a")
		trackBranch(t, repoPath, "feat-a", "main", "main")

		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feat-b")
		writeFile(t, repoPath, "b.txt", "b")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "feat: b")
		trackBranch(t, repoPath, "feat-b", "feat-a", "main")

		// Start on top branch (feat-b)

		// Action
		_, stderr, err := runSoCommandWithOutput(t, "bottom")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "feat-a", currentBranch)
	})

	t.Run("Checkout bottom branch from middle branch", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup stack: main -> feat-a -> feat-b -> feat-c
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feat-a") // on main
		writeFile(t, repoPath, "a.txt", "a")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "a")
		trackBranch(t, repoPath, "feat-a", "main", "main")

		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feat-b") // on feat-a
		writeFile(t, repoPath, "b.txt", "b")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "b")
		trackBranch(t, repoPath, "feat-b", "feat-a", "main")

		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feat-c") // on feat-b
		writeFile(t, repoPath, "c.txt", "c")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "c")
		trackBranch(t, repoPath, "feat-c", "feat-b", "main")

		// Start on middle branch (feat-b)
		testutils.RunCommand(t, repoPath, "git", "checkout", "feat-b")

		// Action
		_, stderr, err := runSoCommandWithOutput(t, "bottom")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "feat-a", currentBranch)
	})

	t.Run("Checkout bottom branch from base branch", func(t *testing.T) {
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

		// Start on base branch
		testutils.RunCommand(t, repoPath, "git", "checkout", "main")

		// Action
		_, stderr, err := runSoCommandWithOutput(t, "bottom")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "feat-a", currentBranch)
	})

	t.Run("Already on bottom branch", func(t *testing.T) {
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

		// Start on bottom branch (feat-a)
		testutils.RunCommand(t, repoPath, "git", "checkout", "feat-a")

		// Action
		stdout, stderr, err := runSoCommandWithOutput(t, "bottom")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		assert.Contains(t, stdout, "Already on the bottom branch: 'feat-a'")
		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "feat-a", currentBranch)
	})

	t.Run("On base branch, no stack exists", func(t *testing.T) {
		_ /* repoPath */, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Start on main, no other branches

		// Action
		stdout, stderr, err := runSoCommandWithOutput(t, "bottom")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		assert.Contains(t, stdout, "Already on the base branch 'main', which is the only branch in the stack.")
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
		stdout, stderr, err := runSoCommandWithOutput(t, "bottom")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		assert.Contains(t, stdout, "Error getting stack info: current branch 'untracked-feat' is not tracked by socle (missing socle-base config) and is not a known base branch.")
		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "untracked-feat", currentBranch)
	})
}
