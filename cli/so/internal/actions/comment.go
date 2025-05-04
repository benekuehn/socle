package actions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/benekuehn/socle/cli/so/gitutils"
	"github.com/benekuehn/socle/cli/so/internal/gh"
)

// EnsureStackComment handles adding or updating the stack overview comment on a given PR.
// It fetches stored comment IDs, finds existing comments via marker, reconciles,
// and performs API calls to update or create the comment.
// Returns potential errors encountered during the process.
// TODO: Define specific error types for warnings vs critical failures?
func EnsureStackComment(ctx context.Context, ghClient gh.ClientInterface, branch string, prNumber int, commentBody string, marker string) error {
	slog.Debug("Executing EnsureStackComment action", "branch", branch, "prNumber", prNumber)
	var accumulatedError error // Collect non-fatal errors

	// 1. Get existing comment ID stored locally
	storedCommentID, configReadErr := gitutils.GetStoredCommentID(branch)
	if configReadErr != nil {
		warnMsg := fmt.Sprintf("failed to read stored comment ID config for branch '%s': %v", branch, configReadErr)
		slog.Warn(warnMsg)
		accumulatedError = errors.New(warnMsg) // Use errors.New for initial assignment
		storedCommentID = 0
	}

	// 2. Find comment on GitHub using marker
	foundCommentID, findErr := ghClient.FindCommentWithMarker(prNumber, marker)
	if findErr != nil {
		// API error occurred trying to find the comment - treat as fatal for this action
		return fmt.Errorf("failed to search for existing stack comment on PR #%d: %w", prNumber, findErr)
	}

	// 3. Update or Create Comment
	if foundCommentID > 0 {
		// --- Comment with marker found ---
		slog.Debug("Found existing stack comment via marker", "foundCommentID", foundCommentID, "prNumber", prNumber)

		// Reconcile stored ID with found ID
		if storedCommentID != 0 && storedCommentID != foundCommentID {
			warnMsg := fmt.Sprintf("stored comment ID (%d) differs from found comment ID (%d) for PR #%d. Updating stored ID", storedCommentID, foundCommentID, prNumber)
			slog.Warn(warnMsg)
			if accumulatedError == nil {
				accumulatedError = fmt.Errorf("%w", errors.New(warnMsg))
			} else {
				accumulatedError = fmt.Errorf("%w; %s", accumulatedError, warnMsg) // Append warning
			}
			if err := gitutils.SetStoredCommentID(branch, foundCommentID); err != nil {
				critErrMsg := fmt.Sprintf("failed to update stored comment ID for branch '%s': %v", branch, err)
				slog.Error(critErrMsg)
				// Treat failure to update config as critical?
				return fmt.Errorf("%s", critErrMsg) // Use %s format specifier
			}
			storedCommentID = foundCommentID
		} else if storedCommentID == 0 && foundCommentID != 0 {
			// Found a comment but didn't have one stored, store it now
			slog.Debug("Storing newly found comment ID", "foundCommentID", foundCommentID, "branch", branch)
			if err := gitutils.SetStoredCommentID(branch, foundCommentID); err != nil {
				critErrMsg := fmt.Sprintf("failed to store found comment ID %d for branch '%s': %v", foundCommentID, branch, err)
				slog.Error(critErrMsg)
				return fmt.Errorf("%s", critErrMsg) // Use %s format specifier
			}
			storedCommentID = foundCommentID
		}

		// Update the found comment
		// TODO: Optionally, get current comment body and check if update is needed
		slog.Debug("Updating stack comment", "commentID", foundCommentID, "prNumber", prNumber)
		_, err := ghClient.UpdateComment(foundCommentID, commentBody)
		if err != nil {
			// Failed to update - treat as fatal for this action
			return fmt.Errorf("failed to update stack comment %d on PR #%d: %w", foundCommentID, prNumber, err)
		}
		slog.Debug("Comment updated successfully.")

	} else {
		// --- Comment with marker NOT found ---
		slog.Debug("No existing stack comment found via marker.", "prNumber", prNumber)

		if storedCommentID != 0 {
			// We had a stored ID, but no comment found. Clear the stale stored ID.
			warnMsg := fmt.Sprintf("stored comment ID %d found, but no matching comment exists on PR #%d. Clearing stored ID", storedCommentID, prNumber)
			slog.Warn(warnMsg)
			if accumulatedError == nil {
				accumulatedError = fmt.Errorf("%w", errors.New(warnMsg))
			} else {
				accumulatedError = fmt.Errorf("%w; %s", accumulatedError, warnMsg)
			}
			if err := gitutils.UnsetStoredCommentID(branch); err != nil {
				critErrMsg := fmt.Sprintf("failed to clear stale comment ID for branch '%s': %v", branch, err)
				slog.Error(critErrMsg)
				// Don't abort, but definitely record the error
				accumulatedError = fmt.Errorf("%w; %s", accumulatedError, critErrMsg)
			}
		}

		// Create new comment
		slog.Debug("Adding stack comment", "prNumber", prNumber)
		newComment, err := ghClient.CreateComment(prNumber, commentBody)
		if err != nil {
			// Failed to create - treat as fatal for this action
			return fmt.Errorf("failed to add stack comment to PR #%d: %w", prNumber, err)
		}
		slog.Debug("Comment added successfully.")

		// Store the new comment ID
		newCommentID := newComment.GetID()
		if err := gitutils.SetStoredCommentID(branch, newCommentID); err != nil {
			critErrMsg := fmt.Sprintf("failed to store new comment ID %d for branch '%s': %v", newCommentID, branch, err)
			slog.Error(critErrMsg)
			// Treat failure to store critical config as fatal?
			return fmt.Errorf("%s", critErrMsg) // Use %s format specifier
		}
		slog.Debug("Stored new comment ID", "commentID", newCommentID)
	}

	return accumulatedError // Return collected non-fatal errors/warnings
}
