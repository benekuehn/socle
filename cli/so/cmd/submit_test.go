package cmd

import (
	"context"
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/google/go-github/v71/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock GitHub Client ---

type MockGHClient struct {
	mock.Mock // Embed testify mock object
}

func (m *MockGHClient) GetPullRequest(number int) (*github.PullRequest, error) {
	args := m.Called(number)
	// Return nil if the first return arg isn't a PR pointer
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.PullRequest), args.Error(1)
}

func (m *MockGHClient) CreatePullRequest(head, base, title, body string, isDraft bool) (*github.PullRequest, error) {
	args := m.Called(head, base, title, body, isDraft)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.PullRequest), args.Error(1)
}

func (m *MockGHClient) UpdatePullRequestBase(number int, newBase string) (*github.PullRequest, error) {
	args := m.Called(number, newBase)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.PullRequest), args.Error(1)
}

func (m *MockGHClient) CreateComment(issueNumber int, body string) (*github.IssueComment, error) {
	args := m.Called(issueNumber, body)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.IssueComment), args.Error(1)
}

func (m *MockGHClient) UpdateComment(commentID int64, body string) (*github.IssueComment, error) {
	args := m.Called(commentID, body)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.IssueComment), args.Error(1)
}

func (m *MockGHClient) FindCommentWithMarker(issueNumber int, marker string) (commentID int64, err error) {
	args := m.Called(issueNumber, marker)
	return args.Get(0).(int64), args.Error(1) // Assert type for int64
}

func (m *MockGHClient) GetIssueComment(commentID int64) (*github.IssueComment, error) {
	args := m.Called(commentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.IssueComment), args.Error(1)
}

func TestSubmitCommand(t *testing.T) {
	originalCreateGHClient := createGHClient // Store original
	// Restore original after all tests in this function finish
	t.Cleanup(func() { createGHClient = originalCreateGHClient })

	t.Run("Submit first branch creates PR and comment", func(t *testing.T) {
		// Setup: main -> feature-a (tracked)
		repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a"})
		defer cleanup()
		testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature-a") // Be on the branch to submit

		// --- Setup Mock ---
		mockClient := new(MockGHClient)
		createGHClient = func(ctx context.Context, owner, repo string) (gh.ClientInterface, error) {
			assert.Equal(t, "test-owner", owner)
			assert.Equal(t, "test-repo", repo)
			return mockClient, nil
		}
		// No need for defer here, t.Cleanup handles it for the parent test

		// Expectations for feature-a
		// Assume config check might happen, return not found
		mockClient.On("GetPullRequest", mock.AnythingOfType("int")).Return(nil, git.ErrConfigNotFound).Maybe()
		// Expect PR creation
		mockClient.On("CreatePullRequest", "feature-a", "main", "feat: commit on feature-a", "Test Body A", false).Return(
			&github.PullRequest{Number: github.Ptr(101), HTMLURL: github.Ptr("url-a"), Title: github.Ptr("feat: commit on feature-a")}, nil,
		).Once()
		// Expect comment creation (assume no existing comment found)
		mockClient.On("FindCommentWithMarker", 101, mock.AnythingOfType("string")).Return(int64(0), nil).Once()
		mockClient.On("CreateComment", 101, mock.AnythingOfType("string")).Return(
			&github.IssueComment{ID: github.Ptr(int64(5001))}, nil,
		).Once()
		// --- End Mock Setup ---

		// Action: Run 'so submit' with test flags
		err := runSoCommand(t, "submit",
			"--no-push",
			"--no-draft",
			"--test-title=feat: commit on feature-a", // Title specific to this test
			"--test-body=Test Body A",                // Body specific to this test
		)

		// Assertions
		require.NoError(t, err)
		mockClient.AssertExpectations(t) // Verify calls

		// Check Git Config was written for feature-a
		prNumA, _ := git.GetGitConfig("branch.feature-a.socle-pr-number")
		commentIdA, _ := git.GetGitConfig("branch.feature-a.socle-comment-id")
		assert.Equal(t, "101", prNumA)
		assert.Equal(t, "5001", commentIdA)
	})

	t.Run("Submit second branch creates PR and comment", func(t *testing.T) {
		// Setup: main -> feature-a (tracked, PR 101) -> feature-b (tracked)
		repoPath, cleanup := setupRepoWithStack(t, []string{"main", "feature-a", "feature-b"})
		defer cleanup()
		testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
		// Pre-seed config for feature-a's existing PR
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-a.socle-pr-number", "101")
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature-a.socle-comment-id", "5001") // Assume comment exists
		testutils.RunCommand(t, repoPath, "git", "checkout", "feature-b")                                          // Be on the branch to submit

		// --- Setup Mock ---
		mockClient := new(MockGHClient)
		createGHClient = func(ctx context.Context, owner, repo string) (gh.ClientInterface, error) {
			assert.Equal(t, "test-owner", owner)
			assert.Equal(t, "test-repo", repo)
			return mockClient, nil
		}
		// No need for defer here, t.Cleanup handles it for the parent test

		// Expectations for feature-a (update path)
		mockClient.On("GetPullRequest", 101).Return( // Simulate finding PR 101
			&github.PullRequest{Number: github.Ptr(101), HTMLURL: github.Ptr("url-a"), Title: github.Ptr("feat: commit on feature-a"), Base: &github.PullRequestBranch{Ref: github.Ptr("main")}}, nil,
		).Once()
		// Expect FindComment for PR 101 (feature-a) - should return existing comment ID 5001
		mockClient.On("FindCommentWithMarker", 101, mock.AnythingOfType("string")).Return(int64(5001), nil).Once()
		// Expect GetIssueComment to fetch the comment for comparison
		mockClient.On("GetIssueComment", int64(5001)).Return(
			&github.IssueComment{ID: github.Ptr(int64(5001)), Body: github.Ptr("Old comment body")}, // Return some body to trigger update check
			nil,
		).Once()
		// Assume base doesn't need update: UpdatePullRequestBase NOT called
		// Expect comment update for feature-a's PR (comment ID 5001)
		expectedBody101 := "**Stack Overview:**\n\n* **#101**  👈\n* **#102** \n* `main` (base)\n\n<!-- socle-stack-overview -->\n"
		mockClient.On("UpdateComment", int64(5001), mock.MatchedBy(func(body string) bool {
			return body == expectedBody101
		})).Return(
			&github.IssueComment{ID: github.Ptr(int64(5001))}, nil,
		).Once()

		// Expectations for feature-b (create path)
		// Assume config check might happen, return not found
		mockClient.On("GetPullRequest", mock.AnythingOfType("int")).Return(nil, git.ErrConfigNotFound).Maybe() // Need to allow this check if it happens before create check
		// Expect PR creation for feature-b
		mockClient.On("CreatePullRequest", "feature-b", "feature-a", "feat: commit on feature-b", "Test Body B", false).Return(
			&github.PullRequest{Number: github.Ptr(102), HTMLURL: github.Ptr("url-b"), Title: github.Ptr("feat: commit on feature-b")}, nil,
		).Once()
		// Expect comment creation for feature-b's PR
		mockClient.On("FindCommentWithMarker", 102, mock.AnythingOfType("string")).Return(int64(0), nil).Once()
		expectedBody102 := "**Stack Overview:**\n\n* **#101** \n* **#102**  👈\n* `main` (base)\n\n<!-- socle-stack-overview -->\n"
		mockClient.On("CreateComment", 102, mock.MatchedBy(func(body string) bool {
			return body == expectedBody102
		})).Return(
			&github.IssueComment{ID: github.Ptr(int64(5002))}, nil,
		).Once()
		// --- End Mock Setup ---

		// Action: Run 'so submit' with test flags specific to feature-b's creation
		err := runSoCommand(t, "submit",
			"--no-push",
			"--no-draft",
			// Provide details expected for feature-b
			"--test-title=feat: commit on feature-b",
			"--test-body=Test Body B",
		)

		// Assertions
		require.NoError(t, err)
		mockClient.AssertExpectations(t) // Verify calls

		// Check Git Config was written/updated correctly
		prNumA, _ := git.GetGitConfig("branch.feature-a.socle-pr-number")
		prNumB, _ := git.GetGitConfig("branch.feature-b.socle-pr-number")
		commentIdA, _ := git.GetGitConfig("branch.feature-a.socle-comment-id")
		commentIdB, _ := git.GetGitConfig("branch.feature-b.socle-comment-id")

		assert.Equal(t, "101", prNumA, "feature-a PR# should still be 101")
		assert.Equal(t, "102", prNumB, "feature-b PR# should be 102")
		assert.Equal(t, "5001", commentIdA, "feature-a comment ID should still be 5001") // Assuming update used same ID
		assert.Equal(t, "5002", commentIdB, "feature-b comment ID should be 5002")
	})
}
