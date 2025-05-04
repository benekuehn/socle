package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/benekuehn/socle/cli/so/gitutils"
	"github.com/benekuehn/socle/cli/so/internal/actions"
	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/ui"
	"github.com/spf13/cobra"
)

type submittedPrInfo struct {
	Number int
}

type submitCmdRunner struct {
	// Dependencies
	logger         *slog.Logger
	ghClient       gh.ClientInterface
	createGHClient func(ctx context.Context, owner, repo string) (gh.ClientInterface, error)
	stdout         io.Writer
	stderr         io.Writer

	// Configuration from flags
	forcePush bool
	noPush    bool
	draft     bool

	// --- TESTING FLAGS --- (passed via options if needed, or kept if strictly for cmd level tests)
	testSubmitTitle       string
	testSubmitBody        string
	testSubmitEditConfirm bool

	// Internal state
	owner        string
	repoName     string
	remoteName   string
	prInfoMap    map[string]submittedPrInfo
	submitErrors []error
}

func (r *submitCmdRunner) run(ctx context.Context, cmd *cobra.Command) error {
	r.logger.Debug("Starting submit command execution")

	// --- Phase 1: Preparation ---
	fullStack, allParents, err := r.prepareSubmit(ctx)
	if err != nil {
		// Handle fatal preparation errors (e.g., bad remote, client creation failed, failed to get stack)
		// Also handles trivial stack case internally.
		// If err is nil but fullStack is nil, it means preparation decided to exit gracefully (e.g., trivial stack).
		if err == errTrivialStack || err == errStartupHandled {
			return nil
		}
		return fmt.Errorf("failed during preparation: %w", err)
	}
	// If fullStack is nil here without error, it implies prepareSubmit handled the exit (e.g., trivial stack)
	// Although the current prepareSubmit returns errTrivialStack in that case.

	r.prInfoMap = make(map[string]submittedPrInfo)
	r.submitErrors = make([]error, 0)

	// --- Phase 2: Process Stack (Submit PRs) ---
	if err := r.processStack(ctx, cmd, fullStack, allParents); err != nil {
		// Handle fatal errors during stack processing (push failed, submit action failed fatally, user cancelled)
		return fmt.Errorf("failed processing stack: %w", err) // Return immediately on fatal error
	}

	// --- Phase 3: Update Stack Comments ---
	r.updateStackComments(ctx, fullStack)

	// --- Phase 4: Final Summary ---
	r.summarizeResults()

	// Return nil to indicate overall success, even if warnings were collected in r.submitErrors.
	// Fatal errors would have caused an earlier return.
	return nil
}

// Define sentinel errors for specific exit conditions
var errTrivialStack = errors.New("trivial stack, nothing to submit")
var errStartupHandled = errors.New("startup handled (e.g., help shown)")

// prepareSubmit handles initial setup: checks, client creation, stack determination.
// Returns the full stack, parent map, or a specific error (including errTrivialStack).
func (r *submitCmdRunner) prepareSubmit(ctx context.Context) ([]string, map[string]string, error) {
	r.logger.Debug("Preparing submit operation")

	r.remoteName = "origin" // TODO: Make configurable?
	remoteURL, err := gitutils.GetRemoteURL(r.remoteName)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot get remote URL for '%s': %w", r.remoteName, err)
	}
	r.logger.Debug("Found remote URL", "remote", r.remoteName, "url", remoteURL)

	r.owner, r.repoName, err = gitutils.ParseOwnerAndRepo(remoteURL)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot parse owner/repo from remote '%s' URL '%s': %w", r.remoteName, remoteURL, err)
	}
	r.logger.Debug("Operating on repository", "owner", r.owner, "repoName", r.repoName)

	r.ghClient, err = r.createGHClient(ctx, r.owner, r.repoName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}
	r.logger.Debug("GitHub client created/obtained")

	// Handle potential startup issues (like not being in a git repo or stack)
	currentBranch, currentStack, _, err := getCurrentStackInfo()
	handled, processedErr := handleShowStartupError(err, currentBranch, r.stdout, r.stderr)
	if processedErr != nil {
		return nil, nil, processedErr
	}
	if handled {
		return nil, nil, errStartupHandled
	}
	r.logger.Debug("Startup checks passed", "currentBranch", currentBranch)

	r.logger.Debug("Determining full stack...")
	fullStack, allParents, err := gitutils.GetFullStackForSubmit(currentStack)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to determine full stack: %w", err)
	}
	r.logger.Debug("Full stack identified for processing", "fullStack", fullStack)

	if len(fullStack) <= 1 {
		fmt.Fprintln(r.stdout, "Current branch is the base or directly on base. Nothing to submit.")
		return nil, nil, errTrivialStack
	}

	return fullStack, allParents, nil
}

// processStack iterates through the stack branches, pushes (if enabled), and submits PRs.
// It populates r.prInfoMap and r.submitErrors (for non-fatal internal errors).
// Returns a fatal error if a push fails, submit action fails critically, or user cancels.
func (r *submitCmdRunner) processStack(ctx context.Context, cmd *cobra.Command, fullStack []string, allParents map[string]string) error {
	fmt.Fprintln(r.stdout, "Processing stack...")
	for i := 1; i < len(fullStack); i++ {
		branch := fullStack[i]
		parent, ok := allParents[branch]
		if !ok {
			// This shouldn't happen if GetFullStackForSubmit worked correctly
			// Create the error object directly
			err := fmt.Errorf("internal error: Could not find tracked parent for '%s' in parent map. Skipping processing", branch)
			r.logger.Error(err.Error(), "branch", branch) // Log the error message string
			r.submitErrors = append(r.submitErrors, err)  // Append the error object
			continue                                      // Skip this branch
		}

		fmt.Fprintf(r.stdout, "\nProcessing branch: %s (parent: %s)\n", branch, parent)

		prInfoResult, err := r.submitBranch(ctx, cmd, branch, parent)
		if err != nil {
			// submitBranch returns fatal errors (push fail, action fail) or ErrSubmitCancelled
			if errors.Is(err, actions.ErrSubmitCancelled) {
				fmt.Fprintln(r.stdout, ui.Colors.WarningStyle.Render("Submit operation cancelled."))
				return err // Return cancellation error to halt processing
			}
			// Otherwise, it's a fatal error from push or action
			return fmt.Errorf("failed processing branch '%s': %w", branch, err)
		}

		if prInfoResult != nil {
			r.prInfoMap[branch] = *prInfoResult
			r.logger.Debug("Stored PR info from submitBranch", "branch", branch, "prInfo", *prInfoResult)
		} else {
			r.logger.Debug("No PR info returned from submitBranch (skipped or handled internally).", "branch", branch)
		}
	}
	return nil
}

// updateStackComments iterates through the processed PRs and updates their comments.
// Errors encountered here are collected in r.submitErrors.
func (r *submitCmdRunner) updateStackComments(ctx context.Context, fullStack []string) {
	r.logger.Debug("Updating PR comments with stack overview")
	stackCommentMarker := "<!-- socle-stack-overview -->"

	if len(r.prInfoMap) == 0 {
		fmt.Fprintln(r.stdout, "\nNo pull requests were found or created/updated. Skipping comment updates.")
		return
	}

	fmt.Fprintln(r.stdout, "\nUpdating PR comments with stack overview...")
	for i := 1; i < len(fullStack); i++ { // Iterate through stack branches again
		branch := fullStack[i]
		prInfo, ok := r.prInfoMap[branch] // Check map for this specific branch
		if !ok {
			r.logger.Debug("Skipping comment update for branch: No PR info was stored.", "branch", branch)
			continue
		}

		commentBody := renderStackCommentBody(fullStack, branch, stackCommentMarker, r.prInfoMap)

		err := actions.EnsureStackComment(ctx, r.ghClient, branch, prInfo.Number, commentBody, stackCommentMarker)
		if err != nil {
			// TODO: Differentiate critical errors vs warnings?
			wrappedErr := fmt.Errorf("error processing stack comment for PR #%d (branch '%s'): %w", prInfo.Number, branch, err)
			fmt.Fprintln(r.stderr, ui.Colors.WarningStyle.Render("  "+wrappedErr.Error())) // Print immediate feedback
			r.submitErrors = append(r.submitErrors, wrappedErr)
			continue // Continue processing comments for other PRs
		} else {
			fmt.Fprintf(r.stdout, "  Stack comment processed for PR #%d.\n", prInfo.Number)
		}
	}
}

// summarizeResults prints the final status and any collected errors.
func (r *submitCmdRunner) summarizeResults() {
	fmt.Fprintln(r.stdout, "\nSubmit process finished.")
	if len(r.submitErrors) > 0 {
		fmt.Fprintln(r.stderr, ui.Colors.WarningStyle.Render("\nEncountered warnings/errors during submit:"))
		for _, submitErr := range r.submitErrors {
			fmt.Fprintln(r.stderr, " - "+submitErr.Error())
		}
	}
}

// submitBranch now orchestrates push and calls the main action.
// It needs access to the runner's state (flags, ghClient). Change signature.
func (r *submitCmdRunner) submitBranch( // Make it a method of submitCmdRunner
	ctx context.Context,
	cmd *cobra.Command, // Keep cmd if needed by actions.SubmitBranch
	branch string,
	parent string,
	// remoteName string, // Get from r.remoteName
) (*submittedPrInfo, error) {

	// Access flags from the runner struct
	doPush := !r.noPush
	forcePush := r.forcePush

	r.logger.Debug("submitBranch: Orchestrating action", "branch", branch, "parent", parent)

	// 1. Push Branch (if enabled)
	if doPush {
		r.logger.Debug("Pushing branch", "branch", branch, "remote", r.remoteName, "force", forcePush)
		err := gitutils.PushBranch(branch, r.remoteName, forcePush)
		if err != nil {
			// Treat push failure as fatal
			return nil, fmt.Errorf("failed to push branch '%s': %w", branch, err)
		}
		fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render("  Branch pushed successfully."))
	} else {
		fmt.Fprintln(r.stdout, "  Skipping push (--no-push).")
	}

	// 2. Call the SubmitBranch action to handle PR logic
	opts := actions.SubmitBranchOptions{
		// Use runner's config
		IsDraft:               r.draft,
		TestSubmitTitle:       r.testSubmitTitle,
		TestSubmitBody:        r.testSubmitBody,
		TestSubmitEditConfirm: r.testSubmitEditConfirm,
	}
	r.logger.Debug("Calling actions.SubmitBranch", "branch", branch, "options", opts)

	finalPR, err := actions.SubmitBranch(ctx, r.ghClient, cmd, branch, parent, opts)
	if err != nil {
		// Error could be fatal API error or ErrSubmitCancelled from action
		return nil, err // Propagate error up (already wrapped by SubmitBranch if needed)
	}

	// 3. Return PR info if available
	if finalPR != nil {
		return &submittedPrInfo{
			Number: finalPR.GetNumber(),
		}, nil
	}

	// If finalPR is nil and err is nil, it means the action determined a skip was needed (e.g., no diff)
	r.logger.Debug("submitBranch determined a skip was needed (no diff likely)", "branch", branch)
	return nil, nil
}

func renderStackCommentBody(stack []string, currentBranch string, stackCommentMarker string, prInfoMap map[string]submittedPrInfo) string {
	var sb strings.Builder
	sb.WriteString("**Stack Overview:**\n\n")
	for i, branchName := range stack {
		if i == 0 {
			continue // Skip base branch
		}
		prInfo, ok := prInfoMap[branchName]
		indicator := ""
		if branchName == currentBranch {
			indicator = " 👈"
		}

		if ok {
			sb.WriteString(fmt.Sprintf("* **#%d** %s\n",
				prInfo.Number,
				indicator,
			))
		} else {
			sb.WriteString(fmt.Sprintf("* `%s` (Coming soon 🤞)%s\n",
				branchName,
				indicator,
			))
		}
	}

	sb.WriteString(fmt.Sprintf("* `%s` (base)\n", stack[0]))
	sb.WriteString(fmt.Sprintf("\n%s\n", stackCommentMarker))

	return sb.String()
}
