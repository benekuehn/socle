package gh

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/google/go-github/v83/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverPRs(t *testing.T) {
	t.Run("keeps existing metadata", func(t *testing.T) {
		repo, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()
		testutils.RunCommand(t, repo, "git", "config", "branch.feature.socle-pr-number", "42")

		prs, err := DiscoverPRs(NewMockClient(), []string{"feature"})
		require.NoError(t, err)
		assert.Equal(t, map[string]int{"feature": 42}, prs)
	})

	t.Run("accepts confirmed absence", func(t *testing.T) {
		_, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()
		client := NewMockClient()
		client.On("FindOpenPullRequestForBranch", "feature").Return(nil, nil).Once()

		prs, err := DiscoverPRs(client, []string{"feature"})
		require.NoError(t, err)
		assert.Empty(t, prs)
		client.AssertExpectations(t)
	})

	t.Run("stores discovered PR", func(t *testing.T) {
		_, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()
		client := NewMockClient()
		client.On("FindOpenPullRequestForBranch", "feature").Return(&github.PullRequest{Number: github.Ptr(42)}, nil).Once()

		prs, err := DiscoverPRs(client, []string{"feature"})
		require.NoError(t, err)
		assert.Equal(t, map[string]int{"feature": 42}, prs)
		number, err := git.GetStoredPRNumber("feature")
		require.NoError(t, err)
		assert.Equal(t, 42, number)
		client.AssertExpectations(t)
	})

	t.Run("fails on config read", func(t *testing.T) {
		_, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()
		t.Setenv("GIT_CONFIG_COUNT", "1")
		t.Setenv("GIT_CONFIG_KEY_0", "bad key")
		t.Setenv("GIT_CONFIG_VALUE_0", "x")
		_, err := DiscoverPRs(NewMockClient(), []string{"feature"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "read stored PR number")
	})

	t.Run("fails on lookup", func(t *testing.T) {
		_, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()
		client := NewMockClient()
		client.On("FindOpenPullRequestForBranch", "feature").Return(nil, errors.New("unavailable")).Once()

		_, err := DiscoverPRs(client, []string{"feature"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "find open pull request")
		client.AssertExpectations(t)
	})

	t.Run("fails on config write", func(t *testing.T) {
		repo, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()
		configLock := filepath.Join(repo, ".git", "config.lock")
		require.NoError(t, os.WriteFile(configLock, nil, 0o600))
		t.Cleanup(func() { _ = os.Remove(configLock) })
		client := NewMockClient()
		client.On("FindOpenPullRequestForBranch", "feature").Return(&github.PullRequest{Number: github.Ptr(42)}, nil).Once()

		_, err := DiscoverPRs(client, []string{"feature"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "store discovered PR number")
		client.AssertExpectations(t)
	})

	t.Run("accepts no branches", func(t *testing.T) {
		prs, err := DiscoverPRs(NewMockClient(), nil)
		require.NoError(t, err)
		assert.Empty(t, prs)
	})
}

func TestDiscoverAllPRsStoresClosedPR(t *testing.T) {
	repo, cleanup := testutils.SetupGitRepo(t)
	defer cleanup()
	testutils.RunCommand(t, repo, "git", "checkout", "-b", "feature")
	headSHA, err := git.GetCurrentCommit()
	require.NoError(t, err)
	client := NewMockClient()
	client.On("FindPullRequestForBranch", "feature").Return(&github.PullRequest{
		Number: github.Ptr(42),
		State:  github.Ptr("closed"),
		Head:   &github.PullRequestBranch{SHA: github.Ptr(headSHA)},
	}, nil).Once()

	prs, err := DiscoverAllPRs(client, []string{"feature"})
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"feature": 42}, prs)
	number, err := git.GetStoredPRNumber("feature")
	require.NoError(t, err)
	assert.Equal(t, 42, number)
	client.AssertExpectations(t)
}

func TestDiscoverAllPRsSkipsClosedPRWithStaleHead(t *testing.T) {
	repo, cleanup := testutils.SetupGitRepo(t)
	defer cleanup()
	testutils.RunCommand(t, repo, "git", "checkout", "-b", "feature")
	client := NewMockClient()
	client.On("FindPullRequestForBranch", "feature").Return(&github.PullRequest{
		Number: github.Ptr(42),
		State:  github.Ptr("closed"),
		Head:   &github.PullRequestBranch{SHA: github.Ptr("stale")},
	}, nil).Once()

	prs, err := DiscoverAllPRs(client, []string{"feature"})
	require.NoError(t, err)
	assert.Empty(t, prs)
	number, err := git.GetStoredPRNumber("feature")
	require.NoError(t, err)
	assert.Zero(t, number)
	client.AssertExpectations(t)
}
