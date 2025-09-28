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
	prURL           string
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
	rebaseDotStyle        = ui.Colors.SuccessStyle
	rebaseDotWarningStyle = ui.Colors.WarningStyle
	prDotDefaultStyle     = ui.Colors.InfoStyle
	prDotMergedStyle      = ui.Colors.SuccessStyle
	prDotSubmittedStyle   = ui.Colors.InfoStyle
	prDotClosedStyle      = ui.Colors.FailureStyle
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
		secondDot = prDotMergedStyle.Render("●")
	case strings.Contains(branchInfo.prText, "Open"), strings.Contains(branchInfo.prText, "Draft"):
		secondDot = prDotSubmittedStyle.Render("●")
	case strings.Contains(branchInfo.prText, "Closed"):
		secondDot = prDotClosedStyle.Render("●")
	default:
		secondDot = prDotDefaultStyle.Render("○")
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

	// 4. Handle multiple stacks case or empty stacks
	if stackInfo == nil {
		_, _ = fmt.Fprintf(r.stdout, "Error: Could not get stack information.\n")
		return nil
	}
	
	// If FullStack is nil, it means there are multiple stacks from the base
	// But only show multiple stacks if we're actually ON the base branch
	if stackInfo.FullStack == nil && currentBranch == stackInfo.BaseBranch {
		return r.displayMultipleStacks(ctx, stackInfo.BaseBranch, currentBranch)
	}
	
	// Determine which stack to use for display
	var stackToDisplay []string
	if stackInfo.FullStack != nil {
		stackToDisplay = stackInfo.FullStack
	} else {
		// FullStack is nil but we're not on base - use CurrentStack instead
		stackToDisplay = stackInfo.CurrentStack
	}
	
	if len(stackToDisplay) <= 1 {
		_, _ = fmt.Fprintf(r.stdout, "Currently on the base branch '%s'.\n", currentBranch)
		return nil
	}

	// Continue with the regular log command logic
	numBranchesInStack := len(stackToDisplay) - 1
	if numBranchesInStack < 0 {
		numBranchesInStack = 0
	}

	// Pre-fetch all parent OIDs for branches in the stack to reduce git calls
	parentOIDs := make(map[string]string)
	if numBranchesInStack > 0 {
		parentNamesToFetch := make(map[string]struct{})
		for i := len(stackToDisplay) - 1; i >= 1; i-- {
			if i > 0 {
				parentNamesToFetch[stackToDisplay[i-1]] = struct{}{}
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

	var ghClient gh.ClientInterface
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
			client, errCli := gh.CreateClient(ctx, owner, repoName)
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
	for i := len(stackToDisplay) - 1; i >= 1; i-- {
		branchName := stackToDisplay[i]
		parentName := stackToDisplay[i-1]
		parentOID := parentOIDs[parentName]

		wg.Add(1)
		go func(branch, parent string, parentOID string) {
			defer wg.Done()

			// Get PR status
			prNumber, err := git.GetStoredPRNumber(branch)
			var prStatus string
			var prURL string
			if err != nil || prNumber == 0 {
				prStatus = gh.PRStatusNotFound
			} else if ghClient != nil {
				prStatus, prURL, err = ghClient.GetPullRequestStatus(prNumber)
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
				prURL:           prURL,
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
	for i := len(stackToDisplay) - 1; i >= 1; i-- {
		branchName := stackToDisplay[i]
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

		// Add PR status with hyperlink if URL exists
		if info.prURL != "" {
			// OSC 8 escape sequence for hyperlinks
			prStatus := ""
			switch info.prText {
			case gh.PRStatusDraft:
				prStatus = "pr drafted"
			case gh.PRStatusMerged:
				prStatus = "pr merged"
			case gh.PRStatusOpen:
				prStatus = "pr open"
			case gh.PRStatusClosed:
				prStatus = "pr closed"
			case gh.PRStatusAPIError:
				prStatus = "pr check failed"
			case gh.PRStatusNotFound:
				prStatus = "no PR submitted"
			default:
				prStatus = "no PR submitted"
			}
			prLink := fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", info.prURL, prStatus)
			statusText += ", " + prLink
		} else {
			// No PR URL, just add the status text
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

func (r *logCmdRunner) displayMultipleStacks(ctx context.Context, baseBranch, currentBranch string) error {
	// Get available stacks from the base
	availableStacks, err := git.GetAvailableStacksFromBase(baseBranch)
	if err != nil {
		return fmt.Errorf("failed to get available stacks from base '%s': %w", baseBranch, err)
	}

	if len(availableStacks) == 0 {
		_, _ = fmt.Fprintf(r.stdout, "No stacks found starting from base branch '%s'.\n", baseBranch)
		return nil
	}

	// Display header with count
	stackCount := len(availableStacks)
	if stackCount == 1 {
		_, _ = fmt.Fprintf(r.stdout, "1 stack from base '%s':\n\n", baseBranch)
	} else {
		_, _ = fmt.Fprintf(r.stdout, "%d stacks from base '%s':\n\n", stackCount, baseBranch)
	}

	// Display each stack with detailed info
	for _, stack := range availableStacks {
		err := r.displaySingleStackDetailed(ctx, stack, currentBranch)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *logCmdRunner) displaySingleStackDetailed(ctx context.Context, stack []string, currentBranch string) error {
	if len(stack) <= 1 {
		// Stack with only base branch
		mutedBase := mutedStyle.Render(stack[0] + " (base, no branches)")
		_, _ = fmt.Fprintf(r.stdout, "     %s\n\n", mutedBase)
		return nil
	}

	// Get GitHub client for PR status (same setup as main log)
	var ghClient gh.ClientInterface
	remoteName := "origin"
	remoteURL, errURL := git.GetRemoteURL(remoteName)
	if errURL != nil {
		// No GitHub client available
	} else {
		owner, repoName, errParse := git.ParseOwnerAndRepo(remoteURL)
		if errParse != nil {
			// No GitHub client available
		} else {
			client, errCli := gh.CreateClient(ctx, owner, repoName)
			if errCli == nil {
				ghClient = client
			}
		}
	}

	// Pre-fetch parent OIDs for rebase status checks
	parentOIDs := make(map[string]string)
	if len(stack) > 1 {
		parentNamesToFetch := make(map[string]struct{})
		for i := len(stack) - 1; i >= 1; i-- {
			if i > 0 {
				parentNamesToFetch[stack[i-1]] = struct{}{}
			}
		}
		uniqueParentNames := make([]string, 0, len(parentNamesToFetch))
		for name := range parentNamesToFetch {
			uniqueParentNames = append(uniqueParentNames, name)
		}

		if len(uniqueParentNames) > 0 {
			var errFetchParentOIDs error
			parentOIDs, errFetchParentOIDs = git.GetMultipleBranchCommits(uniqueParentNames)
			if errFetchParentOIDs != nil && parentOIDs == nil {
				parentOIDs = make(map[string]string)
			}
		}
	}

	// Process branches in parallel to get PR and rebase status
	branchInfos := make([]branchLogInfo, 0, len(stack)-1)
	var wg sync.WaitGroup
	results := make(map[string]branchLogInfo)
	var mu sync.Mutex

	for i := len(stack) - 1; i >= 1; i-- {
		branchName := stack[i]
		parentName := stack[i-1]
		parentOID := parentOIDs[parentName]

		wg.Add(1)
		go func(branch, parent string, parentOID string) {
			defer wg.Done()

			// Get PR status
			prNumber, err := git.GetStoredPRNumber(branch)
			var prStatus string
			var prURL string
			if err != nil || prNumber == 0 {
				prStatus = gh.PRStatusNotFound
			} else if ghClient != nil {
				prStatus, prURL, err = ghClient.GetPullRequestStatus(prNumber)
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
				prURL:           prURL,
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
	for i := len(stack) - 1; i >= 1; i-- {
		branchName := stack[i]
		branchInfos = append(branchInfos, results[branchName])
	}

	// Create a temporary list for this stack
	l := list.New()
	stackBranchInfoMap := make(map[string]branchLogInfo)

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

		// Add PR status with hyperlink if URL exists
		if info.prURL != "" {
			// OSC 8 escape sequence for hyperlinks
			prStatus := ""
			switch info.prText {
			case gh.PRStatusDraft:
				prStatus = "pr drafted"
			case gh.PRStatusMerged:
				prStatus = "pr merged"
			case gh.PRStatusOpen:
				prStatus = "pr open"
			case gh.PRStatusClosed:
				prStatus = "pr closed"
			case gh.PRStatusAPIError:
				prStatus = "pr check failed"
			case gh.PRStatusNotFound:
				prStatus = "no PR submitted"
			default:
				prStatus = "no PR submitted"
			}
			prLink := fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", info.prURL, prStatus)
			statusText += ", " + prLink
		} else {
			// No PR URL, just add the status text
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
		}
		statusText += ")"

		stackBranchInfoMap[info.branchName] = info

		boldBranchName := lipgloss.NewStyle().Bold(true).Render(info.branchName)
		mutedStatus := mutedStyle.Render(statusText)
		l.Item(boldBranchName + " " + mutedStatus)
	}

	mutedBase := mutedStyle.Render(stack[0] + " (base)")
	l.Item(mutedBase)

	// Create custom enumerator for this stack
	stackEnumerator := func(items list.Items, i int) string {
		item := items.At(i)
		// Strip ANSI escape codes from the branch name
		branchName := strings.SplitN(item.Value(), " ", 2)[0]
		branchName = strings.TrimSuffix(branchName, "\x1b[0m")
		branchName = strings.TrimPrefix(branchName, "\x1b[1m")

		// Base or without info need space for alignment
		branchInfo, exists := stackBranchInfoMap[branchName]
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
			secondDot = prDotMergedStyle.Render("●")
		case strings.Contains(branchInfo.prText, "Open"), strings.Contains(branchInfo.prText, "Draft"):
			secondDot = prDotSubmittedStyle.Render("●")
		case strings.Contains(branchInfo.prText, "Closed"):
			secondDot = prDotClosedStyle.Render("●")
		default:
			secondDot = prDotDefaultStyle.Render("○")
		}

		return firstDot + " " + secondDot
	}

	l = l.Enumerator(stackEnumerator).
		EnumeratorStyle(lipgloss.NewStyle().MarginRight(1).Bold(true)).
		ItemStyle(lipgloss.NewStyle().MarginRight(1))

	paddedList := lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingTop(0).
		PaddingBottom(1).
		Render(l.String())
	_, _ = fmt.Fprintln(r.stdout, paddedList)

	return nil
}

func (r *logCmdRunner) displaySingleStack(stack []string, currentBranch string) {
	if len(stack) <= 1 {
		// Stack with only base branch - show it indented to match other output
		_, _ = fmt.Fprintf(r.stdout, "  %s (base, no branches)\n\n", stack[0])
		return
	}

	// Display branches from top to bottom (like so log), with proper spacing
	// Skip the base branch (index 0) and show stack branches in reverse order
	for i := len(stack) - 1; i >= 1; i-- {
		branch := stack[i]
		
		// Use the same dot pattern as so log
		branchDot := "○"  // Empty circle for branch status (could be enhanced later with actual status)
		statusDot := "●"  // Filled circle for git status 
		
		// Check if this is the current branch and style accordingly
		isCurrentBranch := branch == currentBranch
		var branchText string
		
		if isCurrentBranch {
			// Highlight current branch similar to log output
			branchText = ui.Colors.InfoStyle.Render(fmt.Sprintf("%s %s %s", statusDot, branchDot, branch))
		} else {
			branchText = fmt.Sprintf("%s %s %s", statusDot, branchDot, branch)
		}
		
		_, _ = fmt.Fprintf(r.stdout, "  %s\n", branchText)
	}
	
	// Display base branch at the bottom with consistent spacing
	baseBranch := stack[0]
	_, _ = fmt.Fprintf(r.stdout, "  %s (base)\n", baseBranch)
	_, _ = fmt.Fprintln(r.stdout) // Add blank line after each stack
}
