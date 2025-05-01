package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/benekuehn/socle/cli/so/gitutils"
	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type statusResult struct {
	text   string
	render func(string) string
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Display the current tracked stack of branches",
	Long: `Shows the sequence of tracked branches leading from the stack's base
branch to the current branch, based on metadata set by 'socle track'.
Includes status indicating if a branch needs rebasing onto its parent.`, // Updated Long desc
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// --- Get Current Branch & Stack Info ---
		currentBranch, stack, baseBranch, err := getCurrentStackInfo()
		if err != nil {
			return handleShowStartupError(err, currentBranch)
		} // Use helper for startup errors
		fmt.Printf("Current branch: %s (Stack base: %s)\n", currentBranch, baseBranch)

		// --- Initialize GitHub Client (Lazy) ---
		// We only initialize it if we actually need to fetch PRs
		var ghClient *gh.Client
		var ghClientErr error                // Store potential client init error
		var ghClientInitialized bool = false // Track if we tried init
		// --- Print the stack with status ---
		fmt.Println("\nCurrent Stack Status:")
		// Print base branch first (no status check needed)
		baseMarker := ""
		if baseBranch == currentBranch {
			baseMarker = " *"
		}
		fmt.Printf("  %s (base)%s\n", baseBranch, baseMarker)

		// Loop through stack branches (skip base at index 0)
		for i := 1; i < len(stack); i++ {
			branchName := stack[i]
			parentName := stack[i-1]

			rebaseStatus := getRebaseStatus(parentName, branchName)
			// Call the refactored PR status helper
			prStatus := getPrStatusDisplay(ctx, &ghClient, &ghClientErr, &ghClientInitialized, branchName)

			marker := ""
			if branchName == currentBranch {
				marker = " *"
			}

			fmt.Printf("  -> %s %s %s%s\n",
				branchName,
				rebaseStatus.render(rebaseStatus.text),
				prStatus.render(prStatus.text),
				marker,
			)
		} // End stack loop

		return nil
	},
}

// --- Helper Functions ---

// handleShowStartupError manages errors from getCurrentStackInfo for better UX
func handleShowStartupError(err error, currentBranchAttempt string) error {
	// Attempt to get current branch again if helper failed early
	cb := currentBranchAttempt
	if cb == "" {
		cb, _ = gitutils.GetCurrentBranch()
	}

	if err != nil && strings.Contains(err.Error(), "not tracked by socle") {
		knownBases := map[string]bool{"main": true, "master": true, "develop": true} // TODO: Configurable
		if knownBases[cb] {
			fmt.Printf("Currently on the base branch '%s'.\n", cb)
		} else {
			fmt.Printf("Branch '%s' is not currently tracked by socle.\n", cb)
			fmt.Println("Use 'so track' to associate it with a parent branch and start a stack.")
		}
		return nil // Clean exit for untracked/base branch
	}
	// Otherwise, return the original, unexpected error
	return err
}

// getRebaseStatus determines the rebase status string and render function
func getRebaseStatus(parentName, branchName string) statusResult {
	needsRestack, errCheck := gitutils.NeedsRestack(parentName, branchName)

	if errCheck != nil {
		fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("  Warning: Could not check restack status for '%s': %v\n"), branchName, errCheck)
		return statusResult{"(Rebase: Error)", func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	} else if needsRestack {
		return statusResult{"(Needs Restack)", func(s string) string { return ui.Colors.WarningStyle.Render(s) }}
	} else {
		return statusResult{"(Up-to-date)", func(s string) string { return ui.Colors.SuccessStyle.Render(s) }}
	}
}

// getPrStatusDisplay reads config, calls gh service, and maps result to display statusResult
func getPrStatusDisplay(ctx context.Context, ghClient **gh.Client, clientErr *error, clientInitialized *bool, branchName string) statusResult {
	defaultRender := func(s string) string { return s }

	// --- Step 1: Read Local Config ---
	prNumberKey := fmt.Sprintf("branch.%s.socle-pr-number", branchName)
	prNumberStr, errPRNum := gitutils.GetGitConfig(prNumberKey) // Uses updated GetGitConfig

	// --- FIX: Treat "Not Found" error correctly ---
	if errors.Is(errPRNum, gitutils.ErrConfigNotFound) {
		// Key doesn't exist, meaning PR hasn't been submitted and tracked by socle yet
		return statusResult{"(PR: Not Submitted)", defaultRender}
	} else if errPRNum != nil {
		// Any other error reading the config IS a config error
		fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("  Warning: Could not read PR number config for '%s': %v\n"), branchName, errPRNum)
		return statusResult{"(PR: Config Err)", func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	}

	if prNumberStr == "" {
		return statusResult{"(PR: Not Submitted)", defaultRender}
	}

	// --- Step 2: Parse PR Number ---
	prNumber, errParsePR := strconv.Atoi(prNumberStr)
	if errParsePR != nil {
		fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("  Warning: Could not parse PR number '%s' for '%s': %v\n"), prNumberStr, branchName, errParsePR)
		return statusResult{"(PR: Invalid #)", func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	}
	// --- Step 3: Ensure GitHub Client is Initialized (Lazy) ---
	if !*clientInitialized {
		*clientInitialized = true
		remoteName := "origin"
		remoteURL, errURL := gitutils.GetRemoteURL(remoteName)
		if errURL != nil {
			*clientErr = fmt.Errorf("cannot get remote URL '%s': %w", remoteName, errURL)
		} else {
			owner, repoName, errParse := gitutils.ParseOwnerAndRepo(remoteURL)
			if errParse != nil {
				*clientErr = fmt.Errorf("cannot parse owner/repo from '%s': %w", remoteURL, errParse)
			} else {
				client, errCli := gh.NewClient(ctx, owner, repoName) // Call the constructor from gh package
				if errCli != nil {
					*clientErr = fmt.Errorf("GitHub client init failed: %w", errCli)
				} else {
					*ghClient = client
				}
			}
		}
		if *clientErr != nil {
			fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("Warning: Cannot fetch PR status: %v\n"), *clientErr)
		}
	}

	// --- Step 4: Check if Client Failed Initialization ---
	if *ghClient == nil {
		return statusResult{"(PR: Login/Setup Needed)", func(s string) string { return ui.Colors.WarningStyle.Render(s) }}
	}

	// --- Step 5: Call GitHub Service ---
	semanticStatus, _, errGetStatus := (*ghClient).GetPullRequestStatus(prNumber) // Dereference pointer
	if errGetStatus != nil {
		// API error occurred during fetch
		fmt.Fprintf(os.Stderr, ui.Colors.WarningStyle.Render("  Warning: Could not fetch PR #%d for '%s': %v\n"), prNumber, branchName, errGetStatus)
		statusText := fmt.Sprintf("(PR #%d: API Error)", prNumber)
		return statusResult{statusText, func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	}
	// --- Step 6: Map Semantic Status to Display Status ---
	prNumStr := fmt.Sprintf("#%d", prNumber)
	switch semanticStatus {
	case gh.PRStatusMerged:
		return statusResult{fmt.Sprintf("(PR %s: Merged)", prNumStr), func(s string) string { return lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Render(s) }} // Purple
	case gh.PRStatusClosed:
		return statusResult{fmt.Sprintf("(PR %s: Closed)", prNumStr), func(s string) string { return ui.Colors.FailureStyle.Render(s) }} // Red
	case gh.PRStatusDraft:
		return statusResult{fmt.Sprintf("(PR %s: Draft)", prNumStr), func(s string) string { return ui.Colors.FaintStyle.Render(s) }} // Faint
	case gh.PRStatusOpen:
		return statusResult{fmt.Sprintf("(PR %s: Open)", prNumStr), func(s string) string { return ui.Colors.SuccessStyle.Render(s) }} // Green
	case gh.PRStatusNotFound:
		return statusResult{fmt.Sprintf("(PR #%d: Not Found)", prNumber), func(s string) string { return ui.Colors.FailureStyle.Render(s) }} // Red
	default: // Unknown or other API error status from gh package
		return statusResult{fmt.Sprintf("(PR #%d: %s)", prNumber, semanticStatus), func(s string) string { return ui.Colors.FailureStyle.Render(s) }} // Red
	}
}

func init() {
	AddCommand(showCmd)
}
