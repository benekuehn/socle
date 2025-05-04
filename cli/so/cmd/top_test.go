package cmd

import (
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopCommand(t *testing.T) {
	t.Run("Checkout top branch from lower branch", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup stack: main -> feat-a -> feat-b
		// Create feat-a
		testutils.RunCommand(t, repoPath, "git", "checkout", "main")
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feat-a")
		writeFile(t, repoPath, "a.txt", "a")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "feat: a")
		trackBranch(t, repoPath, "feat-a", "main", "main")
		// Create feat-b
		testutils.RunCommand(t, repoPath, "git", "checkout", "feat-a")
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feat-b")
		writeFile(t, repoPath, "b.txt", "b")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "feat: b")
		trackBranch(t, repoPath, "feat-b", "feat-a", "main")

		// Start on a lower branch
		testutils.RunCommand(t, repoPath, "git", "checkout", "feat-a")

		// Action
		stdout, stderr, err := runSoCommandWithOutput(t, "top")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		assert.Contains(t, stdout, "Checked out top branch: 'feat-b'")
		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "feat-b", currentBranch)
	})

	t.Run("Already on top branch", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup stack: main -> feat-a -> feat-b
		// Create feat-a
		testutils.RunCommand(t, repoPath, "git", "checkout", "main")
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feat-a")
		writeFile(t, repoPath, "a.txt", "a")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "feat: a")
		trackBranch(t, repoPath, "feat-a", "main", "main")
		// Create feat-b
		testutils.RunCommand(t, repoPath, "git", "checkout", "feat-a")
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feat-b")
		writeFile(t, repoPath, "b.txt", "b")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "feat: b")
		trackBranch(t, repoPath, "feat-b", "feat-a", "main")

		// Start on the top branch (already on feat-b from setup)

		// Action
		stdout, stderr, err := runSoCommandWithOutput(t, "top")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		assert.Contains(t, stdout, "Already on the top branch: 'feat-b'")
		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "feat-b", currentBranch)
	})

	t.Run("On base branch of a stack", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup stack: main -> feat-a
		testutils.RunCommand(t, repoPath, "git", "checkout", "main")
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feat-a")
		writeFile(t, repoPath, "a.txt", "a")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "feat: a")
		trackBranch(t, repoPath, "feat-a", "main", "main")

		// Start on the base branch
		testutils.RunCommand(t, repoPath, "git", "checkout", "main")

		// Action
		stdout, stderr, err := runSoCommandWithOutput(t, "top")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		assert.Contains(t, stdout, "Currently on the base branch 'main'. Use 'git checkout feat-a'", "Expected info message about being on base branch with suggestion")

		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "main", currentBranch)
	})

	t.Run("On an untracked branch", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup branch but don't track
		testutils.RunCommand(t, repoPath, "git", "checkout", "main")
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "untracked-feat")
		writeFile(t, repoPath, "untracked.txt", "untracked")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "feat: untracked")

		// Action
		stdout, _ /* stderr */, err := runSoCommandWithOutput(t, "top")

		// Assertions
		require.Error(t, err) // Expect an error because GetCurrentStackInfo logic should fail
		assert.Contains(t, err.Error(), "not tracked by socle")
		assert.Empty(t, stdout) // No success message
		// Stderr check removed, as the error is returned, not printed to stderr by the command itself
		// assert.Contains(t, stderr, "not tracked by socle")

		currentBranch, gitErr := git.GetCurrentBranch() // Use standard GetCurrentBranch
		require.NoError(t, gitErr)
		assert.Equal(t, "untracked-feat", currentBranch) // Should remain on the same branch
	})

	t.Run("No stack exists (only main)", func(t *testing.T) {
		_ /* repoPath */, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Start on main, no other branches/tracking

		// Action
		stdout, stderr, err := runSoCommandWithOutput(t, "top")

		// Assertions
		require.NoError(t, err)
		assert.Empty(t, stderr)
		assert.Contains(t, stdout, "Currently on the base branch 'main', which is the only branch in the stack.")

		currentBranch, gitErr := git.GetCurrentBranch()
		require.NoError(t, gitErr)
		assert.Equal(t, "main", currentBranch)
	})
}
