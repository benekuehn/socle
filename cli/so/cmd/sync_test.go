package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSyncCommand_MultipleStacksFromBase(t *testing.T) {
	repoPath, cleanup := setupRepoWithMultipleStacks(t)
	defer cleanup()
	testutils.RunCommand(t, repoPath, "git", "checkout", "main")
	testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")

	mockClient := gh.NewMockClient()
	originalCreateGHClient := gh.CreateClient
	gh.CreateClient = func(context.Context, string, string) (gh.ClientInterface, error) { return mockClient, nil }
	t.Cleanup(func() { gh.CreateClient = originalCreateGHClient })

	_, _, err := runSoCommandWithOutput(t, "sync", "--test-no-fetch")
	require.ErrorContains(t, err, "cannot sync from base branch 'main' with multiple stacks")
	mockClient.AssertNotCalled(t, "FindPullRequestForBranch", mock.Anything)
}

func TestSyncCommand_AdjacentDeletionsReparentSurvivor(t *testing.T) {
	repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a", "feature-b", "feature-c"})
	defer cleanup()
	testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
	testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-a.socle-pr-number", "101")
	testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-b.socle-pr-number", "102")
	testutils.RunCommand(t, repoPath, "git", "branch", "origin/main", "main")

	mockClient := gh.NewMockClient()
	mockClient.PRStatuses[101] = gh.PRStatusMerged
	mockClient.PRStatuses[102] = gh.PRStatusClosed
	mockClient.On("FindOpenPullRequestForBranch", "feature-c").Return(nil, nil).Once()

	originalCreateGHClient := gh.CreateClient
	gh.CreateClient = func(ctx context.Context, owner, repo string) (gh.ClientInterface, error) {
		return mockClient, nil
	}
	t.Cleanup(func() { gh.CreateClient = originalCreateGHClient })

	_, _, err := runSoCommandWithOutput(t, "sync", "--test-no-fetch", "--test-no-survey", "--no-restack")

	require.NoError(t, err)
	for _, branch := range []string{"feature-a", "feature-b"} {
		exists, err := git.BranchExists(branch)
		require.NoError(t, err)
		assert.False(t, exists)
	}
	exists, err := git.BranchExists("feature-c")
	require.NoError(t, err)
	assert.True(t, exists)
	parent, err := git.GetGitConfig("branch.feature-c.socle-parent")
	require.NoError(t, err)
	assert.Equal(t, "main", parent)
	base, err := git.GetGitConfig("branch.feature-c.socle-base")
	require.NoError(t, err)
	assert.Equal(t, "main", base)
	parents, err := git.GetAllSocleParents()
	require.NoError(t, err)
	for _, parent := range parents {
		assert.NotContains(t, []string{"feature-a", "feature-b"}, parent)
	}
	mockClient.AssertExpectations(t)
}

func TestSyncCommand_SingleDeletionReparentsChild(t *testing.T) {
	repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a", "feature-b"})
	defer cleanup()
	testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
	testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-a.socle-pr-number", "101")
	testutils.RunCommand(t, repoPath, "git", "branch", "origin/main", "main")

	mockClient := gh.NewMockClient()
	mockClient.PRStatuses[101] = gh.PRStatusMerged
	mockClient.On("FindOpenPullRequestForBranch", "feature-b").Return(nil, nil).Once()
	originalCreateGHClient := gh.CreateClient
	gh.CreateClient = func(context.Context, string, string) (gh.ClientInterface, error) { return mockClient, nil }
	t.Cleanup(func() { gh.CreateClient = originalCreateGHClient })

	_, _, err := runSoCommandWithOutput(t, "sync", "--test-no-fetch", "--test-no-survey", "--no-restack")
	require.NoError(t, err)
	exists, err := git.BranchExists("feature-a")
	require.NoError(t, err)
	assert.False(t, exists)
	parent, err := git.GetGitConfig("branch.feature-b.socle-parent")
	require.NoError(t, err)
	assert.Equal(t, "main", parent)
	base, err := git.GetGitConfig("branch.feature-b.socle-base")
	require.NoError(t, err)
	assert.Equal(t, "main", base)
	mockClient.AssertExpectations(t)
}

func TestSyncCommand_DeclinedDeletionLeavesBranchesUnchanged(t *testing.T) {
	repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a", "feature-b"})
	defer cleanup()
	testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
	testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-a.socle-pr-number", "101")
	testutils.RunCommand(t, repoPath, "git", "branch", "origin/main", "main")

	mockClient := gh.NewMockClient()
	mockClient.PRStatuses[101] = gh.PRStatusMerged
	mockClient.On("FindOpenPullRequestForBranch", "feature-b").Return(nil, nil).Once()
	originalCreateGHClient := gh.CreateClient
	gh.CreateClient = func(context.Context, string, string) (gh.ClientInterface, error) { return mockClient, nil }
	t.Cleanup(func() { gh.CreateClient = originalCreateGHClient })

	runner := &syncCmdRunner{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}, noFetch: true, confirm: func() (bool, error) { return false, nil }}
	require.NoError(t, runner.run(&cobra.Command{}))
	for branch, wantParent := range map[string]string{"feature-a": "main", "feature-b": "feature-a"} {
		exists, err := git.BranchExists(branch)
		require.NoError(t, err)
		assert.True(t, exists)
		parent, err := git.GetGitConfig("branch." + branch + ".socle-parent")
		require.NoError(t, err)
		assert.Equal(t, wantParent, parent)
		base, err := git.GetGitConfig("branch." + branch + ".socle-base")
		require.NoError(t, err)
		assert.Equal(t, "main", base)
	}
	mockClient.AssertExpectations(t)
}

func TestReplacementParentsRejectsInvalidGraphs(t *testing.T) {
	for _, test := range []struct {
		name    string
		parents map[string]string
		deleted []string
	}{
		{"missing parent", map[string]string{"feature-c": "feature-a"}, []string{"feature-a"}},
		{"cycle", map[string]string{"feature-c": "feature-a", "feature-a": "feature-b", "feature-b": "feature-a"}, []string{"feature-a", "feature-b"}},
		{"self parent", map[string]string{"feature-c": "feature-a", "feature-a": "feature-c"}, []string{"feature-a"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := replacementParents(test.parents, test.deleted)
			assert.Error(t, err)
		})
	}
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
	mockClient.On("FindPullRequestForBranch", "feature-b").Return(nil, nil).Once()

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
