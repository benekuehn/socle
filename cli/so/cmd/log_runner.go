package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/benekuehn/socle/cli/so/internal/cmdutils"
	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
)

// branchLogInfo holds all data needed to render a single branch entry in the log.
type branchLogInfo struct {
	branchName      string
	parentName      string
	branchNameStyle func(string) string // Function to apply style (e.g., bold)
	prText          string
	needsRebase     bool
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

// Custom styles for our list
var (
	rebaseDotStyle        = ui.Colors.DotFilledStyle
	rebaseDotWarningStyle = ui.Colors.DotWarningStyle
	prDotStyle            = ui.Colors.DotStyle
	prDotFilledStyle      = ui.Colors.DotFilledStyle
	prDotSubmittedStyle   = ui.Colors.DotStyle
	mutedStyle            = ui.Colors.MutedStyle
)

// Global map to store branch information
var branchInfoMap = make(map[string]branchLogInfo)

// Custom enumerator for our list
func branchEnumerator(items list.Items, i int) string {
	item := items.At(i)
	// Strip ANSI escape codes from the branch name
	branchName := strings.SplitN(item.Value(), " ", 2)[0]
	branchName = strings.TrimSuffix(branchName, "\x1b[0m") // Remove ANSI reset
	branchName = strings.TrimPrefix(branchName, "\x1b[1m") // Remove ANSI bold

	// Check if this is the base branch
	if strings.Contains(item.Value(), "(base)") {
		return "   " // Three spaces for base branch
	}

	// Get branch info from our map
	branchInfo, exists := branchInfoMap[branchName]
	if !exists {
		return "   " // Three spaces for branches without info
	}

	// First dot: Rebase status
	var firstDot string
	switch {
	case branchInfo.needsRebase:
		firstDot = rebaseDotWarningStyle.Render("●")
	default:
		firstDot = rebaseDotStyle.Render("●")
	}

	// Second dot: PR status
	var secondDot string
	switch {
	case strings.Contains(branchInfo.prText, "Merged"):
		secondDot = prDotFilledStyle.Render("●")
	case strings.Contains(branchInfo.prText, "Open"), strings.Contains(branchInfo.prText, "Draft"):
		secondDot = prDotSubmittedStyle.Render("●")
	default:
		secondDot = prDotStyle.Render("○")
	}

	return firstDot + " " + secondDot
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
	var ghClientInitError error
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

	if ghClientInitError != nil {
		_, _ = fmt.Fprintf(r.stderr, ui.Colors.WarningStyle.Render("Warning: GitHub client initialization failed: %v\nPR statuses may not be available.\n"), ghClientInitError)
	}

	numBranchesInStack := len(fullOrderedStack) - 1
	if numBranchesInStack < 0 {
		numBranchesInStack = 0
	}

	branchInfos := make([]branchLogInfo, 0, numBranchesInStack)

	// Pre-fetch all parent OIDs for branches in the stack to reduce git calls
	parentOIDs := make(map[string]string)
	if numBranchesInStack > 0 {
		parentNamesToFetch := make(map[string]struct{})
		for i := len(fullOrderedStack) - 1; i >= 1; i-- {
			if i > 0 {
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
				_, _ = fmt.Fprintf(r.stderr, ui.Colors.WarningStyle.Render("Warning: Could not pre-fetch some parent OIDs: %v\nRebase statuses might be affected.\n"), errFetchParentOIDs)
				if parentOIDs == nil {
					parentOIDs = make(map[string]string)
				}
			}
		}
	}

	for i := len(fullOrderedStack) - 1; i >= 1; i-- {
		branchName := fullOrderedStack[i]
		parentName := fullOrderedStack[i-1]

		parentOID, ok := parentOIDs[parentName]
		if !ok {
			parentOID = ""
			_, _ = fmt.Fprintf(r.stderr, ui.Colors.WarningStyle.Render("Warning: Parent OID for '%s' not found after pre-fetch. Rebase status for '%s' may be incorrect.\n"), parentName, branchName)
		}

		rebaseStatusResult := getRebaseStatus(parentName, branchName, parentOID, r.stderr)
		needsRebase := rebaseStatusResult.text == "(Needs Restack)"

		initialPrText := "⏳ Loading PR..."
		if ghClient == nil {
			initialPrText = "⚠️ PR client N/A"
		}

		branchInfos = append(branchInfos, branchLogInfo{
			branchName:      branchName,
			parentName:      parentName,
			needsRebase:     needsRebase,
			branchNameStyle: boldRenderer,
			prText:          initialPrText,
		})
	}

	// Create a new list
	l := list.New()

	// Clear the global map
	branchInfoMap = make(map[string]branchLogInfo)

	// Add branches to the list
	for _, info := range branchInfos {
		// Format status text
		var statusText string
		if info.needsRebase {
			statusText = "(needs restack"
		} else {
			statusText = "(up-to-date"
		}

		// Add PR status using switch
		switch {
		case strings.Contains(info.prText, "Draft"):
			statusText += ", pr drafted"
		case strings.Contains(info.prText, "Merged"):
			statusText += ", pr merged"
		case strings.Contains(info.prText, "Open"):
			statusText += ", pr open"
		case strings.Contains(info.prText, "Closed"):
			statusText += ", pr closed"
		default:
			statusText += ", no PR submitted"
		}
		statusText += ")"

		// Store branch info in our map
		branchInfoMap[info.branchName] = info

		// Add the branch to the list with its status, only making the branch name bold
		boldBranchName := lipgloss.NewStyle().Bold(true).Render(info.branchName)
		mutedStatus := mutedStyle.Render(statusText)
		l.Item(boldBranchName + " " + mutedStatus)
	}

	// Add the base branch at the end with muted style
	mutedBase := mutedStyle.Render(baseBranch + " (base)")
	l.Item(mutedBase)

	// Configure the list with custom styles
	l = l.Enumerator(branchEnumerator).
		EnumeratorStyle(lipgloss.NewStyle().MarginRight(1).Bold(true)).
		ItemStyle(lipgloss.NewStyle().MarginRight(1))

	// Print the list with padding
	paddedList := lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingTop(1).
		PaddingBottom(1).
		Render(l.String())
	_, _ = fmt.Fprintln(r.stdout, paddedList)

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
