// cli/cmd/submit.go
package cmd

import (
	"context" // Need context for GitHub client
	"fmt"
	"os"
	"strings"

	"github.com/benekuehn/socle/cli/so/gitutils"
	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/ui"

	"github.com/AlecAivazis/survey/v2"
	"github.com/google/go-github/v71/github" // Added for comment handling
	"github.com/spf13/cobra"
)

// Flag variables
var (
	submitForcePush bool
	submitNoPush    bool
	submitDraft     = true // Default set in flag definition
)

// Struct to hold PR info needed for comments
type submittedPrInfo struct {
	Number int
	URL    string
	Title  string // Add title for comment rendering
}

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Create or update GitHub Pull Requests for the current stack",
	Long: `Pushes branches in the current stack to the remote ('origin' by default)
and creates or updates corresponding GitHub Pull Requests.

- Requires GITHUB_TOKEN environment variable with 'repo' scope.
- Reads PR templates from .github/ or root directory.
- Creates Draft PRs by default (use --draft=false to override).
- Stores PR numbers locally in '.git/config' for future updates.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background() // Base context for API calls

		// Check if git push/fetch remote exists (e.g., 'origin')
		remoteName := "origin" // TODO: Make configurable?
		remoteURL, err := gitutils.GetRemoteURL(remoteName)
		if err != nil {
			return fmt.Errorf("cannot submit: %w", err)
		}

		// 1. Get Owner/Repo & GitHub Client
		owner, repoName, err := gitutils.ParseOwnerAndRepo(remoteURL)
		if err != nil {
			return fmt.Errorf("cannot parse owner/repo from remote '%s' URL '%s': %w", remoteName, remoteURL, err)
		}
		ghClient, err := gh.NewClient(ctx, owner, repoName)
		if err != nil {
			return fmt.Errorf("failed to create GitHub client: %w", err)
		}
		fmt.Printf("Operating on repository: %s/%s\n", owner, repoName)

		// 2. Get Stack Info
		_, stack, _, err := getCurrentStackInfo()
		if err != nil {
			return err // Error message already formatted
		}
		if len(stack) <= 1 {
			fmt.Println("Current branch is the base or directly on base. Nothing to submit.")
			return nil
		}

		// Map to store PR info for the comment phase
		prInfoMap := make(map[string]submittedPrInfo)

		// 3. Iterate and Process Stack (skip base branch at index 0)
		for i := 1; i < len(stack); i++ {
			branch := stack[i]
			parent := stack[i-1] // Base for the PR

			fmt.Printf("\nProcessing branch: %s (parent: %s)\n",
				ui.Colors.UserInputStyle.Render(branch),
				ui.Colors.UserInputStyle.Render(parent))

			// 3a. Push Branch (unless skipped)
			if !submitNoPush {
				fmt.Printf("Pushing branch '%s' to '%s'...\n", branch, remoteName)
				err := gitutils.PushBranch(branch, remoteName, submitForcePush)
				if err != nil {
					// Allow continuing? Or stop? Let's stop for now.
					return fmt.Errorf("failed to push branch '%s', aborting submit: %w", branch, err)
				}
				fmt.Println(ui.Colors.SuccessStyle.Render("Push successful."))
			} else {
				fmt.Println("Skipping push (--no-push).")
			}

			// 3b. Check for existing PR via stored number
			prNumberKey := fmt.Sprintf("branch.%s.socle-pr-number", branch)
			prNumberStr, err := gitutils.GetGitConfig(prNumberKey)
			prNumber := 0 // 0 indicates no stored PR number
			if err == nil && prNumberStr != "" {
				// Found stored PR number, try parsing
				_, errScan := fmt.Sscan(prNumberStr, &prNumber)
				if errScan != nil {
					fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("Warning: Could not parse stored PR number ('%s') for branch '%s'. Will attempt to create new PR. Error: %v\n"), prNumberStr, branch, errScan)
					prNumber = 0 // Reset on parse error
				}
			} else if err != nil && !strings.Contains(err.Error(), "not found") {
				// Error reading config, other than "not found"
				fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("Warning: Failed to read PR number config for branch '%s'. Will attempt to create new PR. Error: %v\n"), branch, err)
				prNumber = 0
			}
			// Now prNumber holds the stored number, or 0 if none/error

			// Variable to hold current PR object for this branch
			var currentPR *github.PullRequest

			// 3c. Create or Update PR
			if prNumber > 0 {
				// --- Update Existing PR ---
				fmt.Printf("Found existing PR #%d associated with branch '%s'. Checking for updates...\n", prNumber, branch)
				existingPR, err := ghClient.GetPullRequest(prNumber)
				if err != nil {
					// Maybe PR was deleted? Allow creating new one?
					fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("Warning: Failed to fetch existing PR #%d for branch '%s'. Will try to create a new one. Error: %v\n"), prNumber, branch, err)
					// Clear local config so we don't try to update again
					_ = gitutils.UnsetGitConfig(prNumberKey)
					prNumber = 0 // Force creation path
					currentPR = nil
				} else {
					// Check if base needs update
					if existingPR.GetBase().GetRef() != parent {
						fmt.Printf("Updating base branch for PR #%d from '%s' to '%s'...\n", prNumber, existingPR.GetBase().GetRef(), parent)
						updatedPR, errUpdate := ghClient.UpdatePullRequestBase(prNumber, parent)
						if errUpdate != nil {
							// Log error but maybe continue? Or abort? Abort for safety.
							return fmt.Errorf("failed to update base for PR #%d: %w", prNumber, errUpdate)
						}
						currentPR = updatedPR
						fmt.Println(ui.Colors.SuccessStyle.Render("PR base updated."))
					} else {
						fmt.Println("PR base branch is already correct.")
						currentPR = existingPR
					}
					fmt.Printf("PR #%d link: %s\n", prNumber, existingPR.GetHTMLURL())

					// Store PR info for comment phase
					prInfoMap[branch] = submittedPrInfo{
						Number: currentPR.GetNumber(),
						URL:    currentPR.GetHTMLURL(),
						Title:  currentPR.GetTitle(),
					}
					fmt.Printf("DEBUG: Stored PR info for %s: %+v\n", branch, prInfoMap[branch])

					continue // Move to next branch in stack
				}
			}

			// If we reach here, we need to create a new PR
			if prNumber == 0 {
				// --- Create New PR ---
				fmt.Printf("Attempting to create PR for branch '%s' -> '%s'...\n", branch, parent)

				// --- Check for Diff First ---
				hasDiff, errDiff := gitutils.HasDiff(parent, branch)
				if errDiff != nil {
					// Treat inability to check diff as fatal for this branch's processing
					fmt.Fprintf(os.Stderr, ui.Colors.FailureStyle.Render("ERROR: Failed to check for differences between '%s' and '%s': %v\n"), parent, branch, errDiff)
					// Should we abort the whole submit? Or just skip this branch? Let's skip for now.
					fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("Skipping PR processing for branch '%s' due to diff check error.\n"), branch)
					currentPR = nil // Ensure no PR object exists for map storage
					continue        // Skip to the next branch
				}

				if !hasDiff {
					// Inform the user clearly, then skip this branch's PR creation/update
					fmt.Println(ui.Colors.InfoStyle.Render(fmt.Sprintf("  No differences found between '%s' and '%s'.", parent, branch)))
					fmt.Println(ui.Colors.InfoStyle.Render(fmt.Sprintf("  GitHub requires changes to create a Pull Request. Skipping PR creation for '%s'.", branch)))
					currentPR = nil // Ensure no PR object exists for map storage
					continue        // Skip to the next branch in the stack
				}
				// --- Proceed only if diff exists ---
				fmt.Println("  Differences found. Proceeding with PR creation details...")

				// --- Get title ---
				// Attempt to get the first commit subject for the default title
				defaultTitle := ""
				firstSubject, errSubject := gitutils.GetFirstCommitSubject(parent, branch)

				if errSubject != nil {
					// Warn about error getting subject, but proceed with fallback
					fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("Warning: Could not determine first commit subject for default title: %v\n"), errSubject)
					defaultTitle = strings.ReplaceAll(branch, "-", " ") // Fallback to branch name
				} else if firstSubject == "" {
					// No unique commits found on branch relative to parent
					fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("Warning: No unique commits found on branch '%s' relative to '%s'. Using branch name for default title.\n"), branch, parent)
					defaultTitle = strings.ReplaceAll(branch, "-", " ") // Fallback
				} else {
					defaultTitle = firstSubject // Use commit subject!
					fmt.Printf("Using commit subject for default title: \"%s\"\n", defaultTitle)
				}

				// Prompt user for title, using the determined default
				title := ""
				titlePrompt := &survey.Input{
					Message: "Pull Request Title:",
					Default: defaultTitle, // Use determined default
				}
				err = survey.AskOne(titlePrompt, &title, survey.WithValidator(survey.Required), survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
				if err != nil {
					return handleSurveyInterrupt(err, "Submit cancelled.")
				}

				// --- Get body (with template) ---
				body := ""
				templateContent, errTpl := gitutils.FindAndReadPRTemplate()
				if errTpl != nil {
					fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("Warning: Failed to read PR template: %v\n"), errTpl)
				}

				editBody := false // Default to not editing
				if templateContent != "" {
					fmt.Println("Found PR template.")
					prompt := &survey.Confirm{Message: "Edit description before submitting?", Default: false}
					err = survey.AskOne(prompt, &editBody, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
					if err != nil {
						return handleSurveyInterrupt(err, "Submit cancelled.")
					}
				} else {
					// No template, maybe still offer edit? Or just use empty default?
					// Let's default to empty and maybe offer edit later if needed.
					fmt.Println("No PR template found. Using empty description.")
				}

				if editBody {
					// Open editor
					editorPrompt := &survey.Editor{
						Message:       "Pull Request Body (Markdown):",
						FileName:      "*.md",
						Default:       templateContent,
						HideDefault:   templateContent == "",
						AppendDefault: false,
					}
					err = survey.AskOne(editorPrompt, &body, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
					if err != nil {
						return handleSurveyInterrupt(err, "Submit cancelled.")
					}
				} else {
					// Use template content directly (or empty if no template)
					body = templateContent
				}

				// Create PR call
				isDraft := submitDraft // Use flag value
				fmt.Printf("Submitting as %s PR...\n", map[bool]string{true: "Draft", false: "Ready"}[isDraft])
				newPR, err := ghClient.CreatePullRequest(branch, parent, title, body, isDraft)
				if err != nil {
					// Abort on failure to create
					return fmt.Errorf("failed to create pull request for branch '%s': %w", branch, err)
				}
				currentPR = newPR

				// Store the new PR number
				newPrNumberStr := fmt.Sprintf("%d", newPR.GetNumber())
				fmt.Printf("Storing PR number %s for branch '%s'...\n", newPrNumberStr, branch)
				err = gitutils.SetGitConfig(prNumberKey, newPrNumberStr)
				if err != nil {
					// Don't fail the whole process, but warn heavily
					fmt.Fprintf(os.Stderr, ui.Colors.FailureStyle.Render("CRITICAL WARNING: Failed to store PR number %d locally for branch '%s': %v\n"), newPR.GetNumber(), branch, err)
					fmt.Fprint(os.Stderr, ui.Colors.FailureStyle.Render("  Future updates to this PR via 'socle submit' may fail or create duplicates!\n"))
				}

				fmt.Println(ui.Colors.SuccessStyle.Render(
					fmt.Sprintf("Successfully created %s PR #%d: %s", map[bool]string{true: "Draft", false: "Ready"}[isDraft], newPR.GetNumber(), newPR.GetHTMLURL()),
				))
			}

			// --- Store info for comment phase ---
			if currentPR != nil {
				prInfoMap[branch] = submittedPrInfo{
					Number: currentPR.GetNumber(),
					URL:    currentPR.GetHTMLURL(),
					Title:  currentPR.GetTitle(),
				}
				fmt.Printf("DEBUG: Stored PR info for %s: %+v\n", branch, prInfoMap[branch])
			} else {
				// This case now happens if we skipped due to no diff or update failed
				fmt.Printf("DEBUG: No valid PR object for %s after processing (skipped or failed).\n", branch)
			}

		} // End of first loop

		// --- Debug before second loop ---
		fmt.Printf("DEBUG: prInfoMap contains %d entries before comment phase.\n", len(prInfoMap))
		for branch, info := range prInfoMap {
			fmt.Printf("DEBUG: Map entry - branch: %s, PR #%d, URL: %s\n", branch, info.Number, info.URL)
		}
		fmt.Println("\n--- Phase 2: Updating PR comments with stack overview ---")
		if len(prInfoMap) == 0 {
			fmt.Println("No pull requests were found or created. Skipping comment updates.")
		} else {
			for i := 1; i < len(stack); i++ { // Iterate through stack branches again
				branch := stack[i]
				prInfo, ok := prInfoMap[branch] // Check map for this specific branch
				if !ok {
					fmt.Printf("Skipping comment update for branch '%s': No PR info was stored.\n", branch)
					continue
				}
				// If we reach here, we *do* have PR info for this branch
				fmt.Printf("Processing comment for branch %s, PR #%d...\n", branch, prInfo.Number)

				commentBody := renderStackCommentBody(stack, branch, prInfoMap)
				fmt.Printf("  Generated comment body (length %d)\n", len(commentBody)) // DEBUG

				// Get existing comment ID
				commentIDKey := fmt.Sprintf("branch.%s.socle-comment-id", branch)
				commentIDStr, errGetID := gitutils.GetGitConfig(commentIDKey)
				commentID := int64(0)
				if errGetID == nil && commentIDStr != "" {
					fmt.Sscan(commentIDStr, &commentID)                        // Ignore scan error
					fmt.Printf("  Found existing comment ID: %d\n", commentID) // DEBUG
				} else {
					fmt.Println("  No existing comment ID found.") // DEBUG
				}

				var resultingComment *github.IssueComment // Use GH type
				var commentErr error
				actionTaken := "" // For logging

				if commentID > 0 {
					// Try Update
					fmt.Printf("  Attempting to update comment ID %d...\n", commentID) // DEBUG
					resultingComment, commentErr = ghClient.UpdateComment(commentID, commentBody)
					if commentErr != nil {
						fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("  Warning: Failed to update comment ID %d: %v\n"), commentID, commentErr) // DEBUG
						// Assume comment was deleted, clear stored ID and try creating new
						commentID = 0
						_ = gitutils.UnsetGitConfig(commentIDKey)
					} else {
						actionTaken = "updated"
					}
				}

				// If no ID or update failed, try Create
				if commentID == 0 && commentErr == nil { // Only create if update didn't have a non-recoverable error
					fmt.Println("  Attempting to create new comment...") // DEBUG
					resultingComment, commentErr = ghClient.CreateComment(prInfo.Number, commentBody)
					if commentErr == nil {
						actionTaken = "created"
					}
				}

				// Handle final outcome for this PR's comment
				if commentErr != nil {
					fmt.Fprintf(os.Stderr, ui.Colors.FailureStyle.Render("  ERROR: Failed to %s stack comment for PR #%d: %v\n"), map[bool]string{true: "update", false: "create"}[commentID > 0], prInfo.Number, commentErr)
					// Continue to next PR
				} else if actionTaken != "" {
					fmt.Printf("  Stack comment %s successfully.\n", actionTaken)
					// Store the new/updated comment ID if we have one
					if resultingComment != nil && resultingComment.ID != nil {
						newCommentIDStr := fmt.Sprintf("%d", resultingComment.GetID())
						if newCommentIDStr != commentIDStr {
							fmt.Printf("  Storing new comment ID %s for branch '%s'...\n", newCommentIDStr, branch)
							errSet := gitutils.SetGitConfig(commentIDKey, newCommentIDStr)
							if errSet != nil {
								fmt.Fprintf(os.Stderr, ui.Colors.FailureStyle.Render("  CRITICAL WARNING: Failed to store comment ID %s locally for branch '%s': %v\n"), newCommentIDStr, branch, errSet)
							}
						}
					}
				} else {
					fmt.Println("  No comment action was successfully completed for this PR.") // Should not happen?
				}
			} // End of second loop
		} // End else
		fmt.Println("\nSubmit process complete.")
		return nil
	}, // Close the RunE function
}

// renderStackCommentBody generates the markdown comment.
func renderStackCommentBody(stack []string, currentBranch string, prInfoMap map[string]submittedPrInfo) string {
	var sb strings.Builder
	sb.WriteString("### 📚 Socle Stack Overview\n") // Add tool name for clarity
	sb.WriteString("This PR is part of a stack. Links to other PRs in the stack:\n\n")

	// Iterate stack from top (end) down to base (start)
	for i := len(stack) - 1; i > 0; i-- { // Skip base branch stack[0]
		branchName := stack[i]
		prInfo, ok := prInfoMap[branchName]
		indicator := ""
		if branchName == currentBranch {
			indicator = " 👈 **(Current PR)**"
		}

		if ok {
			// Found PR info for this branch
			sb.WriteString(fmt.Sprintf("* **#%d** [%s](%s)%s\n",
				prInfo.Number,
				prInfo.Title, // Use fetched title
				prInfo.URL,
				indicator,
			))
		} else {
			// No PR info (maybe submit failed for this branch earlier?)
			sb.WriteString(fmt.Sprintf("* `%s` (PR not submitted/found)%s\n",
				branchName,
				indicator,
			))
		}
	}

	// Add the base branch at the end
	sb.WriteString(fmt.Sprintf("* `%s` (base)\n", stack[0]))
	sb.WriteString("\n<!-- socle-stack-overview -->\n") // Add marker comment

	return sb.String()
}

// --- init() function for flags ---
func init() {
	AddCommand(submitCmd)
	submitCmd.Flags().BoolVar(&submitForcePush, "force", false, "Force push branches ('git push --force')")
	submitCmd.Flags().BoolVar(&submitNoPush, "no-push", false, "Skip pushing branches to the remote")
	// Default is true, flag makes it false. Use `BoolVar` and maybe check `cmd.Flags().Changed("draft")` if needed later.
	submitCmd.Flags().BoolVar(&submitDraft, "draft", true, "Create new pull requests as drafts")
}
