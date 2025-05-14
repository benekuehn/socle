package gh

import (
	"fmt"

	"github.com/google/go-github/v71/github"
)

// MockClient implements the ClientInterface for testing
type MockClient struct {
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

	// Simulate not found error
	if _, ok := c.PRStatuses[number]; !ok {
		return nil, fmt.Errorf("pull request #%d not found", number)
	}

	// Return a mock PR
	return &github.PullRequest{}, nil
}

// CreatePullRequest simulates creating a PR
func (c *MockClient) CreatePullRequest(head, base, title, body string, isDraft bool) (*github.PullRequest, error) {
	// Count the operation
	if c.CounterChan != nil {
		c.CounterChan <- "CreatePullRequest"
	}
	Counter.Increment("CreatePullRequest")

	return &github.PullRequest{}, nil
}

// UpdatePullRequestBase simulates updating a PR's base branch
func (c *MockClient) UpdatePullRequestBase(number int, newBase string) (*github.PullRequest, error) {
	// Count the operation
	if c.CounterChan != nil {
		c.CounterChan <- "UpdatePullRequestBase"
	}
	Counter.Increment("UpdatePullRequestBase")

	return &github.PullRequest{}, nil
}

// CreateComment simulates creating a comment
func (c *MockClient) CreateComment(issueNumber int, body string) (*github.IssueComment, error) {
	// Count the operation
	if c.CounterChan != nil {
		c.CounterChan <- "CreateComment"
	}
	Counter.Increment("CreateComment")

	return &github.IssueComment{}, nil
}

// UpdateComment simulates updating a comment
func (c *MockClient) UpdateComment(commentID int64, body string) (*github.IssueComment, error) {
	// Count the operation
	if c.CounterChan != nil {
		c.CounterChan <- "UpdateComment"
	}
	Counter.Increment("UpdateComment")

	return &github.IssueComment{}, nil
}

// FindCommentWithMarker simulates finding a comment with a specific marker
func (c *MockClient) FindCommentWithMarker(issueNumber int, marker string) (commentID int64, err error) {
	// Count the operation
	if c.CounterChan != nil {
		c.CounterChan <- "FindCommentWithMarker"
	}
	Counter.Increment("FindCommentWithMarker")

	// Always return 0 for simplicity
	return 0, nil
}

// GetIssueComment simulates retrieving a comment
func (c *MockClient) GetIssueComment(commentID int64) (*github.IssueComment, error) {
	// Count the operation
	if c.CounterChan != nil {
		c.CounterChan <- "GetIssueComment"
	}
	Counter.Increment("GetIssueComment")

	return &github.IssueComment{}, nil
}
