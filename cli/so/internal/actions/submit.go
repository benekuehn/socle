package actions

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
	"github.com/benekuehn/socle/cli/so/gitutils"
	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/ui"
	"github.com/google/go-github/v71/github"
	"github.com/spf13/cobra" // Still needed for cmd access for prompts
)

// SubmitBranchOptions holds configuration for the SubmitBranch action.
type SubmitBranchOptions struct {
	IsDraft               bool
	TestSubmitTitle       string
	TestSubmitBody        string
	TestSubmitEditConfirm bool
	// Add other relevant flags/configs if needed, e.g., ForcePush
}

// ErrSubmitCancelled indicates the user cancelled the operation during a prompt.
var ErrSubmitCancelled = errors.New("submit cancelled by user")

// SubmitBranch encapsulates the core logic for ensuring a PR exists and is up-to-date for a branch.
// It checks local config, interacts with the GitHub API, checks diffs, prompts user (currently via cmd),
// updates/creates PRs, and updates local config.
// Returns the final PR state (or nil if skipped) and an error (including ErrSubmitCancelled).
func SubmitBranch(ctx context.Context, ghClient gh.ClientInterface, cmd *cobra.Command, branch, parent string, opts SubmitBranchOptions) (*github.PullRequest, error) {
	slog.Debug("Executing SubmitBranch action", "branch", branch, "parent", parent)

	// 1. Check for existing PR via stored number
	prNumber, configReadErr := gitutils.GetStoredPRNumber(branch)
	if configReadErr != nil {
		// Log actual config read errors, but don't stop the process
		// TODO: Return warnings properly instead of printing here?
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", ui.Colors.WarningStyle.Render(fmt.Sprintf("  Warning: Failed to read stored PR number config for branch '%s': %v. Will attempt to create/find PR.", branch, configReadErr)))
		prNumber = 0 // Ensure we proceed to create/find logic
	}

	var finalPR *github.PullRequest

	// 2. Try to Update Existing PR if number was found
	if prNumber > 0 {
		updatedPR, errUpdate := updateExistingPRAction(ctx, ghClient, cmd, prNumber, branch, parent)
		if errUpdate != nil {
			// Fatal error during update attempt
			return nil, fmt.Errorf("failed trying to update PR #%d: %w", prNumber, errUpdate)
		}

		if updatedPR == nil {
			// PR number was stored, but PR didn't exist on GitHub (404). Clear stored number.
			fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", ui.Colors.WarningStyle.Render(fmt.Sprintf("  Warning: Stored PR #%d not found on GitHub. Clearing stored number.", prNumber)))
			if unsetErr := gitutils.UnsetStoredPRNumber(branch); unsetErr != nil {
				// TODO: Return warnings properly
				fmt.Fprintf(cmd.ErrOrStderr(), "%s", ui.Colors.FailureStyle.Render(fmt.Sprintf("  CRITICAL WARNING: Failed to clear stale PR number %d locally for branch '%s': %v\n", prNumber, branch, unsetErr)))
			}
			prNumber = 0 // Reset prNumber so we attempt creation below
		} else {
			// PR exists and was potentially updated.
			finalPR = updatedPR
			fmt.Printf("  Verified/Updated PR #%d: %s\n", finalPR.GetNumber(), finalPR.GetHTMLURL())
			// Ensure number is stored correctly
			if errSet := gitutils.SetStoredPRNumber(branch, finalPR.GetNumber()); errSet != nil {
				// TODO: Return warnings properly
				fmt.Fprintf(cmd.ErrOrStderr(), "%s", ui.Colors.FailureStyle.Render(fmt.Sprintf("  CRITICAL WARNING: Failed to store PR number %d locally after update for branch '%s': %v\n", finalPR.GetNumber(), branch, errSet)))
			}
		}
	}

	// 3. If we don't have a PR yet, try creating one.
	if finalPR == nil {
		slog.Debug("No valid existing PR found, attempting creation...", "branch", branch)
		createdPR, errCreate := createNewPRAction(ctx, ghClient, cmd, branch, parent, opts)
		if errCreate != nil {
			// Check for cancellation or other fatal error from create helper
			return nil, errCreate
		}

		if createdPR == nil {
			// createNewPRAction returns nil, nil if skipped (e.g., no diff)
			slog.Debug("PR creation skipped by createNewPRAction.", "branch", branch)
			return nil, nil // Indicate skipped
		} else {
			// PR successfully created.
			finalPR = createdPR
			// Store the new PR number
			if errSet := gitutils.SetStoredPRNumber(branch, finalPR.GetNumber()); errSet != nil {
				// TODO: Return warnings properly
				fmt.Fprintf(cmd.ErrOrStderr(), "%s", ui.Colors.FailureStyle.Render(fmt.Sprintf("  CRITICAL WARNING: Failed to store new PR number %d locally for branch '%s': %v\n", finalPR.GetNumber(), branch, errSet)))
				fmt.Fprint(cmd.ErrOrStderr(), ui.Colors.FailureStyle.Render("  Future updates to this PR via 'socle submit' may fail or create duplicates!\n"))
			}
		}
	}

	// 4. Return final PR state
	return finalPR, nil
}

// --- Helpers internal to the actions package ---

// updateExistingPRAction tries to fetch and potentially update the base of an existing PR.
// Corresponds to former updateExistingPRHelper in cmd/submit.go
func updateExistingPRAction(ctx context.Context, ghClient gh.ClientInterface, cmd *cobra.Command, prNumber int, branch, parent string) (*github.PullRequest, error) {
	// (Logic is identical to the former updateExistingPRHelper)
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

// createNewPRAction handles the creation of a new PR after checking for diffs.
// Corresponds to former createNewPRHelper in cmd/submit.go
func createNewPRAction(ctx context.Context, ghClient gh.ClientInterface, cmd *cobra.Command, branch, parent string, opts SubmitBranchOptions) (*github.PullRequest, error) {
	// (Logic is identical to the former createNewPRHelper)
	// --- Check for Diff First ---
	fmt.Printf("  Checking for differences between '%s' and '%s'...\n", parent, branch)
	hasDiff, errDiff := gitutils.HasDiff(parent, branch)
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

	// --- Get Title/Body from User ---
	title, body, errPrompt := promptForPRDetailsAction(cmd, branch, parent, opts)
	if errPrompt != nil {
		return nil, errPrompt // Includes cancellation error
	}

	// --- Create PR via API ---
	draftStatus := map[bool]string{true: "Draft", false: "Ready"}[opts.IsDraft]
	fmt.Printf("  Submitting %s PR for '%s' -> '%s'...\n", draftStatus, branch, parent)
	slog.Debug("Creating PR via API", "branch", branch, "parent", parent, "title", title, "isDraft", opts.IsDraft)
	newPR, errCreate := ghClient.CreatePullRequest(branch, parent, title, body, opts.IsDraft)
	if errCreate != nil {
		return nil, fmt.Errorf("github API error creating pull request: %w", errCreate)
	}

	// --- Success ---
	fmt.Println(ui.Colors.SuccessStyle.Render(
		fmt.Sprintf("  Successfully created %s PR #%d: %s", draftStatus, newPR.GetNumber(), newPR.GetHTMLURL()),
	))
	return newPR, nil
}

// promptForPRDetailsAction prompts the user for PR title and body using defaults.
// Corresponds to former promptForPRDetails in cmd/submit.go
func promptForPRDetailsAction(cmd *cobra.Command, branch, parent string, opts SubmitBranchOptions) (title, body string, err error) {
	// (Logic is identical to the former promptForPRDetails)
	var surveyErr error
	title = ""
	defaultTitle := ""
	firstSubject, errSubject := gitutils.GetFirstCommitSubject(parent, branch)
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
		surveyErr = survey.AskOne(titlePrompt, &title, survey.WithValidator(survey.Required), survey.WithStdio(os.Stdin, os.Stdout, os.Stderr))
		if surveyErr != nil {
			return "", "", handleSurveyInterruptAction(surveyErr, "Submit cancelled during title entry.")
		}
	}
	body = ""
	if opts.TestSubmitBody != "" {
		slog.Debug("Using body from test flag", "testBody", opts.TestSubmitBody)
		body = opts.TestSubmitBody
	} else {
		templateContent, errTpl := gitutils.FindAndReadPRTemplate()
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
				return "", "", handleSurveyInterruptAction(surveyErr, "Submit cancelled during edit confirmation.")
			}
		}
		if editBody {
			editorPrompt := &survey.Editor{Message: "Pull Request Body (Markdown):", FileName: "*.md", Default: templateContent, HideDefault: false}
			surveyErr = survey.AskOne(editorPrompt, &body, survey.WithStdio(os.Stdin, os.Stdout, os.Stderr))
			if surveyErr != nil {
				return "", "", handleSurveyInterruptAction(surveyErr, "Submit cancelled during body editing.")
			}
		} else {
			body = templateContent
		}
	}
	return title, body, nil
}

// handleSurveyInterruptAction checks for survey's interrupt error.
// Corresponds to former handleSurveyInterrupt in cmd/submit.go
func handleSurveyInterruptAction(err error, message string) error {
	if err == terminal.InterruptErr {
		fmt.Println(ui.Colors.WarningStyle.Render(message))
		return ErrSubmitCancelled // Return specific error type for actions
	}
	if err == io.EOF {
		return fmt.Errorf("prompt failed: %w (received io.EOF, potentially non-interactive environment?)", err)
	}
	return fmt.Errorf("prompt failed: %w", err) // Return other survey errors
}
