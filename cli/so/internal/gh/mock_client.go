package gh

import (
	"fmt"

	"github.com/google/go-github/v71/github"
	"github.com/stretchr/testify/mock"
)

// MockClient implements the ClientInterface for testing
type MockClient struct {
	mock.Mock   // Embed testify mock object
	PRStatuses  map[int]string
	PRNumbers   map[string]int
	CounterChan chan string // Channel to receive operation names
}

// NewMockClient creates a new MockClient
func NewMockClient() *MockClient {
	return &MockClient{
		PRStatuses:  make(map[int]string),
		PRNumbers:   make(map[string]int),
		CounterChan: make(chan string, 100), // Buffer for counting operations
	}
}

// GetPullRequestStatus returns a simulated PR status
func (c *MockClient) GetPullRequestStatus(prNumber int) (status string, prURL string, err error) {
	// Count the operation
	if c.CounterChan != nil {
		c.CounterChan <- "GetPullRequestStatus"
	}
	Counter.Increment("GetPullRequestStatus")

	// If status is predefined, return it
	if status, ok := c.PRStatuses[prNumber]; ok {
		return status, fmt.Sprintf("https://github.com/mock/mock/pull/%d", prNumber), nil
	}

	// Default status
	return PRStatusNotFound, "", nil
}

// GetPullRequest simulates retrieving a PR
func (c *MockClient) GetPullRequest(number int) (*github.PullRequest, error) {
	// Count the operation
	if c.CounterChan != nil {
		c.CounterChan <- "GetPullRequest"
	}
	Counter.Increment("GetPullRequest")

	args := c.Called(number)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.PullRequest), args.Error(1)
}

// CreatePullRequest simulates creating a PR
func (c *MockClient) CreatePullRequest(head, base, title, body string, isDraft bool) (*github.PullRequest, error) {
	// Count the operation
	if c.CounterChan != nil {
		c.CounterChan <- "CreatePullRequest"
	}
	Counter.Increment("CreatePullRequest")

	args := c.Called(head, base, title, body, isDraft)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.PullRequest), args.Error(1)
}

// UpdatePullRequestBase simulates updating a PR's base branch
func (c *MockClient) UpdatePullRequestBase(number int, newBase string) (*github.PullRequest, error) {
	// Count the operation
	if c.CounterChan != nil {
		c.CounterChan <- "UpdatePullRequestBase"
	}
	Counter.Increment("UpdatePullRequestBase")

	args := c.Called(number, newBase)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.PullRequest), args.Error(1)
}

// CreateComment simulates creating a comment
func (c *MockClient) CreateComment(issueNumber int, body string) (*github.IssueComment, error) {
	// Count the operation
	if c.CounterChan != nil {
		c.CounterChan <- "CreateComment"
	}
	Counter.Increment("CreateComment")

	args := c.Called(issueNumber, body)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.IssueComment), args.Error(1)
}

// UpdateComment simulates updating a comment
func (c *MockClient) UpdateComment(commentID int64, body string) (*github.IssueComment, error) {
	// Count the operation
	if c.CounterChan != nil {
		c.CounterChan <- "UpdateComment"
	}
	Counter.Increment("UpdateComment")

	args := c.Called(commentID, body)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.IssueComment), args.Error(1)
}

// FindCommentWithMarker simulates finding a comment with a specific marker
func (c *MockClient) FindCommentWithMarker(issueNumber int, marker string) (commentID int64, err error) {
	// Count the operation
	if c.CounterChan != nil {
		c.CounterChan <- "FindCommentWithMarker"
	}
	Counter.Increment("FindCommentWithMarker")

	args := c.Called(issueNumber, marker)
	return args.Get(0).(int64), args.Error(1)
}

// GetIssueComment simulates retrieving a comment
func (c *MockClient) GetIssueComment(commentID int64) (*github.IssueComment, error) {
	// Count the operation
	if c.CounterChan != nil {
		c.CounterChan <- "GetIssueComment"
	}
	Counter.Increment("GetIssueComment")

	args := c.Called(commentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.IssueComment), args.Error(1)
}
