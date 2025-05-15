package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
)

// Define rebase status constants
type RebaseStatus string

const (
	RebaseStatusNeedsRestack RebaseStatus = "Needs Restack"
	RebaseStatusUpToDate     RebaseStatus = "Up-to-date"
	RebaseStatusError        RebaseStatus = "Error"
)

type branchLogInfo struct {
	branchName      string
	parentName      string
	branchNameStyle func(string) string
	prText          string
	rebaseStatus    statusResult
}

type statusResult struct {
	status RebaseStatus
	render func(string) string
}

type logCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
}

var (
	rebaseDotStyle        = ui.Colors.DotFilledStyle
	rebaseDotWarningStyle = ui.Colors.DotWarningStyle
	prDotStyle            = ui.Colors.DotStyle
	prDotFilledStyle      = ui.Colors.DotFilledStyle
	prDotSubmittedStyle   = ui.Colors.DotStyle
	mutedStyle            = ui.Colors.MutedStyle
)

var branchInfoMap = make(map[string]branchLogInfo)

func branchEnumerator(items list.Items, i int) string {
	item := items.At(i)
	// Strip ANSI escape codes from the branch name
	branchName := strings.SplitN(item.Value(), " ", 2)[0]
	branchName = strings.TrimSuffix(branchName, "\x1b[0m")
	branchName = strings.TrimPrefix(branchName, "\x1b[1m")

	// Base or without info need space for alignment
	branchInfo, exists := branchInfoMap[branchName]
	if !exists || strings.Contains(item.Value(), "(base)") {
		return "   " // Three spaces for branches without info
	}

	// First dot: Rebase status
	var firstDot string
	switch branchInfo.rebaseStatus.status {
	case RebaseStatusNeedsRestack:
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
	// 1. Get current branch first (best effort, for error handling)
	currentBranch, _ := git.GetCurrentBranch()

	// 2. Get stack info
	stackInfo, err := git.GetStackInfo()

	// 3. Handle specific error cases for log command
	if err != nil {
		if strings.Contains(err.Error(), "not tracked by socle") {
			// For the log command, we should print the message ourselves to match test expectations
			_, _ = fmt.Fprintf(r.stdout, "Branch '%s' is not currently tracked by socle.\n", currentBranch)
			_, _ = fmt.Fprintln(r.stdout, "Use 'so track' to associate it with a parent branch and start a stack.")
			return nil // Return nil to prevent additional error handling
		}
		return err // For other errors, just return them
	}

	// 4. If we get here and have a nil stack or only base branch, print the base branch message
	if stackInfo == nil || len(stackInfo.FullStack) <= 1 {
		_, _ = fmt.Fprintf(r.stdout, "Currently on the base branch '%s'.\n", currentBranch)
		return nil
	}

	// Continue with the regular log command logic
	numBranchesInStack := len(stackInfo.FullStack) - 1
	if numBranchesInStack < 0 {
		numBranchesInStack = 0
	}

	// Pre-fetch all parent OIDs for branches in the stack to reduce git calls
	parentOIDs := make(map[string]string)
	if numBranchesInStack > 0 {
		parentNamesToFetch := make(map[string]struct{})
		for i := len(stackInfo.FullStack) - 1; i >= 1; i-- {
			if i > 0 {
				parentNamesToFetch[stackInfo.FullStack[i-1]] = struct{}{}
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

	// Process branches in parallel
	branchInfos := make([]branchLogInfo, 0, numBranchesInStack)
	var wg sync.WaitGroup
	results := make(map[string]branchLogInfo)
	var mu sync.Mutex

	// Process each branch in parallel
	for i := len(stackInfo.FullStack) - 1; i >= 1; i-- {
		branchName := stackInfo.FullStack[i]
		parentName := stackInfo.FullStack[i-1]
		parentOID := parentOIDs[parentName]

		wg.Add(1)
		go func(branch, parent string, parentOID string) {
			defer wg.Done()

			// Get PR status
			prNumber, err := git.GetStoredPRNumber(branch)
			var prStatus string
			if err != nil || prNumber == 0 {
				prStatus = gh.PRStatusNotFound
			} else if ghClient != nil {
				prStatus, _, err = ghClient.GetPullRequestStatus(prNumber)
				if err != nil {
					prStatus = gh.PRStatusAPIError
				}
			} else {
				prStatus = gh.PRStatusAPIError
			}

			// Get rebase status
			rebaseStatusResult := getRebaseStatus(parent, branch, parentOID, r.stderr)

			// Create branch info
			info := branchLogInfo{
				branchName:      branch,
				parentName:      parent,
				branchNameStyle: func(s string) string { return lipgloss.NewStyle().Bold(true).Render(s) },
				prText:          prStatus,
				rebaseStatus:    rebaseStatusResult,
			}

			mu.Lock()
			results[branch] = info
			mu.Unlock()
		}(branchName, parentName, parentOID)
	}

	// Wait for all checks to complete
	wg.Wait()

	// Process branches in order to maintain the original order
	for i := len(stackInfo.FullStack) - 1; i >= 1; i-- {
		branchName := stackInfo.FullStack[i]
		branchInfos = append(branchInfos, results[branchName])
	}

	// Create a new list
	l := list.New()

	// Clear the global map
	branchInfoMap = make(map[string]branchLogInfo)

	for _, info := range branchInfos {
		var statusText string
		switch info.rebaseStatus.status {
		case RebaseStatusNeedsRestack:
			statusText = "(needs restack"
		case RebaseStatusError:
			statusText = "(rebase check failed"
		default:
			statusText = "(up-to-date"
		}

		switch info.prText {
		case gh.PRStatusDraft:
			statusText += ", pr drafted"
		case gh.PRStatusMerged:
			statusText += ", pr merged"
		case gh.PRStatusOpen:
			statusText += ", pr open"
		case gh.PRStatusClosed:
			statusText += ", pr closed"
		case gh.PRStatusAPIError:
			statusText += ", pr check failed"
		case gh.PRStatusNotFound:
			statusText += ", no PR submitted"
		default:
			statusText += ", no PR submitted"
		}
		statusText += ")"

		branchInfoMap[info.branchName] = info

		boldBranchName := lipgloss.NewStyle().Bold(true).Render(info.branchName)
		mutedStatus := mutedStyle.Render(statusText)
		l.Item(boldBranchName + " " + mutedStatus)
	}

	mutedBase := mutedStyle.Render(stackInfo.BaseBranch + " (base)")
	l.Item(mutedBase)

	l = l.Enumerator(branchEnumerator).
		EnumeratorStyle(lipgloss.NewStyle().MarginRight(1).Bold(true)).
		ItemStyle(lipgloss.NewStyle().MarginRight(1))

	paddedList := lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingTop(1).
		PaddingBottom(1).
		Render(l.String())
	_, _ = fmt.Fprintln(r.stdout, paddedList)

	return nil
}

// It calculates needsRestack by comparing parentOID with the merge-base of parentName and branchName.
func getRebaseStatus(parentName, branchName string, parentOID string, errW io.Writer) statusResult {
	// parentOID is pre-fetched. We still need the merge-base between parentName and branchName.
	mergeBase, errMergeBase := git.GetMergeBase(parentName, branchName)
	if errMergeBase != nil {
		// Log the warning about failing to get merge-base.
		_, _ = fmt.Fprintf(errW, ui.Colors.WarningStyle.Render("  Warning: Could not get merge base between '%s' and '%s' to check rebase status: %v\n"), parentName, branchName, errMergeBase)
		return statusResult{RebaseStatusError, func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	}

	if parentOID == "" { // Can happen if parent OID fetch failed
		_, _ = fmt.Fprintf(errW, ui.Colors.WarningStyle.Render("  Warning: Provided parent OID for '%s' is empty. Cannot determine rebase status for '%s'.\n"), parentName, branchName)
		return statusResult{RebaseStatusError, func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	}

	needsRestack := (mergeBase != parentOID)

	if needsRestack {
		return statusResult{RebaseStatusNeedsRestack, func(s string) string { return ui.Colors.WarningStyle.Render(s) }}
	} else {
		return statusResult{RebaseStatusUpToDate, func(s string) string { return ui.Colors.SuccessStyle.Render(s) }}
	}
}
