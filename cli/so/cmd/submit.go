// cli/cmd/submit.go
package cmd

import (
	"context" // Need context for GitHub client
	"errors"
	"fmt"
	"log/slog"
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
		// Get the output/error streams from the command
		outW := cmd.OutOrStdout()
		errW := cmd.ErrOrStderr()

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
		slog.Debug("Operating on repository:", "owner", owner, "repoName", repoName)

		// --- Get Stack Info ---
		currentBranch, currentStack, _, err := getCurrentStackInfo()
		handled, processedErr := handleShowStartupError(err, currentBranch, outW, errW)
		if processedErr != nil {
			// Handle unexpected errors from getCurrentStackInfo or the handler itself
			return processedErr // Return actual error
		}
		if handled {
			// If the handler dealt with it (printed "on base" or "untracked"), exit successfully.
			return nil
		}

		// --- Determine Full Stack ---
		slog.Debug("Determining full stack...")
		allParents, err := gitutils.GetAllSocleParents()
		if err != nil {
			return fmt.Errorf("failed to read all tracking relationships: %w", err)
		}
		childMap := gitutils.BuildChildMap(allParents)
		// Find all descendants of the *last* branch in the current stack
		// This assumes the current checkout is somewhere within the stack of interest
		tipOfCurrentStack := currentStack[len(currentStack)-1]
		descendants := gitutils.FindAllDescendants(tipOfCurrentStack, childMap)

		// Combine the current stack (base -> current -> tip) with any further descendants
		fullStack := currentStack
		processedDescendants := make(map[string]bool) // Avoid adding duplicates if currentStack already had some
		for _, b := range currentStack {
			processedDescendants[b] = true
		}

		// Simple append - order might not be perfect topological, but contains all nodes
		// TODO: Improve ordering if needed later (e.g., topological sort)
		for _, desc := range descendants {
			if !processedDescendants[desc] {
				fullStack = append(fullStack, desc)
				processedDescendants[desc] = true
			}
		}
		slog.Debug("Full stack identified for processing:", "fullStack", fullStack)
		// --- End Determine Full Stack ---
		if len(fullStack) <= 1 {
			fmt.Println("Current branch is the base or directly on base. Nothing to submit.")
			return nil
		}

		// Map to store PR info for the comment phase
		prInfoMap := make(map[string]submittedPrInfo)

		// 3. Iterate and Process Stack (skip base branch at index 0)
		for i := 1; i < len(fullStack); i++ {
			branch := fullStack[i]
			parent, ok := allParents[branch]
			if !ok {
				// This shouldn't happen if the branch came from tracking data
				fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("Warning: Could not find tracked parent for '%s'. Skipping processing.\n"), branch)
				continue
			}

			slog.Debug("Processing", "branch", branch, "parent", parent)

			// 3a. Push Branch (unless skipped)
			if !submitNoPush {
				slog.Debug("Pushing branch", "branch", branch, "remote", remoteName)
				err := gitutils.PushBranch(branch, remoteName, submitForcePush)
				if err != nil {
					// Allow continuing? Or stop? Let's stop for now.
					return fmt.Errorf("failed to push branch '%s', aborting submit: %w", branch, err)
				}
				slog.Debug("Push successful.")
			} else {
				slog.Debug("Skipping push (--no-push).")
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
					currentPR = existingPR

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
					}

					currentPrNumberStr := fmt.Sprintf("%d", currentPR.GetNumber()) // Use number from fetched/updated PR

					if currentPrNumberStr != prNumberStr { // Only write if missing or different
						slog.Debug("Updating stored PR number", "currentPrNumberStr", currentPrNumberStr, "branch", branch)
						errSet := gitutils.SetGitConfig(prNumberKey, currentPrNumberStr)
						if errSet != nil {
							fmt.Fprintf(os.Stderr, ui.Colors.FailureStyle.Render("CRITICAL WARNING: Failed to store PR number %d locally for branch '%s': %v\n"), currentPR.GetNumber(), branch, errSet)
						}
					}

					fmt.Printf("PR #%d link: %s\n", currentPR.GetNumber(), currentPR.GetHTMLURL())
				}
			}

			// If we reach here, we need to create a new PR
			if prNumber == 0 {
				// --- Create New PR ---
				slog.Debug("Attempting to create PR", "branch", branch, "parent", parent)

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
				slog.Debug("  Differences found. Proceeding with PR creation details...")

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
				slog.Debug("Submitting PR", "as type", map[bool]string{true: "Draft", false: "Ready"}[isDraft])
				newPR, err := ghClient.CreatePullRequest(branch, parent, title, body, isDraft)
				if err != nil {
					// Abort on failure to create
					return fmt.Errorf("failed to create pull request for branch '%s': %w", branch, err)
				}
				currentPR = newPR

				// Store the new PR number
				newPrNumberStr := fmt.Sprintf("%d", newPR.GetNumber())
				slog.Debug("Storing PR number for branch", "newPrNumberStr", newPrNumberStr, "branch", branch)
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
				slog.Debug("Stored PR info", "branch", branch, "prInfoMap", prInfoMap[branch])
			} else {
				slog.Debug("DEBUG: No valid PR object after processing (skipped or failed).", "branch", branch)
			}

		}

		slog.Debug("Before comment phase", "prInfoMap entries", len(prInfoMap))
		for branch, info := range prInfoMap {
			slog.Debug("Map entry", "branch", branch, "PR number", info.Number, "URL", info.URL)
		}

		// --- Second Loop: Add/Update Stack Comments ---
		slog.Debug("\n--- Phase 2: Updating PR comments with stack overview ---")
		stackCommentMarker := "<!-- socle-stack-overview -->"

		if len(prInfoMap) == 0 {
			fmt.Println("No pull requests were found or created. Skipping comment updates.")
		} else {
			for i := 1; i < len(fullStack); i++ { // Iterate through stack branches again
				branch := fullStack[i]
				prInfo, ok := prInfoMap[branch] // Check map for this specific branch
				if !ok {
					fmt.Printf("Skipping comment update for branch '%s': No PR info was stored.\n", branch)
					continue
				}
				// If we reach here, we *do* have PR info for this branch
				slog.Debug("Processing comment", "branch", branch, "PR number", prInfo.Number)

				commentBody := renderStackCommentBody(fullStack, branch, prInfoMap)
				// Get existing comment ID
				commentIDKey := fmt.Sprintf("branch.%s.socle-comment-id", branch)
				commentIDStr, errGetID := gitutils.GetGitConfig(commentIDKey)
				commentID := int64(0)
				configReadError := false

				if errGetID == nil && commentIDStr != "" {
					_, errScan := fmt.Sscan(commentIDStr, &commentID)
					if errScan != nil {
						fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("  Warning: Could not parse stored comment ID '%s': %v\n"), commentIDStr, errScan)
						commentID = 0 // Reset on error
					} else {
						fmt.Printf("  Found comment ID %d in local config.\n", commentID)
					}
				} else if errGetID != nil && !errors.Is(errGetID, gitutils.ErrConfigNotFound) {
					// Unexpected error reading config
					fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("  Warning: Failed to read comment ID config: %v\n"), errGetID)
					configReadError = true // Flag that we had a config issue
				}
				// If we get here and commentID is still 0, it means config wasn't found or had error

				// If local ID wasn't found/valid, try searching GitHub using marker
				if commentID == 0 && !configReadError {
					slog.Debug("No valid local comment ID found. Searching GitHub for marker comment ...")
					foundID, errFind := ghClient.FindCommentWithMarker(prInfo.Number, stackCommentMarker)
					if errFind != nil {
						// Error calling the API to find the comment
						fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("  Warning: Failed to search for marker comment on PR #%d: %v\n"), prInfo.Number, errFind)
						// Proceed to try creating a new comment? Or bail? Let's try creating.
					} else if foundID > 0 {
						fmt.Printf("  Found existing comment ID %d via marker on GitHub.\n", foundID)
						commentID = foundID // Use the ID found via API
						// Optionally store this back to local config for next time?
						// errSet := gitutils.SetGitConfig(commentIDKey, fmt.Sprintf("%d", commentID)) ... handle error ...
					} else {
						slog.Debug("No existing marker comment found on GitHub.")
					}
				}

				var resultingComment *github.IssueComment
				var commentErr error
				actionTaken := ""

				if commentID > 0 {
					// Try Update
					slog.Debug("Attempting to update comment", "commentID", commentID)
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
					slog.Debug("Stack comment", "actionTaken", actionTaken)
					if resultingComment != nil && resultingComment.ID != nil {
						newCommentIDStr := fmt.Sprintf("%d", resultingComment.GetID())
						if newCommentIDStr != commentIDStr {
							slog.Debug("Storing new comment ID", "newCommentIDStr", newCommentIDStr, "branch", branch)
							errSet := gitutils.SetGitConfig(commentIDKey, newCommentIDStr)
							if errSet != nil {
								fmt.Fprintf(os.Stderr, ui.Colors.FailureStyle.Render("  CRITICAL WARNING: Failed to store comment ID %s locally for branch '%s': %v\n"), newCommentIDStr, branch, errSet)
							}
						}
					}
				} else {
					slog.Debug("No comment action was successfully completed for this PR.")
				}
			}
		}
		slog.Debug("Submit process complete.")
		return nil
	},
}

// renderStackCommentBody generates the markdown comment.
func renderStackCommentBody(stack []string, currentBranch string, prInfoMap map[string]submittedPrInfo) string {
	var sb strings.Builder
	sb.WriteString("### 📚 Stack Overview\n") // Add tool name for clarity
	sb.WriteString("This PR is part of a stack. Links to other PRs in the stack:\n\n")

	// Iterate stack from top (end) down to base (start)
	for i := len(stack) - 1; i > 0; i-- { // Skip base branch stack[0]
		branchName := stack[i]
		prInfo, ok := prInfoMap[branchName]
		indicator := ""
		if branchName == currentBranch {
			indicator = " 👈"
		}

		if ok {
			// Found PR info for this branch
			sb.WriteString(fmt.Sprintf("* **#%d** %s\n",
				prInfo.Number,
				indicator,
			))
		} else {
			// No PR info (maybe submit failed for this branch earlier?)
			sb.WriteString(fmt.Sprintf("* `%s` (Coming soon 🤞)%s\n",
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
	submitCmd.Flags().BoolVar(&submitDraft, "draft", true, "Create new pull requests as drafts")
}
