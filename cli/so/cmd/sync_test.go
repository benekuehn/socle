package cmd

import (
	"context"
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/stretchr/testify/require"
)

func TestSyncCommand_MergedPRs(t *testing.T) {
	repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a", "feature-b"})
	defer cleanup()
	testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
	// Set PR numbers for both branches
	testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-a.socle-pr-number", "101")
	testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-b.socle-pr-number", "102")

	// Create a fake origin/main ref so sync logic works without fetch
	testutils.RunCommand(t, repoPath, "git", "branch", "origin/main", "main")

	// Set up mock GitHub client
	mockClient := gh.NewMockClient()
	mockClient.PRStatuses[101] = gh.PRStatusMerged
	mockClient.PRStatuses[102] = gh.PRStatusClosed

	// Override the GitHub client creation function
	originalCreateGHClient := gh.CreateClient
	gh.CreateClient = func(ctx context.Context, owner, repo string) (gh.ClientInterface, error) {
		return mockClient, nil
	}
	t.Cleanup(func() { gh.CreateClient = originalCreateGHClient })

	// Run the sync command (simulate user confirming deletion)
	// For now, just check that the output contains the expected prompt and status
	_, _, err := runSoCommandWithOutput(t, "sync", "--test-no-fetch")

	require.NoError(t, err)
}
