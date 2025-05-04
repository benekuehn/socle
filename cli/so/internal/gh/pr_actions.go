package gh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
	"github.com/google/go-github/v71/github"
	"github.com/spf13/cobra"
)

// SubmitBranchOptions holds configuration for the SubmitBranch action.
type SubmitBranchOptions struct {
	IsDraft               bool
	TestSubmitTitle       string
	TestSubmitBody        string
	TestSubmitEditConfirm bool
}

// ErrSubmitCancelled indicates the user cancelled the operation during a prompt.
// Moved ErrExitSilently to client.go
var ErrSubmitCancelled = errors.New("submit cancelled by user")

// SubmitBranch encapsulates the core logic for ensuring a PR exists and is up-to-date for a branch.
// It checks local config, interacts with the GitHub API, checks diffs, prompts user (currently via cmd),
// updates/creates PRs, and updates local config.
// Returns the final PR state (or nil if skipped) and an error (including ErrSubmitCancelled).
func SubmitBranch(ctx context.Context, ghClient ClientInterface, cmd *cobra.Command, branch, parent string, opts SubmitBranchOptions) (*github.PullRequest, error) {
	slog.Debug("Executing SubmitBranch action", "branch", branch, "parent", parent)

	// 1. Check for existing PR via stored number
	prNumber, configReadErr := git.GetStoredPRNumber(branch)
	if configReadErr != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", ui.Colors.WarningStyle.Render(fmt.Sprintf("  Warning: Failed to read stored PR number config for branch '%s': %v. Will attempt to create/find PR.", branch, configReadErr)))
		prNumber = 0 // Ensure we proceed to create/find logic
	}

	var finalPR *github.PullRequest

	// 2. Try to Update Existing PR if number was found
	if prNumber > 0 {
		// Call renamed helper function
		updatedPR, errUpdate := updateExistingPR(ghClient, prNumber, parent)
		if errUpdate != nil {
			return nil, fmt.Errorf("failed trying to update PR #%d: %w", prNumber, errUpdate)
		}

		if updatedPR == nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", ui.Colors.WarningStyle.Render(fmt.Sprintf("  Warning: Stored PR #%d not found on GitHub. Clearing stored number.", prNumber)))
			if unsetErr := git.UnsetStoredPRNumber(branch); unsetErr != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s", ui.Colors.FailureStyle.Render(fmt.Sprintf("  CRITICAL WARNING: Failed to clear stale PR number %d locally for branch '%s': %v\n", prNumber, branch, unsetErr)))
			}
			prNumber = 0 // Reset prNumber so we attempt creation below
		} else {
			finalPR = updatedPR
			fmt.Printf("  Verified/Updated PR #%d: %s\n", finalPR.GetNumber(), finalPR.GetHTMLURL())
			if errSet := git.SetStoredPRNumber(branch, finalPR.GetNumber()); errSet != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s", ui.Colors.FailureStyle.Render(fmt.Sprintf("  CRITICAL WARNING: Failed to store PR number %d locally after update for branch '%s': %v\n", finalPR.GetNumber(), branch, errSet)))
			}
		}
	}

	// 3. If we don't have a PR yet, try creating one.
	if finalPR == nil {
		slog.Debug("No valid existing PR found, attempting creation...", "branch", branch)
		// Call renamed helper function
		createdPR, errCreate := createNewPR(ghClient, cmd, branch, parent, opts)
		if errCreate != nil {
			return nil, errCreate
		}

		if createdPR == nil {
			slog.Debug("PR creation skipped by createNewPR.", "branch", branch)
			return nil, nil // Indicate skipped
		} else {
			finalPR = createdPR
			if errSet := git.SetStoredPRNumber(branch, finalPR.GetNumber()); errSet != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s", ui.Colors.FailureStyle.Render(fmt.Sprintf("  CRITICAL WARNING: Failed to store new PR number %d locally for branch '%s': %v\n", finalPR.GetNumber(), branch, errSet)))
				fmt.Fprint(cmd.ErrOrStderr(), ui.Colors.FailureStyle.Render("  Future updates to this PR via 'socle submit' may fail or create duplicates!\n"))
			}
		}
	}

	// 4. Return final PR state
	return finalPR, nil
}

// updateExistingPR tries to fetch and potentially update the base of an existing PR.
func updateExistingPR(ghClient ClientInterface, prNumber int, parent string) (*github.PullRequest, error) {
	fmt.Printf("  Verifying existing PR #%d and checking for base update...\n", prNumber)
	existingPR, err := ghClient.GetPullRequest(prNumber)
	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) && ghErr.Response.StatusCode == 404 {
			return nil, nil // Not found is not an error for this helper
		}
		return nil, fmt.Errorf("failed to fetch existing PR #%d: %w", prNumber, err)
	}
	if existingPR.GetBase().GetRef() != parent {
		fmt.Printf("  Updating base branch for PR #%d from '%s' to '%s'...\n", prNumber, existingPR.GetBase().GetRef(), parent)
		updatedPR, errUpdate := ghClient.UpdatePullRequestBase(prNumber, parent)
		if errUpdate != nil {
			return nil, fmt.Errorf("failed to update base for PR #%d: %w", prNumber, errUpdate)
		}
		fmt.Println(ui.Colors.SuccessStyle.Render("  PR base updated."))
		return updatedPR, nil
	} else {
		fmt.Println("  PR base branch is already correct.")
		return existingPR, nil
	}
}

// createNewPR handles the creation of a new PR after checking for diffs.
func createNewPR(ghClient ClientInterface, cmd *cobra.Command, branch, parent string, opts SubmitBranchOptions) (*github.PullRequest, error) {
	fmt.Printf("  Checking for differences between '%s' and '%s'...\n", parent, branch)
	hasDiff, errDiff := git.HasDiff(parent, branch)
	if errDiff != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", ui.Colors.FailureStyle.Render(fmt.Sprintf("  ERROR: Failed to check for differences: %v", errDiff)))
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", ui.Colors.WarningStyle.Render(fmt.Sprintf("  Skipping PR processing for branch '%s' due to diff check error.", branch)))
		return nil, nil // Indicate skip
	}
	if !hasDiff {
		fmt.Println(ui.Colors.InfoStyle.Render(fmt.Sprintf("  No differences found between '%s' and '%s'.", parent, branch)))
		fmt.Println(ui.Colors.InfoStyle.Render(fmt.Sprintf("  GitHub requires changes to create a Pull Request. Skipping PR creation for '%s'.", branch)))
		return nil, nil // Indicate skip
	}
	slog.Debug("Differences found. Proceeding with PR creation details...")

	// Call renamed helper function
	title, body, errPrompt := promptForPRDetails(cmd, branch, parent, opts)
	if errPrompt != nil {
		return nil, errPrompt // Includes cancellation error
	}

	draftStatus := map[bool]string{true: "Draft", false: "Ready"}[opts.IsDraft]
	fmt.Printf("  Submitting %s PR for '%s' -> '%s'...\n", draftStatus, branch, parent)
	slog.Debug("Creating PR via API", "branch", branch, "parent", parent, "title", title, "isDraft", opts.IsDraft)
	newPR, errCreate := ghClient.CreatePullRequest(branch, parent, title, body, opts.IsDraft)
	if errCreate != nil {
		return nil, fmt.Errorf("github API error creating pull request: %w", errCreate)
	}

	fmt.Println(ui.Colors.SuccessStyle.Render(
		fmt.Sprintf("  Successfully created %s PR #%d: %s", draftStatus, newPR.GetNumber(), newPR.GetHTMLURL()),
	))
	return newPR, nil
}

// promptForPRDetails prompts the user for PR title and body using defaults.
func promptForPRDetails(cmd *cobra.Command, branch, parent string, opts SubmitBranchOptions) (title, body string, err error) {
	var surveyErr error
	title = ""
	defaultTitle := ""
	firstSubject, errSubject := git.GetFirstCommitSubject(parent, branch)
	if errSubject != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", ui.Colors.WarningStyle.Render(fmt.Sprintf("  Warning: Could not determine first commit subject for default title: %v", errSubject)))
		defaultTitle = strings.ReplaceAll(branch, "-", " ")
	} else if firstSubject == "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", ui.Colors.WarningStyle.Render("  Warning: No unique commits found for default title. Using branch name."))
		defaultTitle = strings.ReplaceAll(branch, "-", " ")
	} else {
		defaultTitle = firstSubject
		fmt.Printf("  Using commit subject for default title: \"%s\"\n", defaultTitle)
	}
	if opts.TestSubmitTitle != "" {
		title = opts.TestSubmitTitle
	} else {
		titlePrompt := &survey.Input{Message: "Pull Request Title:", Default: defaultTitle}
		// Call renamed helper function
		surveyErr = survey.AskOne(titlePrompt, &title, survey.WithValidator(survey.Required), survey.WithStdio(os.Stdin, os.Stdout, os.Stderr))
		if surveyErr != nil {
			return "", "", handleSurveyInterrupt(surveyErr, "Submit cancelled during title entry.")
		}
	}
	body = ""
	if opts.TestSubmitBody != "" {
		slog.Debug("Using body from test flag", "testBody", opts.TestSubmitBody)
		body = opts.TestSubmitBody
	} else {
		templateContent, errTpl := git.FindAndReadPRTemplate()
		if errTpl != nil {
			slog.Warn("Failed to read PR template", "error", errTpl)
			fmt.Fprintln(cmd.ErrOrStderr(), ui.Colors.WarningStyle.Render("  Warning: Could not read PR template: "+errTpl.Error()))
		} else if templateContent != "" {
			fmt.Println("  Found PR template.")
		} else {
			fmt.Println("  No PR template found. Using empty description.")
		}
		editBody := false
		if opts.TestSubmitEditConfirm {
			editBody = true
		} else {
			confirmPrompt := &survey.Confirm{Message: "Edit description before submitting?", Default: templateContent == ""}
			surveyErr = survey.AskOne(confirmPrompt, &editBody, survey.WithStdio(os.Stdin, os.Stdout, os.Stderr))
			if surveyErr != nil {
				return "", "", handleSurveyInterrupt(surveyErr, "Submit cancelled during edit confirmation.")
			}
		}
		if editBody {
			editorPrompt := &survey.Editor{Message: "Pull Request Body (Markdown):", FileName: "*.md", Default: templateContent, HideDefault: false}
			surveyErr = survey.AskOne(editorPrompt, &body, survey.WithStdio(os.Stdin, os.Stdout, os.Stderr))
			if surveyErr != nil {
				return "", "", handleSurveyInterrupt(surveyErr, "Submit cancelled during body editing.")
			}
		} else {
			body = templateContent
		}
	}
	return title, body, nil
}

// handleSurveyInterrupt checks for survey's interrupt error.
func handleSurveyInterrupt(err error, message string) error {
	if err == terminal.InterruptErr {
		fmt.Println(ui.Colors.WarningStyle.Render(message))
		return ErrSubmitCancelled // Return specific error type for actions
	}
	if err == io.EOF {
		return fmt.Errorf("prompt failed: %w (received io.EOF, potentially non-interactive environment?)", err)
	}
	return fmt.Errorf("prompt failed: %w", err)
}

// --- Commenting Logic ---

// EnsureStackComment handles adding or updating the stack overview comment on a given PR.
func EnsureStackComment(ctx context.Context, ghClient ClientInterface, branch string, prNumber int, commentBody string, marker string) error {
	slog.Debug("Executing EnsureStackComment action", "branch", branch, "prNumber", prNumber)
	var accumulatedError error // Collect non-fatal errors

	// 1. Get existing comment ID stored locally
	storedCommentID, configReadErr := git.GetStoredCommentID(branch)
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
			if err := git.SetStoredCommentID(branch, foundCommentID); err != nil {
				critErrMsg := fmt.Sprintf("failed to update stored comment ID for branch '%s': %v", branch, err)
				slog.Error(critErrMsg)
				return fmt.Errorf("%s", critErrMsg)
			}
			storedCommentID = foundCommentID
		} else if storedCommentID == 0 && foundCommentID != 0 {
			slog.Debug("Storing newly found comment ID", "foundCommentID", foundCommentID, "branch", branch)
			if err := git.SetStoredCommentID(branch, foundCommentID); err != nil {
				critErrMsg := fmt.Sprintf("failed to store found comment ID %d for branch '%s': %v", foundCommentID, branch, err)
				slog.Error(critErrMsg)
				return fmt.Errorf("%s", critErrMsg)
			}
			storedCommentID = foundCommentID
		}

		// Update the found comment
		slog.Debug("Updating stack comment", "commentID", foundCommentID, "prNumber", prNumber)
		_, err := ghClient.UpdateComment(foundCommentID, commentBody)
		if err != nil {
			return fmt.Errorf("failed to update stack comment %d on PR #%d: %w", foundCommentID, prNumber, err)
		}
		slog.Debug("Comment updated successfully.")

	} else {
		// --- Comment with marker NOT found ---
		slog.Debug("No existing stack comment found via marker.", "prNumber", prNumber)

		if storedCommentID != 0 {
			warnMsg := fmt.Sprintf("stored comment ID %d found, but no matching comment exists on PR #%d. Clearing stored ID", storedCommentID, prNumber)
			slog.Warn(warnMsg)
			if accumulatedError == nil {
				accumulatedError = fmt.Errorf("%w", errors.New(warnMsg))
			} else {
				accumulatedError = fmt.Errorf("%w; %s", accumulatedError, warnMsg)
			}
			if err := git.UnsetStoredCommentID(branch); err != nil {
				critErrMsg := fmt.Sprintf("failed to clear stale comment ID for branch '%s': %v", branch, err)
				slog.Error(critErrMsg)
				accumulatedError = fmt.Errorf("%w; %s", accumulatedError, critErrMsg)
			}
		}

		// Create new comment
		slog.Debug("Adding stack comment", "prNumber", prNumber)
		newComment, err := ghClient.CreateComment(prNumber, commentBody)
		if err != nil {
			return fmt.Errorf("failed to add stack comment to PR #%d: %w", prNumber, err)
		}
		slog.Debug("Comment added successfully.")

		// Store the new comment ID
		newCommentID := newComment.GetID()
		if err := git.SetStoredCommentID(branch, newCommentID); err != nil {
			critErrMsg := fmt.Sprintf("failed to store new comment ID %d for branch '%s': %v", newCommentID, branch, err)
			slog.Error(critErrMsg)
			return fmt.Errorf("%s", critErrMsg)
		}
		slog.Debug("Stored new comment ID", "commentID", newCommentID)
	}

	return accumulatedError // Return collected non-fatal errors/warnings
}
