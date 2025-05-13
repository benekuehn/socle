// cli/so/cmd/log_test.go
package cmd

import (
	"regexp"
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stripAnsi removes ANSI escape codes from a string.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripAnsi(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func TestLogCommand(t *testing.T) {
	t.Run("Log on base branch", func(t *testing.T) {
		_, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		stdout, _, err := runSoCommandWithOutput(t, "log")

		require.NoError(t, err)
		strippedStdout := stripAnsi(stdout)
		assert.Contains(t, strippedStdout, "Currently on the base branch 'main'.")
		assert.NotContains(t, strippedStdout, "●")
	})

	t.Run("Log on untracked branch", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature-untracked")

		stdout, _, err := runSoCommandWithOutput(t, "log")

		require.NoError(t, err)
		strippedStdout := stripAnsi(stdout)
		assert.Contains(t, strippedStdout, "Branch 'feature-untracked' is not currently tracked by socle.")
		assert.Contains(t, strippedStdout, "Use 'so track' to associate it")
		assert.NotContains(t, strippedStdout, "●")
	})

	t.Run("Log simple tracked stack (up-to-date, no PR)", func(t *testing.T) {
		repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a", "feature-b"})
		defer cleanup()
		testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/example/test-repo.git")
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature-b")

		stdout, _, err := runSoCommandWithOutput(t, "log")

		require.NoError(t, err)
		actualContent := stripAnsi(stdout)
		assert.Contains(t, actualContent, "  ● ○ feature-b (up-to-date, no PR submitted)")
		assert.Contains(t, actualContent, "  ● ○ feature-a (up-to-date, no PR submitted)")
		assert.Contains(t, actualContent, "      main (base)")
	})

	t.Run("Log stack needs restack (no PR)", func(t *testing.T) {
		repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a", "feature-b"})
		defer cleanup()
		testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/example/test-repo.git")

		testutils.RunCommand(t, repoPath, "git", "checkout", "main")
		writeFile(t, repoPath, "main_change.txt", "change")
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", "change main")
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature-b")

		stdout, _, err := runSoCommandWithOutput(t, "log")

		require.NoError(t, err)
		actualContent := stripAnsi(stdout)
		assert.Contains(t, actualContent, "  ● ○ feature-b (up-to-date, no PR submitted)")
		assert.Contains(t, actualContent, "  ● ○ feature-a (needs restack, no PR submitted)")
		assert.Contains(t, actualContent, "      main (base)")
	})

	t.Run("Log stack with PR config (PR not found scenario)", func(t *testing.T) {
		repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a"})
		defer cleanup()
		testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/example/test-repo.git")
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature-a")

		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-a.socle-pr-number", "123")

		stdout, _, err := runSoCommandWithOutput(t, "log")

		require.NoError(t, err)
		actualContent := stripAnsi(stdout)
		assert.Contains(t, actualContent, "  ● ○ feature-a (up-to-date, no PR submitted)")
		assert.Contains(t, actualContent, "      main (base)")
	})

	t.Run("Log stack with invalid PR config", func(t *testing.T) {
		repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a"})
		defer cleanup()
		testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/example/test-repo.git")
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature-a")

		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-a.socle-pr-number", "not-a-number")

		stdout, _, err := runSoCommandWithOutput(t, "log")

		require.NoError(t, err)
		actualContent := stripAnsi(stdout)
		assert.Contains(t, actualContent, "  ● ○ feature-a (up-to-date, no PR submitted)")
		assert.Contains(t, actualContent, "      main (base)")
	})

	t.Run("Log stack with current branch not at the top", func(t *testing.T) {
		repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-parent", "feature-current", "feature-topmost"})
		defer cleanup()
		testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/example/test-repo.git")
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature-current")

		stdout, _, err := runSoCommandWithOutput(t, "log")

		require.NoError(t, err)
		actualContent := stripAnsi(stdout)
		assert.Contains(t, actualContent, "  ● ○ feature-topmost (up-to-date, no PR submitted)")
		assert.Contains(t, actualContent, "  ● ○ feature-current (up-to-date, no PR submitted)")
		assert.Contains(t, actualContent, "  ● ○ feature-parent (up-to-date, no PR submitted)")
		assert.Contains(t, actualContent, "      main (base)")
	})
}
