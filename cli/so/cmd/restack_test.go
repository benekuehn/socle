// cli/so/cmd/restack_test.go
package cmd

import (
	"fmt"
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupRepoWithStack helper function
func setupRepoWithStack(t *testing.T, branches []string) (repoPath string, cleanup func()) {
	t.Helper()
	repoPath, cleanup = testutils.SetupGitRepo(t) // Starts with main commit

	// Create and track branches sequentially
	for i := 1; i < len(branches); i++ {
		parent := branches[i-1]
		branch := branches[i]
		// Create branch off parent
		testutils.RunCommand(t, repoPath, "git", "checkout", parent)
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", branch)
		// Add a unique commit to distinguish the branch
		writeFile(t, repoPath, fmt.Sprintf("%s.txt", branch), branch)
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", fmt.Sprintf("feat: commit on %s", branch))
		// Track it (using runSoCommand with test flags)
		err := runSoCommand(t, "track", fmt.Sprintf("--test-parent=%s", parent))
		require.NoError(t, err, "Setup: failed to track branch %s", branch)
	}
	// Go back to a known branch (e.g., the tip)
	testutils.RunCommand(t, repoPath, "git", "checkout", branches[len(branches)-1])
	return repoPath, cleanup
}

func TestRestackCommand(t *testing.T) {
	t.Run("Stack is already up-to-date", func(t *testing.T) {
		_, cleanup := setupRepoWithStack(t, []string{"main", "feature-a", "feature-b"})
		defer cleanup()

		// Get initial hashes
		hashA1, _ := git.GetCurrentBranchCommit("feature-a")
		hashB1, _ := git.GetCurrentBranchCommit("feature-b")

		// Action
		err := runSoCommand(t, "restack", "--no-fetch") // No fetch needed

		// Assertions
		require.NoError(t, err)
		hashA2, _ := git.GetCurrentBranchCommit("feature-a")
		hashB2, _ := git.GetCurrentBranchCommit("feature-b")
		assert.Equal(t, hashA1, hashA2, "feature-a hash should not change")
		assert.Equal(t, hashB1, hashB2, "feature-b hash should not change")
		// TODO: Assert output contains skipping messages? Requires capturing output.
	})

	t.Run("Stack needs rebase onto updated base", func(t *testing.T) {
		repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a", "feature-b"})
		defer cleanup()

		// Get original hashes
		hashA1, _ := git.GetCurrentBranchCommit("feature-a")
		hashB1, _ := git.GetCurrentBranchCommit("feature-b")
		hashMain1, _ := git.GetCurrentBranchCommit("main")

		// Setup: Add commit to main
		testutils.RunCommand(t, repoPath, "git", "checkout", "main")
		writeFile(t, repoPath, "main_change.txt", "change")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "feat: commit on main")
		hashMain2, _ := git.GetCurrentBranchCommit("main")
		require.NotEqual(t, hashMain1, hashMain2) // Verify main moved

		// Go back to tip branch
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature-b")

		// Action
		err := runSoCommand(t, "restack", "--no-fetch") // Use local main

		// Assertions
		require.NoError(t, err)
		hashA2, _ := git.GetCurrentBranchCommit("feature-a")
		hashB2, _ := git.GetCurrentBranchCommit("feature-b")
		assert.NotEqual(t, hashA1, hashA2, "feature-a hash should change")
		assert.NotEqual(t, hashB1, hashB2, "feature-b hash should change")

		// Verify parentage
		parentA, _ := git.GetMergeBase("main", "feature-a")
		parentB, _ := git.GetMergeBase("feature-a", "feature-b")
		assert.Equal(t, hashMain2, parentA, "feature-a should now be based on new main")
		assert.Equal(t, hashA2, parentB, "feature-b should now be based on new feature-a")
	})

	t.Run("Conflict during rebase", func(t *testing.T) {
		repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a"})
		defer cleanup()

		// Setup: Create conflict
		// main: file.txt = "a"
		writeFile(t, repoPath, "file.txt", "a")
		testutils.RunCommand(t, repoPath, "git", "add", "file.txt")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "add file on main")

		// feature-a: file.txt = "b"
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature-a")
		writeFile(t, repoPath, "file.txt", "b")
		testutils.RunCommand(t, repoPath, "git", "add", "file.txt")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "change file on feature-a")

		// Update main: file.txt = "c"
		testutils.RunCommand(t, repoPath, "git", "checkout", "main")
		writeFile(t, repoPath, "file.txt", "c")
		testutils.RunCommand(t, repoPath, "git", "add", "file.txt")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "update file on main")

		// Go back to feature-a
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature-a")

		// Action
		err := runSoCommand(t, "restack", "--no-fetch") // Should conflict

		// Assertions
		require.NoError(t, err, "so restack should exit cleanly (nil error) on conflict")
		// Check Git state
		isRebasing := git.IsRebaseInProgress()
		assert.True(t, isRebasing, "Git should be in a rebase state after conflict")
		// TODO: Capture stderr and assert the conflict message was printed? More complex.
	})

}
