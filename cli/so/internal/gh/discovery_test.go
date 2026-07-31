package gh

import (
	"errors"
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/google/go-github/v71/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverPRsFailsOnLookup(t *testing.T) {
	_, cleanup := testutils.SetupGitRepo(t)
	defer cleanup()

	client := NewMockClient()
	client.On("FindOpenPullRequestForBranch", "feature").Return(nil, errors.New("unavailable")).Once()

	_, err := DiscoverPRs(client, []string{"feature"})
	require.ErrorContains(t, err, "find open pull request")
	client.AssertExpectations(t)
}

func TestDiscoverCompletedPRsRejectsAmbiguousBranch(t *testing.T) {
	_, cleanup := testutils.SetupGitRepo(t)
	defer cleanup()

	client := NewMockClient()
	client.On("FindPullRequestsForBranch", "feature", "all").Return([]*github.PullRequest{
		{Number: github.Ptr(1)}, {Number: github.Ptr(2)},
	}, nil).Once()

	_, err := DiscoverCompletedPRs(client, []string{"feature"})
	require.ErrorContains(t, err, "multiple pull requests")
	client.AssertExpectations(t)
}

func TestDiscoverCompletedPRsSkipsClosedPRWithDifferentHead(t *testing.T) {
	repoPath, cleanup := testutils.SetupGitRepo(t)
	defer cleanup()
	testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature")
	testutils.RunCommand(t, repoPath, "touch", "feature.txt")
	testutils.RunCommand(t, repoPath, "git", "add", "feature.txt")
	testutils.RunCommand(t, repoPath, "git", "commit", "-m", "feature")

	client := NewMockClient()
	client.On("FindPullRequestsForBranch", "feature", "all").Return([]*github.PullRequest{{
		Number: github.Ptr(1), State: github.Ptr("closed"),
		Head: &github.PullRequestBranch{SHA: github.Ptr("old-head")},
	}}, nil).Once()

	discovered, err := DiscoverCompletedPRs(client, []string{"feature"})
	require.NoError(t, err)
	assert.Empty(t, discovered)
	prNumber, err := git.GetStoredPRNumber("feature")
	require.NoError(t, err)
	assert.Zero(t, prNumber)
	client.AssertExpectations(t)
}

func TestDiscoverCompletedPRsStoresClosedPRWithMatchingHead(t *testing.T) {
	repoPath, cleanup := testutils.SetupGitRepo(t)
	defer cleanup()
	testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature")
	testutils.RunCommand(t, repoPath, "touch", "feature.txt")
	testutils.RunCommand(t, repoPath, "git", "add", "feature.txt")
	testutils.RunCommand(t, repoPath, "git", "commit", "-m", "feature")
	head, err := git.GetCurrentBranchCommit("feature")
	require.NoError(t, err)

	client := NewMockClient()
	client.On("FindPullRequestsForBranch", "feature", "all").Return([]*github.PullRequest{{
		Number: github.Ptr(1), State: github.Ptr("closed"),
		Head: &github.PullRequestBranch{SHA: github.Ptr(head)},
	}}, nil).Once()

	discovered, err := DiscoverCompletedPRs(client, []string{"feature"})
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"feature": 1}, discovered)
	prNumber, err := git.GetStoredPRNumber("feature")
	require.NoError(t, err)
	assert.Equal(t, 1, prNumber)
	client.AssertExpectations(t)
}
