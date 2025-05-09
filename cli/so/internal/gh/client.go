package gh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	cmdexec "github.com/benekuehn/socle/cli/so/internal/exec"
	"github.com/google/go-github/v71/github"
	"golang.org/x/oauth2"
)

const (
	cacheDirName  = "socle"
	cacheFileName = "gh_token.json"
	tokenCacheTTL = 1 * time.Hour
)

// CachedGhToken stores the GitHub token and its expiry time.
type CachedGhToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

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
	GetIssueComment(commentID int64) (*github.IssueComment, error)
}

var _ ClientInterface = (*Client)(nil)

func getCacheFilePath() (string, error) {
	usrCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user cache directory: %w", err)
	}
	return filepath.Join(usrCacheDir, cacheDirName, cacheFileName), nil
}

func loadTokenFromCache(filePath string) (*CachedGhToken, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Debug("Cache file does not exist.", "path", filePath)
		return nil, nil // Not an error, just no cache
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file '%s': %w", filePath, err)
	}

	if len(data) == 0 {
		slog.Debug("Cache file is empty.", "path", filePath)
		return nil, nil // Not an error, just empty cache
	}

	var cachedToken CachedGhToken
	if err := json.Unmarshal(data, &cachedToken); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached token from '%s': %w", filePath, err)
	}

	if time.Now().After(cachedToken.ExpiresAt) {
		slog.Debug("Cached token is expired.", "path", filePath, "expires_at", cachedToken.ExpiresAt)
		return nil, nil // Expired, treat as no cache
	}

	slog.Debug("Successfully loaded valid token from cache.", "path", filePath, "expires_at", cachedToken.ExpiresAt)
	return &cachedToken, nil
}

func saveTokenToCache(filePath string, token string, ttl time.Duration) error {
	cacheDir := filepath.Dir(filePath)
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory '%s': %w", cacheDir, err)
	}

	tokenData := CachedGhToken{
		Token:     token,
		ExpiresAt: time.Now().Add(ttl),
	}

	data, err := json.MarshalIndent(tokenData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token for cache: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token to cache file '%s': %w", filePath, err)
	}
	slog.Debug("Successfully saved token to cache.", "path", filePath, "expires_at", tokenData.ExpiresAt)
	return nil
}

// NewClient creates a new GitHub client.
// It prioritizes GITHUB_TOKEN env var, then a cached token from 'gh auth token',
// then a fresh 'gh auth token' call if no valid cache.
func NewClient(ctx context.Context, owner, repo string) (*Client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	authMethod := "GITHUB_TOKEN"

	if token == "" {
		authMethod = "gh CLI (cached)"
		cacheFilePath, err := getCacheFilePath()
		if err != nil {
			slog.Warn("Failed to determine cache file path. Proceeding without cache.", "error", err)
			// Proceed without cache, will try gh auth token directly.
		} else {
			cachedToken, errLoad := loadTokenFromCache(cacheFilePath)
			if errLoad != nil {
				slog.Warn("Failed to load token from cache. Proceeding to fetch fresh token.", "path", cacheFilePath, "error", errLoad)
				// Invalidate bad cache file by attempting to remove it
				if errRemove := os.Remove(cacheFilePath); errRemove != nil && !errors.Is(errRemove, fs.ErrNotExist) {
					slog.Warn("Failed to remove corrupted cache file.", "path", cacheFilePath, "error", errRemove)
				}
			}
			if cachedToken != nil {
				token = cachedToken.Token
			}
		}

		if token == "" { // Still no token (no GITHUB_TOKEN, no valid cache)
			authMethod = "gh CLI (live)"
			slog.Debug("GITHUB_TOKEN not set and no valid cached token. Checking 'gh' CLI for authentication...")

			ghPath, errLookPath := exec.LookPath("gh")
			if errLookPath != nil {
				return nil, fmt.Errorf("authentication failed: GITHUB_TOKEN not set, no cached token, and 'gh' CLI not found in PATH. Please set GITHUB_TOKEN or install and authenticate GitHub CLI ('gh auth login')")
			}
			slog.Debug("Found 'gh' CLI. Attempting to fetch token...", "ghPath", ghPath)

			// Check if gh cli is installed by trying to run 'gh --version'
			// This check might be redundant if LookPath succeeded and 'gh auth token' works,
			// but keeping for robustness.
			_, errVersion := cmdexec.RunExternalCommand("gh", "--version")
			if errVersion != nil {
				return nil, fmt.Errorf("gh cli not installed or not found in PATH (despite LookPath success): %w. Please run 'gh auth login' or set GITHUB_TOKEN", errVersion)
			}

			ghToken, errGhAuth := cmdexec.RunExternalCommand("gh", "auth", "token")
			if errGhAuth != nil {
				return nil, fmt.Errorf("error getting token via 'gh auth token': %w. Please run 'gh auth login' or set GITHUB_TOKEN", errGhAuth)
			}
			if ghToken == "" {
				return nil, fmt.Errorf("authentication failed: GITHUB_TOKEN not set, no cache, and 'gh auth token' returned empty. Please run 'gh auth login' or set GITHUB_TOKEN")
			}

			token = strings.TrimSpace(ghToken)
			slog.Debug("Successfully retrieved token using 'gh auth token'.")

			if cacheFilePath != "" { // If we determined a cache path earlier
				if errSave := saveTokenToCache(cacheFilePath, token, tokenCacheTTL); errSave != nil {
					slog.Warn("Failed to save fetched token to cache.", "path", cacheFilePath, "error", errSave)
				}
			}
		}
	}

	slog.Debug("Using token for GitHub client.", "auth_method", authMethod)

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	httpClientWithTimeout := &http.Client{
		Transport: tc.Transport,
		Timeout:   15 * time.Second, // Consider making timeout configurable
	}
	ghClient := github.NewClient(httpClientWithTimeout)

	// Optional: Verify token works (e.g., ghClient.Users.Get(ctx, "")).
	// This would add a network call but ensure the token (from env/cache/live) is valid.
	// For now, we proceed optimistically. If API calls fail later, it will be evident.

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

// GetIssueComment retrieves a specific issue/PR comment by its ID.
func (c *Client) GetIssueComment(commentID int64) (*github.IssueComment, error) {
	comment, _, err := c.gh.Issues.GetComment(c.Ctx, c.Owner, c.Repo, commentID)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok && ghErr.Response.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("comment ID %d not found", commentID)
		}
		return nil, fmt.Errorf("failed to get comment ID %d: %w", commentID, err)
	}
	return comment, nil
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
