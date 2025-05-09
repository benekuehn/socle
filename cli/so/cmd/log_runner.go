package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/benekuehn/socle/cli/so/internal/cmdutils"
	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// branchLogInfo holds all data needed to render a single branch entry in the log.
type branchLogInfo struct {
	branchName      string
	parentName      string
	indicator       string
	branchNameStyle func(string) string // Function to apply style (e.g., bold)
	rebaseText      string
	prText          string
	isLastInStack   bool // To handle the final vertical connector │
}

// prFetchResult holds the outcome of a single PR status fetch operation.
// It's used to send data from goroutines back to the main processing logic.
type prFetchResult struct {
	branchName string       // The name of the branch for which PR status was fetched.
	prText     string       // The formatted text to display for the PR status.
	prStatus   statusResult // The raw statusResult, if needed for color or more details.
	err        error        // Any error encountered during the fetch for this specific PR.
}

type statusResult struct {
	text   string
	render func(string) string
}

type logCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
}

func (r *logCmdRunner) run(ctx context.Context) error {
	currentBranch, stackToCurrent, baseBranch, err := git.GetCurrentStackInfo()
	handled, processedErr := cmdutils.HandleStartupError(err, currentBranch, r.stdout, r.stderr)
	if processedErr != nil {
		return processedErr
	}
	if handled {
		return nil
	}

	fullOrderedStack, _, err := git.GetFullStack(stackToCurrent)
	if err != nil {
		_, _ = fmt.Fprintf(r.stderr, ui.Colors.FailureStyle.Render("Error: Could not determine the full stack: %v\n"), err)
		return err
	}

	boldStyle := lipgloss.NewStyle().Bold(true)
	boldRenderer := func(s string) string { return boldStyle.Render(s) }

	// Initialize GitHub client once.
	var ghClient *gh.Client
	var ghClientInitError error // Store client init error, distinct from individual PR fetch errors
	remoteName := "origin"
	remoteURL, errURL := git.GetRemoteURL(remoteName)
	if errURL != nil {
		ghClientInitError = fmt.Errorf("cannot get remote URL '%s': %w", remoteName, errURL)
	} else {
		owner, repoName, errParse := git.ParseOwnerAndRepo(remoteURL)
		if errParse != nil {
			ghClientInitError = fmt.Errorf("cannot parse owner/repo from '%s': %w", remoteURL, errParse)
		} else {
			client, errCli := gh.NewClient(ctx, owner, repoName)
			if errCli != nil {
				ghClientInitError = fmt.Errorf("GitHub client init failed: %w", errCli)
			} else {
				ghClient = client
			}
		}
	}
	// Removed Fprintf for GitHub client initialization duration

	if ghClientInitError != nil {
		_, _ = fmt.Fprintf(r.stderr, ui.Colors.WarningStyle.Render("Warning: GitHub client initialization failed: %v\nPR statuses may not be available.\n"), ghClientInitError)
	}

	numBranchesInStack := len(fullOrderedStack) - 1 // Number of actual branches, excluding base
	if numBranchesInStack < 0 {
		numBranchesInStack = 0
	}

	branchInfos := make([]branchLogInfo, 0, numBranchesInStack)

	// Pre-fetch all parent OIDs for branches in the stack to reduce git calls
	parentOIDs := make(map[string]string)
	if numBranchesInStack > 0 {
		parentNamesToFetch := make(map[string]struct{}) // Use a map for unique names
		for i := len(fullOrderedStack) - 1; i >= 1; i-- {
			// fullOrderedStack is [base, child1, child2, ..., current, ..., top]
			// We iterate from top down. Parent of fullOrderedStack[i] is fullOrderedStack[i-1]
			if i > 0 { // Ensure parent index is valid
				parentNamesToFetch[fullOrderedStack[i-1]] = struct{}{}
			}
		}
		uniqueParentNames := make([]string, 0, len(parentNamesToFetch))
		for name := range parentNamesToFetch {
			uniqueParentNames = append(uniqueParentNames, name)
		}

		if len(uniqueParentNames) > 0 {
			var errFetchParentOIDs error
			parentOIDs, errFetchParentOIDs = git.GetMultipleBranchCommits(uniqueParentNames)
			if errFetchParentOIDs != nil {
				// Log a warning but proceed; getRebaseStatus will handle missing OIDs by returning an error status
				_, _ = fmt.Fprintf(r.stderr, ui.Colors.WarningStyle.Render("Warning: Could not pre-fetch some parent OIDs: %v\nRebase statuses might be affected.\n"), errFetchParentOIDs)
				// Ensure parentOIDs map is not nil to avoid panics, even if it's empty or partially filled
				if parentOIDs == nil {
					parentOIDs = make(map[string]string)
				}
			}
		}
	}

	for i := len(fullOrderedStack) - 1; i >= 1; i-- {
		branchName := fullOrderedStack[i]
		parentName := fullOrderedStack[i-1]
		indicator := "◯"
		if branchName == currentBranch {
			indicator = "◉"
		}

		parentOID, ok := parentOIDs[parentName]
		if !ok {
			// This might happen if GetMultipleBranchCommits failed for this parent
			// getRebaseStatus will handle an empty parentOID string by returning an error status
			parentOID = "" // Ensure it's an empty string if not found
			_, _ = fmt.Fprintf(r.stderr, ui.Colors.WarningStyle.Render("Warning: Parent OID for '%s' not found after pre-fetch. Rebase status for '%s' may be incorrect.\n"), parentName, branchName)
		}

		// Call getRebaseStatus with the pre-fetched parentOID
		rebaseStatusResult := getRebaseStatus(parentName, branchName, parentOID, r.stderr)

		var rebaseText string
		switch rebaseStatusResult.text {
		case "(Needs Restack)":
			rebaseText = "🚧 Needs restack"
		case "(Up-to-date)":
			rebaseText = "🤝 Up-to-date"
		default:
			rebaseText = "⚠️ Rebase status error"
		}

		initialPrText := "⏳ Loading PR..."
		if ghClient == nil { // If client init totally failed
			initialPrText = "⚠️ PR client N/A"
		}

		branchInfos = append(branchInfos, branchLogInfo{
			branchName:      branchName,
			parentName:      parentName,
			indicator:       indicator,
			branchNameStyle: boldRenderer,
			rebaseText:      rebaseText,
			prText:          initialPrText,
			isLastInStack:   (i == 1), // True if this is the branch just above base
		})
	}

	// Pass 2: Fetch PR statuses in parallel if ghClient is available
	if ghClient != nil && numBranchesInStack > 0 {
		// Removed Start timing for GH operations

		prResultsChan := make(chan prFetchResult, numBranchesInStack)
		var wg sync.WaitGroup

		for _, bi := range branchInfos { // Iterate over the already prepared branchInfos to launch goroutines
			wg.Add(1)
			go func(branchNameToFetch string, currentCtx context.Context) {
				defer wg.Done()
				prStatus := getPrStatusDisplay(ghClient, branchNameToFetch, r.stderr)

				// Convert prStatus.text to the desired icon-based string
				var finalPrText string
				if strings.Contains(prStatus.text, ": Merged)") {
					finalPrText = "✅ PR merged"
				} else if strings.Contains(prStatus.text, ": Closed)") {
					finalPrText = "🚫 PR closed"
				} else if strings.Contains(prStatus.text, ": Draft)") {
					finalPrText = "📝 PR draft"
				} else if strings.Contains(prStatus.text, ": Open)") {
					finalPrText = "➡️ PR open"
				} else if strings.Contains(prStatus.text, "Not Submitted") {
					finalPrText = "⚪ PR not submitted"
				} else if strings.Contains(prStatus.text, "Client N/A") { // from getPrStatusDisplay if client is nil
					finalPrText = "⚠️ PR client N/A" // Should not happen if ghClient != nil here
				} else { // Covers API errors, config errors, invalid #, etc.
					finalPrText = "⚠️ PR status error"
				}
				prResultsChan <- prFetchResult{branchName: branchNameToFetch, prText: finalPrText, prStatus: prStatus, err: nil} // Assuming err from getPrStatus itself is handled by its return text
			}(bi.branchName, ctx) // Pass branchName and ctx to goroutine
		}

		go func() {
			wg.Wait()
			close(prResultsChan)
		}()

		// Collect results and update branchInfos
		// Create a map for quick updates
		updatedPrTexts := make(map[string]string)
		for result := range prResultsChan {
			if result.err != nil {
				_, _ = fmt.Fprintf(r.stderr, "Error fetching PR for %s: %v\n", result.branchName, result.err)
				updatedPrTexts[result.branchName] = "⚠️ PR fetch error"
			} else {
				updatedPrTexts[result.branchName] = result.prText
			}
		}
		// Apply updates to the main branchInfos slice
		for i, bi := range branchInfos {
			if text, ok := updatedPrTexts[bi.branchName]; ok {
				branchInfos[i].prText = text
			}
		}
	}

	// Pass 3: Render the log
	_, _ = fmt.Fprintln(r.stdout) // Top padding

	// branchInfos is already in display order (top of stack first)
	for _, info := range branchInfos {
		_, _ = fmt.Fprintf(r.stdout, "    %s  %s\n", info.indicator, info.branchNameStyle(info.branchName))
		_, _ = fmt.Fprintf(r.stdout, "    │  %s | %s\n", info.rebaseText, info.prText)
		if !info.isLastInStack {
			_, _ = fmt.Fprintln(r.stdout, "    │")
		}
	}

	_, _ = fmt.Fprintln(r.stdout, "  ╭─╯")
	_, _ = fmt.Fprintf(r.stdout, "   ~ %s\n", boldRenderer(baseBranch))

	_, _ = fmt.Fprintln(r.stdout) // Bottom padding

	return nil
}

// getRebaseStatus now takes a pre-fetched parentOID.
// It calculates needsRestack by comparing parentOID with the merge-base of parentName and branchName.
func getRebaseStatus(parentName, branchName string, parentOID string, errW io.Writer) statusResult {
	// parentOID is pre-fetched. We still need the merge-base between parentName and branchName.
	mergeBase, errMergeBase := git.GetMergeBase(parentName, branchName)
	if errMergeBase != nil {
		// Log the warning about failing to get merge-base.
		_, _ = fmt.Fprintf(errW, ui.Colors.WarningStyle.Render("  Warning: Could not get merge base between '%s' and '%s' to check rebase status: %v\n"), parentName, branchName, errMergeBase)
		return statusResult{"(Rebase: Error)", func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	}

	if parentOID == "" { // Can happen if parent OID fetch failed
		_, _ = fmt.Fprintf(errW, ui.Colors.WarningStyle.Render("  Warning: Provided parent OID for '%s' is empty. Cannot determine rebase status for '%s'.\n"), parentName, branchName)
		return statusResult{"(Rebase: Error)", func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	}

	needsRestack := (mergeBase != parentOID)

	if needsRestack {
		return statusResult{"(Needs Restack)", func(s string) string { return ui.Colors.WarningStyle.Render(s) }}
	} else {
		return statusResult{"(Up-to-date)", func(s string) string { return ui.Colors.SuccessStyle.Render(s) }}
	}
}

// getPrStatusDisplay fetches PR status for a given branch using an initialized GitHub client.
// It no longer handles client initialization itself.
func getPrStatusDisplay(ghClient *gh.Client, branchName string, errW io.Writer) statusResult {
	defaultRender := func(s string) string { return s }

	// If the global client isn't available (e.g., initial setup failed), return a specific status.
	if ghClient == nil {
		return statusResult{"(PR: Client N/A)", defaultRender}
	}

	prNumberKey := fmt.Sprintf("branch.%s.socle-pr-number", branchName)
	prNumberStr, errPRNum := git.GetGitConfig(prNumberKey)

	if errors.Is(errPRNum, git.ErrConfigNotFound) {
		return statusResult{"(PR: Not Submitted)", defaultRender}
	} else if errPRNum != nil {
		// Log specific error to stderr for this branch, but return a general status for display.
		_, _ = fmt.Fprintf(errW, ui.Colors.WarningStyle.Render("  Warning: Could not read PR number config for '%s': %v\n"), branchName, errPRNum)
		return statusResult{"(PR: Config Err)", func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	}

	if prNumberStr == "" {
		return statusResult{"(PR: Not Submitted)", defaultRender}
	}

	prNumber, errParsePR := strconv.Atoi(prNumberStr)
	if errParsePR != nil {
		_, _ = fmt.Fprintf(errW, ui.Colors.WarningStyle.Render("  Warning: Could not parse PR number '%s' for '%s': %v\n"), prNumberStr, branchName, errParsePR)
		return statusResult{"(PR: Invalid #)", func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	}

	// ghClient is assumed to be non-nil here due to the check at the beginning of the function.
	semanticStatus, _, errGetStatus := ghClient.GetPullRequestStatus(prNumber)
	if errGetStatus != nil {
		_, _ = fmt.Fprintf(errW, ui.Colors.WarningStyle.Render("  Warning: Could not fetch PR #%d for '%s': %v\n"), prNumber, branchName, errGetStatus)
		statusText := fmt.Sprintf("(PR #%d: API Error)", prNumber)
		return statusResult{statusText, func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	}

	prNumStr := fmt.Sprintf("#%d", prNumber)
	switch semanticStatus {
	case gh.PRStatusMerged:
		return statusResult{fmt.Sprintf("(PR %s: Merged)", prNumStr), func(s string) string { return lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Render(s) }}
	case gh.PRStatusClosed:
		return statusResult{fmt.Sprintf("(PR %s: Closed)", prNumStr), func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	case gh.PRStatusDraft:
		return statusResult{fmt.Sprintf("(PR %s: Draft)", prNumStr), func(s string) string { return ui.Colors.FaintStyle.Render(s) }}
	case gh.PRStatusOpen:
		return statusResult{fmt.Sprintf("(PR %s: Open)", prNumStr), func(s string) string { return ui.Colors.SuccessStyle.Render(s) }}
	case gh.PRStatusNotFound:
		return statusResult{fmt.Sprintf("(PR #%d: Not Found)", prNumber), func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	default:
		return statusResult{fmt.Sprintf("(PR #%d: %s)", prNumber, semanticStatus), func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	}
}
