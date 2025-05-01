package gh

import (
	"fmt"
	"net/http" // Import net/http for status code checking

	"github.com/google/go-github/v71/github" // Use correct version
)

// Define constants for PR statuses
const (
	PRStatusUnknown       = "Unknown"
	PRStatusNotSubmitted  = "Not Submitted" // Indicates no PR number was provided
	PRStatusConfigError   = "Config Error"
	PRStatusInvalidNumber = "Invalid PR #"
	PRStatusAPIError      = "API Error"
	PRStatusNotFound      = "Not Found" // GH 404 specifically
	PRStatusOpen          = "Open"
	PRStatusDraft         = "Draft"
	PRStatusMerged        = "Merged"
	PRStatusClosed        = "Closed"
)

// GetPullRequestStatus fetches a PR and returns its semantic status and URL.
// It assumes the gh.Client is valid and prNumber > 0.
func (c *Client) GetPullRequestStatus(prNumber int) (status string, prURL string, err error) {
	// GetPullRequest is already defined in client.go
	pr, err := c.GetPullRequest(prNumber)
	if err != nil {
		// Check for specific 404 Not Found error
		var ghErr *github.ErrorResponse
		if As(err, &ghErr) && ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusNotFound {
			// It's specifically a 404, return NotFound status, nil error
			return PRStatusNotFound, "", nil
		}
		// Any other API error
		return PRStatusAPIError, "", err // Return the original error for logging
	}

	// Successfully fetched PR, determine status
	url := pr.GetHTMLURL() // Get URL regardless of state

	if pr.GetMerged() {
		return PRStatusMerged, url, nil
	}
	if pr.GetState() == "closed" { // State() returns "open" or "closed"
		return PRStatusClosed, url, nil
	}
	if pr.GetDraft() { // Draft is only relevant if not merged/closed
		return PRStatusDraft, url, nil
	}
	if pr.GetState() == "open" { // Explicitly check for open
		return PRStatusOpen, url, nil
	}

	// Should be unreachable if state is always open/closed
	return PRStatusUnknown, url, fmt.Errorf("unknown PR state for #%d: %s", prNumber, pr.GetState())
}

// Helper like errors.As needed because go-github errors might not directly implement standard interfaces easily
// This is a common pattern when dealing with complex error types from libraries.
func As(err error, target any) bool {
	if err == nil {
		return false
	}
	// This part might need adjustment depending on how go-github wraps errors.
	// For now, a simple type assertion check:
	_, ok := err.(*github.ErrorResponse) // Check if it IS the target type directly
	if ok {
		// If direct match, attempt to assign (this part is tricky without concrete example)
		// For simplicity, we just return true if the type matches.
		// A more robust implementation would use reflection if needed.
		targetPtr, okTarget := target.(**github.ErrorResponse)
		if okTarget {
			*targetPtr = err.(*github.ErrorResponse)
		}
		return true
	}
	// Could add more sophisticated unwrapping here if needed later using errors.Unwrap
	return false
}
