// cli/so/cmd/log_test.go
package cmd

import (
	"fmt"
	"regexp"
	"strings"
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

// Helper to build expected log output string for a branch (plain text)
func expectedBranchOutput(dot, branchName, status string) string {
	return fmt.Sprintf("  %s %s %s\n", dot, branchName, status)
}

// Helper to build expected base output (plain text)
func expectedBaseOutput(baseBranchName string) string {
	return fmt.Sprintf("  ● %s\n", baseBranchName)
}

func TestLogCommand(t *testing.T) {
	t.Run("Log on base branch", func(t *testing.T) {
		_, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		stdout, stderr, err := runSoCommandWithOutput(t, "log")

		require.NoError(t, err)
		strippedStderr := stripAnsi(stderr)
		assert.Regexp(t, regexp.MustCompile(`\[DEBUG\] Total execution time for log command: \S+`), strippedStderr)

		strippedStdout := stripAnsi(stdout)
		assert.Contains(t, strippedStdout, "Currently on the base branch 'main'.")
		assert.NotContains(t, strippedStdout, "●")
	})

	t.Run("Log on untracked branch", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature-untracked")

		stdout, stderr, err := runSoCommandWithOutput(t, "log")

		require.NoError(t, err)
		strippedStderr := stripAnsi(stderr)
		assert.Regexp(t, regexp.MustCompile(`\[DEBUG\] Total execution time for log command: \S+`), strippedStderr)

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

		stdout, stderr, err := runSoCommandWithOutput(t, "log")

		require.NoError(t, err)
		strippedStderr := stripAnsi(stderr)
		assert.Regexp(t, regexp.MustCompile(`\[DEBUG\] Time for GitHub PR fetches: \S+`), strippedStderr)
		assert.Regexp(t, regexp.MustCompile(`\[DEBUG\] Total execution time for log command: \S+`), strippedStderr)

		var expectedOutput strings.Builder
		expectedOutput.WriteString(expectedBranchOutput("○", "feature-b", ""))
		expectedOutput.WriteString(expectedBranchOutput("○", "feature-a", ""))
		expectedOutput.WriteString(expectedBaseOutput("main"))

		actualContent := stripAnsi(stdout)
		actualContent = strings.TrimPrefix(actualContent, "\n")
		actualContent = strings.TrimSuffix(actualContent, "\n")
		assert.Equal(t, expectedOutput.String(), actualContent)
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

		stdout, stderr, err := runSoCommandWithOutput(t, "log")

		require.NoError(t, err)
		strippedStderr := stripAnsi(stderr)
		assert.Regexp(t, regexp.MustCompile(`\[DEBUG\] Time for GitHub PR fetches: \S+`), strippedStderr)
		assert.Regexp(t, regexp.MustCompile(`\[DEBUG\] Total execution time for log command: \S+`), strippedStderr)

		var expectedOutput strings.Builder
		expectedOutput.WriteString(expectedBranchOutput("○", "feature-b", ""))
		expectedOutput.WriteString(expectedBranchOutput("○", "feature-a", "(needs rebase)"))
		expectedOutput.WriteString(expectedBaseOutput("main"))

		actualContent := stripAnsi(stdout)
		actualContent = strings.TrimPrefix(actualContent, "\n")
		actualContent = strings.TrimSuffix(actualContent, "\n")
		assert.Equal(t, expectedOutput.String(), actualContent)
	})

	t.Run("Log stack with PR config (PR not found scenario)", func(t *testing.T) {
		repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a"})
		defer cleanup()
		testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/example/test-repo.git")
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature-a")

		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-a.socle-pr-number", "123")

		stdout, stderr, err := runSoCommandWithOutput(t, "log")

		require.NoError(t, err)
		strippedStderr := stripAnsi(stderr)
		assert.Regexp(t, regexp.MustCompile(`\[DEBUG\] Time for GitHub client initialization: \S+`), strippedStderr)
		assert.Regexp(t, regexp.MustCompile(`\[DEBUG\] Time for GitHub PR fetches: \S+`), strippedStderr)
		assert.Regexp(t, regexp.MustCompile(`Warning: Could not fetch PR #123 for 'feature-a': pull request #123 not found`), strippedStderr)
		assert.Regexp(t, regexp.MustCompile(`\[DEBUG\] Total execution time for log command: \S+`), strippedStderr)

		var expectedOutput strings.Builder
		expectedOutput.WriteString(expectedBranchOutput("○", "feature-a", ""))
		expectedOutput.WriteString(expectedBaseOutput("main"))

		actualContent := stripAnsi(stdout)
		actualContent = strings.TrimPrefix(actualContent, "\n")
		actualContent = strings.TrimSuffix(actualContent, "\n")
		assert.Equal(t, expectedOutput.String(), actualContent)
	})

	t.Run("Log stack with invalid PR config", func(t *testing.T) {
		repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a"})
		defer cleanup()
		testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/example/test-repo.git")
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature-a")

		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-a.socle-pr-number", "not-a-number")

		stdout, stderr, err := runSoCommandWithOutput(t, "log")

		require.NoError(t, err)
		strippedStderr := stripAnsi(stderr)
		assert.Regexp(t, regexp.MustCompile(`Warning: Could not parse PR number 'not-a-number' for 'feature-a'`), strippedStderr)
		assert.Regexp(t, regexp.MustCompile(`\[DEBUG\] Total execution time for log command: \S+`), strippedStderr)

		var expectedOutput strings.Builder
		expectedOutput.WriteString(expectedBranchOutput("○", "feature-a", ""))
		expectedOutput.WriteString(expectedBaseOutput("main"))

		actualContent := stripAnsi(stdout)
		actualContent = strings.TrimPrefix(actualContent, "\n")
		actualContent = strings.TrimSuffix(actualContent, "\n")
		assert.Equal(t, expectedOutput.String(), actualContent)
	})

	t.Run("Log stack with current branch not at the top", func(t *testing.T) {
		repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-parent", "feature-current", "feature-topmost"})
		defer cleanup()
		testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/example/test-repo.git")
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature-current")

		stdout, stderr, err := runSoCommandWithOutput(t, "log")

		require.NoError(t, err)
		strippedStderr := stripAnsi(stderr)
		assert.Regexp(t, regexp.MustCompile(`\[DEBUG\] Time for GitHub PR fetches: \S+`), strippedStderr)
		assert.Regexp(t, regexp.MustCompile(`\[DEBUG\] Total execution time for log command: \S+`), strippedStderr)

		var expectedOutput strings.Builder
		expectedOutput.WriteString(expectedBranchOutput("○", "feature-topmost", ""))
		expectedOutput.WriteString(expectedBranchOutput("○", "feature-current", ""))
		expectedOutput.WriteString(expectedBranchOutput("○", "feature-parent", ""))
		expectedOutput.WriteString(expectedBaseOutput("main"))

		actualContent := stripAnsi(stdout)
		actualContent = strings.TrimPrefix(actualContent, "\n")
		actualContent = strings.TrimSuffix(actualContent, "\n")
		assert.Equal(t, expectedOutput.String(), actualContent)
	})
}
