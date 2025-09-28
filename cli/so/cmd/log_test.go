// cli/so/cmd/log_test.go
package cmd

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stripAnsi removes ANSI escape codes from a string.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// hyperlinkRegex matches OSC 8 escape sequences for hyperlinks
var hyperlinkRegex = regexp.MustCompile(`\x1b\]8;;[^\x1b]*\x1b\\[^\x1b]*\x1b\]8;;\x1b\\`)

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
		assert.Contains(t, actualContent, "  ● ○ feature-a (up-to-date, pr check failed)")
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

	t.Run("Log stack with PR hyperlink", func(t *testing.T) {
		repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a"})
		defer cleanup()
		testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/example/test-repo.git")
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature-a")

		// Set up PR number
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-a.socle-pr-number", "123")

		// Create mock GitHub client
		mockClient := gh.NewMockClient()
		mockClient.PRStatuses[123] = gh.PRStatusOpen // Set PR status to Open

		// Override the GitHub client creation function
		originalCreateGHClient := gh.CreateClient
		gh.CreateClient = func(ctx context.Context, owner, repo string) (gh.ClientInterface, error) {
			return mockClient, nil
		}
		t.Cleanup(func() { gh.CreateClient = originalCreateGHClient })

		stdout, _, err := runSoCommandWithOutput(t, "log")

		require.NoError(t, err)

		// Check for hyperlink format in the output
		hyperlinkMatches := hyperlinkRegex.FindAllString(stdout, -1)
		assert.NotEmpty(t, hyperlinkMatches, "Expected to find hyperlink in output")

		// Verify the hyperlink format and content
		for _, match := range hyperlinkMatches {
			assert.Contains(t, match, "\x1b]8;;", "Expected hyperlink start sequence")
			assert.Contains(t, match, "\x1b\\", "Expected hyperlink separator")
			assert.Contains(t, match, "\x1b]8;;\x1b\\", "Expected hyperlink end sequence")
			assert.Contains(t, match, "https://github.com/mock/mock/pull/123", "Expected PR URL in hyperlink")
			assert.Contains(t, match, "pr open", "Expected PR status in hyperlink text")
		}

		// Also verify the stripped content
		strippedContent := stripAnsi(stdout)
		assert.Contains(t, strippedContent, "feature-a")
		assert.Contains(t, strippedContent, "pr open")
	})

	t.Run("Log on base branch with multiple stacks", func(t *testing.T) {
		repoPath, cleanup := setupRepoWithMultipleStacks(t)
		defer cleanup()
		testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/example/test-repo.git")
		
		// Checkout to base branch
		testutils.RunCommand(t, repoPath, "git", "checkout", "main")

		stdout, _, err := runSoCommandWithOutput(t, "log")

		require.NoError(t, err)
		actualContent := stripAnsi(stdout)
		
		// Should show header with stack count
		assert.Contains(t, actualContent, "2 stacks from base 'main':")
		
		// Should show both stacks with detailed info
		assert.Contains(t, actualContent, "● ○ feature-b (up-to-date, no PR submitted)")
		assert.Contains(t, actualContent, "● ○ feature-a (up-to-date, no PR submitted)")
		assert.Contains(t, actualContent, "● ○ feature-y (up-to-date, no PR submitted)")
		assert.Contains(t, actualContent, "● ○ feature-x (up-to-date, no PR submitted)")
		
		// Should show base branch for each stack
		assert.Contains(t, actualContent, "main (base)")
		
		// Should have spacing between stacks
		lines := strings.Split(actualContent, "\n")
		hasBlankLineBetweenStacks := false
		for i, line := range lines {
			if strings.TrimSpace(line) == "main (base)" && i+1 < len(lines) {
				nextLine := lines[i+1]
				if strings.TrimSpace(nextLine) == "" {
					hasBlankLineBetweenStacks = true
					break
				}
			}
		}
		assert.True(t, hasBlankLineBetweenStacks, "Should have blank line between stacks")
	})
}
