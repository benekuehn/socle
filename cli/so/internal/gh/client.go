package gh

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/benekuehn/socle/cli/so/gitutils"
	"github.com/google/go-github/v71/github"
	"golang.org/x/oauth2"
)

// Client wraps the go-github client.
type Client struct {
	gh    *github.Client
	Owner string
	Repo  string
	Ctx   context.Context // Background context for requests
}

type ClientInterface interface {
	GetPullRequest(number int) (*github.PullRequest, error)
	CreatePullRequest(head, base, title, body string, isDraft bool) (*github.PullRequest, error)
	UpdatePullRequestBase(number int, newBase string) (*github.PullRequest, error)
	CreateComment(issueNumber int, body string) (*github.IssueComment, error)
	UpdateComment(commentID int64, body string) (*github.IssueComment, error)
	FindCommentWithMarker(issueNumber int, marker string) (commentID int64, err error)
}

var _ ClientInterface = (*Client)(nil)

// NewClient creates a new GitHub client using GITHUB_TOKEN.
func NewClient(ctx context.Context, owner, repo string) (*Client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	ghUsed := false // Track if we used gh token

	if token == "" {
		// GITHUB_TOKEN not set, try fetching from 'gh' CLI
		slog.Debug("GITHUB_TOKEN not set. Checking 'gh' CLI for authentication...")

		// Check if 'gh' command exists first
		ghPath, errLookPath := exec.LookPath("gh")
		if errLookPath != nil {
			// 'gh' CLI not found in PATH
			return nil, fmt.Errorf("authentication failed: GITHUB_TOKEN not set and 'gh' CLI not found in PATH. Please set GITHUB_TOKEN or install and authenticate GitHub CLI ('gh auth login')")
		}
		slog.Debug("Found 'gh' CLI. Attempting to fetch token...", "ghPath", ghPath)

		ghToken, err := gitutils.RunExternalCommand("gh", "auth", "token")
		if err != nil {
			desc := "Authentication failed: GITHUB_TOKEN environment variable not set "
			desc += "AND failed to get token from 'gh' CLI. "
			desc += "Please either set GITHUB_TOKEN or ensure 'gh auth login' is complete."
			return nil, fmt.Errorf("%s\ngh command error: %w", desc, err)
		}
		if ghToken == "" {
			// gh command ran but returned empty token
			return nil, fmt.Errorf("authentication failed: GITHUB_TOKEN not set and 'gh auth token' returned empty. Please run 'gh auth login' or set GITHUB_TOKEN")
		}
		slog.Debug("Successfully retrieved token using 'gh auth token'.")
		token = strings.TrimSpace(ghToken) // Use the token from gh
		ghUsed = true
	}

	if !ghUsed {
		slog.Debug("Using GITHUB_TOKEN for authentication.")
	}

	// Use the determined token (either from ENV or gh)
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	httpClientWithTimeout := &http.Client{
		Transport: tc.Transport,
		Timeout:   15 * time.Second,
	}
	ghClient := github.NewClient(httpClientWithTimeout)

	// Optional: Verify token works (consider adding this later if needed)
	// _, _, errVerify := ghClient.Users.Get(ctx, "") ...

	return &Client{gh: ghClient, Owner: owner, Repo: repo, Ctx: ctx}, nil
}

// GetPullRequest retrieves a specific PR by number.
func (c *Client) GetPullRequest(number int) (*github.PullRequest, error) {
	pr, _, err := c.gh.PullRequests.Get(c.Ctx, c.Owner, c.Repo, number)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok && ghErr.Response.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("pull request #%d not found", number)
		}
		return nil, fmt.Errorf("failed to get pull request #%d: %w", number, err)
	}
	return pr, nil
}

// CreatePullRequest creates a new pull request.
func (c *Client) CreatePullRequest(head, base, title, body string, isDraft bool) (*github.PullRequest, error) {
	newPR := &github.NewPullRequest{
		Title:               github.Ptr(title),
		Head:                github.Ptr(head), // Assuming owner prefix is not needed for branches in same repo
		Base:                github.Ptr(base),
		Body:                github.Ptr(body),
		Draft:               github.Ptr(isDraft),
		MaintainerCanModify: github.Ptr(true), // Sensible default
	}

	pr, _, err := c.gh.PullRequests.Create(c.Ctx, c.Owner, c.Repo, newPR)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request (%s -> %s): %w", head, base, err)
	}
	return pr, nil
}

// UpdatePullRequestBase changes the base branch of an existing PR.
func (c *Client) UpdatePullRequestBase(number int, newBase string) (*github.PullRequest, error) {
	update := &github.PullRequest{
		Base: &github.PullRequestBranch{Ref: github.Ptr(newBase)},
	}
	pr, _, err := c.gh.PullRequests.Edit(c.Ctx, c.Owner, c.Repo, number, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update base for pull request #%d to '%s': %w", number, newBase, err)
	}
	return pr, nil
}

// CreateComment adds a new comment to an issue/PR.
func (c *Client) CreateComment(issueNumber int, body string) (*github.IssueComment, error) {
	comment := &github.IssueComment{
		Body: github.Ptr(body),
	}
	newComment, _, err := c.gh.Issues.CreateComment(c.Ctx, c.Owner, c.Repo, issueNumber, comment)
	if err != nil {
		return nil, fmt.Errorf("failed to create comment on issue/PR #%d: %w", issueNumber, err)
	}
	return newComment, nil
}

// UpdateComment edits an existing issue/PR comment.
func (c *Client) UpdateComment(commentID int64, body string) (*github.IssueComment, error) {
	comment := &github.IssueComment{
		Body: github.Ptr(body),
	}
	updatedComment, _, err := c.gh.Issues.EditComment(c.Ctx, c.Owner, c.Repo, commentID, comment)
	if err != nil {
		// Check if comment was deleted (returns 404 Not Found)
		if ghErr, ok := err.(*github.ErrorResponse); ok && ghErr.Response.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("comment ID %d not found (deleted?): %w", commentID, err) // Specific error for not found
		}
		return nil, fmt.Errorf("failed to update comment ID %d: %w", commentID, err)
	}
	return updatedComment, nil
}

func (c *Client) FindCommentWithMarker(issueNumber int, marker string) (commentID int64, err error) {
	// GitHub API typically paginates results. We need to handle pagination
	// to ensure we check all comments.
	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 50}, // Get 100 per page
	}
	var foundComment *github.IssueComment = nil

	for {
		comments, resp, errList := c.gh.Issues.ListComments(c.Ctx, c.Owner, c.Repo, issueNumber, opt)
		if errList != nil {
			// Handle potential rate limiting or other API errors
			return 0, fmt.Errorf("failed to list comments for PR #%d: %w", issueNumber, errList)
		}

		// Search for the marker in the current page of comments
		for _, comment := range comments {
			if comment.Body != nil && strings.Contains(*comment.Body, marker) {
				foundComment = comment
				break // Found it, exit inner loop
			}
		}

		if foundComment != nil {
			break // Found it, exit outer loop
		}

		// Check if there are more pages
		if resp.NextPage == 0 {
			break // No more pages
		}
		opt.Page = resp.NextPage // Move to the next page
	}

	if foundComment != nil {
		return foundComment.GetID(), nil // Return the found ID
	}

	// Marker not found in any comment
	return 0, nil // Return 0, nil error signifies "not found"
}
