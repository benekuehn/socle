package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
)

func TestUntrackCommand(t *testing.T) {
	t.Run("Untrack branch successfully", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup: Create and track a feature branch
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature/a")
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature/a.socle-parent", "main")
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature/a.socle-base", "main")

		// Action: Run 'so untrack'
		err := runSoCommand(t, "untrack")

		// Assertion 1: Command should succeed
		if err != nil {
			t.Fatalf("so untrack failed unexpectedly: %v", err)
		}

		// Assertion 2: Check Git config is cleared
		parent, err := git.GetGitConfig("branch.feature/a.socle-parent")
		if err == nil {
			t.Errorf("Expected socle-parent config to be cleared, but got '%s'", parent)
		} else if !errors.Is(err, git.ErrConfigNotFound) {
			t.Errorf("Expected ErrConfigNotFound for socle-parent, but got: %v", err)
		}

		base, err := git.GetGitConfig("branch.feature/a.socle-base")
		if err == nil {
			t.Errorf("Expected socle-base config to be cleared, but got '%s'", base)
		} else if !errors.Is(err, git.ErrConfigNotFound) {
			t.Errorf("Expected ErrConfigNotFound for socle-base, but got: %v", err)
		}
	})

	t.Run("Attempt to untrack base branch", func(t *testing.T) {
		_, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		err := runSoCommand(t, "untrack")

		expectedErrorMsg := "cannot untrack a base branch ('main')"
		if !strings.Contains(err.Error(), expectedErrorMsg) {
			t.Errorf("Expected error message containing '%s', but got: %v", expectedErrorMsg, err)
		}
	})

	t.Run("Attempt to untrack branch with children", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		// Setup: Create parent and child branches
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature/parent")
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature/parent.socle-parent", "main")
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature/parent.socle-base", "main")

		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature/child")
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature/child.socle-parent", "feature/parent")
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature/child.socle-base", "main")

		// Switch back to parent branch
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature/parent")

		// Action: Try to untrack parent branch
		err := runSoCommand(t, "untrack")

		// Assertion: Should fail with appropriate error
		expectedErrorMsg := "cannot untrack branch 'feature/parent' because it has children depending on it"
		if !strings.Contains(err.Error(), expectedErrorMsg) {
			t.Errorf("Expected error message containing '%s', but got: %v", expectedErrorMsg, err)
		}

		// Verify config is still intact
		parent, err := git.GetGitConfig("branch.feature/parent.socle-parent")
		if err != nil || parent != "main" {
			t.Errorf("Parent branch config was unexpectedly modified")
		}
	})
}
