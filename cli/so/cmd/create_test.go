// cli/so/cmd/create_test.go
package cmd

import (
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/stretchr/testify/assert"  // Using testify for assertions
	"github.com/stretchr/testify/require" // Using testify for setup checks
)

func TestCreateCommand(t *testing.T) {
	t.Run("Create branch with no changes", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup: Track main, create feature/a and track it
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature/a")
		err := runSoCommand(t, "track", "--test-parent=main") // Using helper from track_test.go setup
		require.NoError(t, err, "Setup: failed to track feature/a")

		// Action: Create feature/b
		err = runSoCommand(t, "create", "feature/b")

		// Assertions
		require.NoError(t, err, "so create failed unexpectedly")

		// Check current branch
		currentBranch, err := git.GetCurrentBranch()
		require.NoError(t, err)
		assert.Equal(t, "feature/b", currentBranch, "Should be checked out on new branch")

		// Check branch exists
		exists, err := git.BranchExists("feature/b")
		require.NoError(t, err)
		assert.True(t, exists, "New branch feature/b should exist")

		// Check tracking config
		parent, _ := git.GetGitConfig("branch.feature/b.socle-parent")
		base, _ := git.GetGitConfig("branch.feature/b.socle-base")
		assert.Equal(t, "feature/a", parent, "New branch parent should be feature/a")
		assert.Equal(t, "main", base, "New branch base should be main")

		// Check no new commit was made on feature/b immediately
		commitHashB, _ := git.GetCurrentBranchCommit("feature/b")
		commitHashA, _ := git.GetCurrentBranchCommit("feature/a")
		assert.Equal(t, commitHashA, commitHashB, "feature/b should point to same commit as feature/a initially")
	})

	t.Run("Create branch with changes and commit msg flag", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup: Track main, create feature/a, track it, make changes
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature/a")
		err := runSoCommand(t, "track", "--test-parent=main")
		require.NoError(t, err)
		writeFile(t, repoPath, "newfile.txt", "content for b")
		testutils.RunCommand(t, repoPath, "git", "add", "newfile.txt") // Stage changes beforehand

		// Action: Create feature/b with message flag, simulate auto-staging
		err = runSoCommand(t, "create", "feature/b", "-m", "Add newfile", "--test-stage-choice=add-all")

		// Assertions
		require.NoError(t, err, "so create failed unexpectedly")

		currentBranch, _ := git.GetCurrentBranch()
		assert.Equal(t, "feature/b", currentBranch)

		parent, _ := git.GetGitConfig("branch.feature/b.socle-parent")
		base, _ := git.GetGitConfig("branch.feature/b.socle-base")
		assert.Equal(t, "feature/a", parent)
		assert.Equal(t, "main", base)

		// Check commit was made on feature/b
		commitHashB, _ := git.GetCurrentBranchCommit("feature/b")
		commitHashA, _ := git.GetCurrentBranchCommit("feature/a")
		assert.NotEqual(t, commitHashA, commitHashB, "feature/b should have a new commit")

		// Check commit message
		commitMsg, _ := git.GetFirstCommitSubject("feature/a", "feature/b")
		assert.Equal(t, "Add newfile", commitMsg, "Commit message mismatch")

		// Check file content
		content := readFile(t, repoPath, "newfile.txt")
		assert.Equal(t, "content for b", content)
	})

	t.Run("Create branch fails if parent not tracked", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup: Create feature/a but DO NOT track it
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature/a")

		// Action: Try to create feature/b
		err := runSoCommand(t, "create", "feature/b")

		// Assertion
		require.Error(t, err, "so create should fail if parent is not tracked")
		assert.Contains(t, err.Error(), "not tracked by socle", "Error message mismatch")

	})

	t.Run("Create branch fails if branch already exists", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		//Setup: Track main, create feature/a and track it, create feature/b manually
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature/a")
		err := runSoCommand(t, "track", "--test-parent=main")
		require.NoError(t, err)
		testutils.RunCommand(t, repoPath, "git", "branch", "feature/b")   // Create feature/b
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature/a") // Be on parent

		// Acion: Try to create feature/b again
		err = runSoCommand(t, "create", "feature/b")

		// Assertion
		require.Error(t, err, "so create should fail if branch exists")

		assert.Contains(t, err.Error(), "already exists", "Error message mismatch")
	})
	// TODO: Add tests for prompting (needs more setup or PTY)
	// TODO: Add tests for staging choices ('add-p', 'cancel')
	// TODO: Add test for 'add -p' but staging nothing (--test-addp-empty)
	// TODO: Add test for invalid branch name
	// TODO: Add test for creating off base branch directly
}
