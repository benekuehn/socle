package cmd

import (
	"strings"
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
)

func TestTrackCommand(t *testing.T) {
	t.Run("Track new branch successfully", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup() // Ensure we cd back

		// Setup: Create feature/a branch
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature/a")

		// Action: Run 'so track', simulating selection of 'main'
		err := runSoCommand(t, "track", "--test-parent=main")

		// Assertion 1: Command should succeed
		if err != nil {
			t.Fatalf("so track failed unexpectedly: %v", err)
		}

		// Assertion 2: Check Git config
		parent, err := git.GetGitConfig("branch.feature/a.socle-parent")
		if err != nil {
			t.Fatalf("Failed to get socle-parent config after track: %v", err)
		}
		if parent != "main" {
			t.Errorf("Expected socle-parent to be 'main', but got '%s'", parent)
		}

		base, err := git.GetGitConfig("branch.feature/a.socle-base")
		if err != nil {
			t.Fatalf("Failed to get socle-base config after track: %v", err)
		}
		if base != "main" {
			t.Errorf("Expected socle-base to be 'main', but got '%s'", base)
		}
	})

	t.Run("Attempt to track already tracked branch", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature/a")
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature/a.socle-parent", "main")
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature/a.socle-base", "main")

		// Action: Run 'so track' again
		err := runSoCommand(t, "track")

		// Assertion 1: Command should succeed (informational exit)
		if err != nil {
			t.Fatalf("so track failed unexpectedly when branch already tracked: %v", err)
		}

		// Assertion 2: Config should remain unchanged
		parent, err := git.GetGitConfig("branch.feature/a.socle-parent")
		if err != nil {
			t.Fatalf("Failed to get socle-parent config: %v", err)
		}
		if parent != "main" {
			t.Errorf("socle-parent changed unexpectedly, got '%s'", parent)
		}
		base, err := git.GetGitConfig("branch.feature/a.socle-base")
		if err != nil {
			t.Fatalf("Failed to get socle-base config: %v", err)
		}
		if base != "main" {
			t.Errorf("socle-base changed unexpectedly, got '%s'", base)
		}
	})

	t.Run("Attempt to track base branch", func(t *testing.T) {
		_, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		err := runSoCommand(t, "track")

		expectedErrorMsg := "cannot track a base branch ('main') itself"
		if !strings.Contains(err.Error(), expectedErrorMsg) {
			t.Errorf("Expected error message containing '%s', but got: %v", expectedErrorMsg, err)
		}
	})
}
