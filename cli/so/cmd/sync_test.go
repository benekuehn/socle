package cmd

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/google/go-github/v71/github"
	"github.com/stretchr/testify/assert"
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

func TestSyncCommand_DiscoverPRs(t *testing.T) {
	repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a", "feature-b", "feature-c"})
	defer cleanup()
	testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")

	// Create a fake origin/main ref so sync logic works without fetch
	testutils.RunCommand(t, repoPath, "git", "branch", "origin/main", "main")

	// Set up mock GitHub client with PR discovery
	mockClient := gh.NewMockClient()

	// Mock PR discovery for feature-a and feature-c (feature-b has no PR)
	mockPR1 := &github.PullRequest{
		Number: github.Ptr(201),
		Title:  github.Ptr("Feature A PR"),
	}
	mockPR3 := &github.PullRequest{
		Number: github.Ptr(203),
		Title:  github.Ptr("Feature C PR"),
	}

	mockClient.On("FindPullRequestsForBranch", "feature-a", "all").Return([]*github.PullRequest{mockPR1}, nil)
	mockClient.On("FindPullRequestsForBranch", "feature-b", "all").Return(nil, nil) // No PR for feature-b
	mockClient.On("FindPullRequestsForBranch", "feature-c", "all").Return([]*github.PullRequest{mockPR3}, nil)

	// Override the GitHub client creation function
	originalCreateGHClient := gh.CreateClient
	gh.CreateClient = func(ctx context.Context, owner, repo string) (gh.ClientInterface, error) {
		return mockClient, nil
	}
	t.Cleanup(func() { gh.CreateClient = originalCreateGHClient })

	// Run the sync command
	_, _, err := runSoCommandWithOutput(t, "sync", "--test-no-fetch", "--test-no-survey")
	require.NoError(t, err)

	// Verify that PR numbers were discovered and stored
	prNumberA, err := git.GetStoredPRNumber("feature-a")
	assert.NoError(t, err)
	assert.Equal(t, 201, prNumberA)

	prNumberB, err := git.GetStoredPRNumber("feature-b")
	assert.NoError(t, err)
	assert.Equal(t, 0, prNumberB) // Should remain 0 since no PR was found

	prNumberC, err := git.GetStoredPRNumber("feature-c")
	assert.NoError(t, err)
	assert.Equal(t, 203, prNumberC)

	mockClient.AssertExpectations(t)
}

func TestSyncCommand_DiscoverPRsWithExistingPRs(t *testing.T) {
	repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a", "feature-b"})
	defer cleanup()
	testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")

	// Create a fake origin/main ref so sync logic works without fetch
	testutils.RunCommand(t, repoPath, "git", "branch", "origin/main", "main")

	// Set existing PR number for feature-a
	err := git.SetStoredPRNumber("feature-a", 999)
	require.NoError(t, err)

	// Set up mock GitHub client
	mockClient := gh.NewMockClient()

	// Mock PR discovery - should not be called for feature-a since it already has a PR number
	mockPR2 := &github.PullRequest{
		Number: github.Ptr(202),
		Title:  github.Ptr("Feature B PR"),
	}

	mockClient.On("FindPullRequestsForBranch", "feature-b", "all").Return([]*github.PullRequest{mockPR2}, nil)

	// Override the GitHub client creation function
	originalCreateGHClient := gh.CreateClient
	gh.CreateClient = func(ctx context.Context, owner, repo string) (gh.ClientInterface, error) {
		return mockClient, nil
	}
	t.Cleanup(func() { gh.CreateClient = originalCreateGHClient })

	// Run the sync command
	_, _, err = runSoCommandWithOutput(t, "sync", "--test-no-fetch", "--test-no-survey")
	require.NoError(t, err)

	// Verify that existing PR number was preserved
	prNumberA, err := git.GetStoredPRNumber("feature-a")
	assert.NoError(t, err)
	assert.Equal(t, 999, prNumberA) // Should keep existing PR number

	// Verify that new PR number was discovered for feature-b
	prNumberB, err := git.GetStoredPRNumber("feature-b")
	assert.NoError(t, err)
	assert.Equal(t, 202, prNumberB)

	mockClient.AssertExpectations(t)
}

func TestSyncCommand_CleansUpDiscoveredMergedPR(t *testing.T) {
	repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a"})
	defer cleanup()
	testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
	testutils.RunCommand(t, repoPath, "git", "branch", "origin/main", "main")

	mockClient := gh.NewMockClient()
	mockClient.On("FindPullRequestsForBranch", "feature-a", "all").Return([]*github.PullRequest{{
		Number: github.Ptr(201),
	}}, nil).Once()
	mockClient.PRStatuses[201] = gh.PRStatusMerged
	originalCreateGHClient := gh.CreateClient
	gh.CreateClient = func(context.Context, string, string) (gh.ClientInterface, error) { return mockClient, nil }
	t.Cleanup(func() { gh.CreateClient = originalCreateGHClient })

	_, _, err := runSoCommandWithOutput(t, "sync", "--test-no-fetch", "--test-no-survey")
	require.NoError(t, err)
	checkBranch := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/feature-a")
	checkBranch.Dir = repoPath
	assert.Error(t, checkBranch.Run())
	mockClient.AssertExpectations(t)
}

func TestSyncCommandStopsOnDiscoveryError(t *testing.T) {
	repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a"})
	defer cleanup()
	testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
	testutils.RunCommand(t, repoPath, "git", "branch", "origin/main", "main")

	mockClient := gh.NewMockClient()
	mockClient.On("FindPullRequestsForBranch", "feature-a", "all").Return(nil, errors.New("unavailable")).Once()
	originalCreateGHClient := gh.CreateClient
	gh.CreateClient = func(context.Context, string, string) (gh.ClientInterface, error) { return mockClient, nil }
	t.Cleanup(func() { gh.CreateClient = originalCreateGHClient })

	_, _, err := runSoCommandWithOutput(t, "sync", "--test-no-fetch", "--test-no-survey")
	require.Error(t, err)
	mockClient.AssertExpectations(t)
}
