package cmd

import (
	"context"
	"strings"
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

	// Override the GitHub client creation function BEFORE running the sync command
	originalCreateGHClient := gh.CreateClient
	gh.CreateClient = func(ctx context.Context, owner, repo string) (gh.ClientInterface, error) {
		return mockClient, nil
	}
	t.Cleanup(func() { gh.CreateClient = originalCreateGHClient })

	// Run the sync command with auto-confirmation for deletion
	// Use --test-yes to skip the interactive prompt
	_, _, err := runSoCommandWithOutput(t, "sync", "--test-no-fetch", "--test-no-survey")

	require.NoError(t, err)
}

func TestSyncCommand_ReparentedBranchKeepsRemoteTracking(t *testing.T) {
	repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a", "feature-b"})
	defer cleanup()

	testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
	// Create fake remote tracking branch to satisfy trunk update logic
	testutils.RunCommand(t, repoPath, "git", "branch", "origin/main", "main")

	// Simulate existing remote tracking for top branch of stack
	testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-b.remote", "origin")
	testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-b.merge", "refs/heads/feature-b")

	// Mark the middle branch as merged so sync deletes it
	testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-a.socle-pr-number", "101")

	mockClient := gh.NewMockClient()
	mockClient.PRStatuses[101] = gh.PRStatusMerged

	originalCreateGHClient := gh.CreateClient
	gh.CreateClient = func(ctx context.Context, owner, repo string) (gh.ClientInterface, error) {
		return mockClient, nil
	}
	t.Cleanup(func() { gh.CreateClient = originalCreateGHClient })

	// Run sync without fetch/restack to focus on reparent logic, auto-confirm deletions
	_, _, err := runSoCommandWithOutput(t, "sync", "--test-no-fetch", "--no-restack", "--test-no-survey")
	require.NoError(t, err)

	remoteVal := strings.TrimSpace(testutils.RunCommand(t, repoPath, "git", "config", "--get", "branch.feature-b.remote"))
	require.Equal(t, "origin", remoteVal, "remote tracking should still point to origin")

	mergeVal := strings.TrimSpace(testutils.RunCommand(t, repoPath, "git", "config", "--get", "branch.feature-b.merge"))
	require.Equal(t, "refs/heads/feature-b", mergeVal, "merge ref should remain the feature branch on remote")

	parentVal := strings.TrimSpace(testutils.RunCommand(t, repoPath, "git", "config", "--get", "branch.feature-b.socle-parent"))
	require.Equal(t, "main", parentVal, "socle parent should update to the deleted branch's parent")
}
