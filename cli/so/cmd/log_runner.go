package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/benekuehn/socle/cli/so/internal/cmdutils"
	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

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

	// Get the full stack based on the current stack's info
	fullOrderedStack, _, err := git.GetFullStack(stackToCurrent)
	if err != nil {
		// Potentially handle this error more gracefully, e.g., by falling back to stackToCurrent
		// For now, let's report it and exit, as it might indicate broken tracking.
		_, _ = fmt.Fprintf(r.stderr, ui.Colors.FailureStyle.Render("Error: Could not determine the full stack: %v\n"), err)
		return err
	}

	// If GetFullStack returns an empty or only base stack for some reason, but GetCurrentStackInfo succeeded,
	// it might mean we are on the base branch itself and it's the only one.
	// The loop condition `i >= 1` will handle fullOrderedStack having only the base.

	_, _ = fmt.Fprintln(r.stdout) // Add a blank line at the beginning

	boldStyle := lipgloss.NewStyle().Bold(true)

	var ghClient *gh.Client
	var ghClientErr error
	var ghClientInitialized = false

	// Iterate from the top of the fullOrderedStack down to the branch above the base
	// fullOrderedStack is [base, child1, ..., topMost]
	// So, len(fullOrderedStack)-1 is the topMost, fullOrderedStack[0] is the base.
	for i := len(fullOrderedStack) - 1; i >= 1; i-- {
		branchName := fullOrderedStack[i]
		parentName := fullOrderedStack[i-1] // The parent is the one below it in the fullOrderedStack

		branchIndicator := "◯"
		if branchName == currentBranch { // currentBranch is from GetCurrentStackInfo
			branchIndicator = "◉"
		}
		_, _ = fmt.Fprintf(r.stdout, "    %s  %s\n", branchIndicator, boldStyle.Render(branchName))

		rebaseStatusResult := getRebaseStatus(parentName, branchName, r.stderr)
		// Pass ghClientErr and ghClientInitialized by pointer to allow getPrStatusDisplay to update them
		prStatusResult := getPrStatusDisplay(ctx, &ghClient, &ghClientErr, &ghClientInitialized, branchName, r.stderr)

		var rebaseText string
		switch rebaseStatusResult.text {
		case "(Needs Restack)":
			rebaseText = "🚧 Needs restack"
		case "(Up-to-date)":
			rebaseText = "🤝 Up-to-date"
		default:
			rebaseText = "⚠️ Status error"
		}

		var prText string
		if strings.Contains(prStatusResult.text, ": Merged)") {
			prText = "✅ PR merged"
		} else if strings.Contains(prStatusResult.text, ": Closed)") {
			prText = "🚫 PR closed"
		} else if strings.Contains(prStatusResult.text, ": Draft)") {
			prText = "📝 PR draft"
		} else if strings.Contains(prStatusResult.text, ": Open)") {
			prText = "➡️ PR open"
		} else if strings.Contains(prStatusResult.text, "Not Submitted") {
			prText = "⚪ PR not submitted"
		} else if strings.Contains(prStatusResult.text, "Login/Setup Needed") {
			prText = "🔑 GH Login Needed"
		} else {
			prText = "⚠️ PR status error"
		}

		_, _ = fmt.Fprintf(r.stdout, "    │  %s | %s\n", rebaseText, prText)

		if i > 1 {
			_, _ = fmt.Fprintln(r.stdout, "    │")
		}
	}

	_, _ = fmt.Fprintln(r.stdout, "  ╭─╯")
	// baseBranch is determined by GetCurrentStackInfo, which should be consistent with fullOrderedStack[0]
	_, _ = fmt.Fprintf(r.stdout, "   ~ %s\n", boldStyle.Render(baseBranch))

	_, _ = fmt.Fprintln(r.stdout) // Add a blank line at the end

	return nil
}

func getRebaseStatus(parentName, branchName string, errW io.Writer) statusResult {
	needsRestack, errCheck := git.NeedsRestack(parentName, branchName)

	if errCheck != nil {
		_, _ = fmt.Fprintf(errW, ui.Colors.WarningStyle.Render("  Warning: Could not check restack status for '%s': %v\n"), branchName, errCheck)
		return statusResult{"(Rebase: Error)", func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	} else if needsRestack {
		return statusResult{"(Needs Restack)", func(s string) string { return ui.Colors.WarningStyle.Render(s) }}
	} else {
		return statusResult{"(Up-to-date)", func(s string) string { return ui.Colors.SuccessStyle.Render(s) }}
	}
}

func getPrStatusDisplay(ctx context.Context, ghClient **gh.Client, clientErr *error, clientInitialized *bool, branchName string, errW io.Writer) statusResult {
	defaultRender := func(s string) string { return s }

	prNumberKey := fmt.Sprintf("branch.%s.socle-pr-number", branchName)
	prNumberStr, errPRNum := git.GetGitConfig(prNumberKey)

	if errors.Is(errPRNum, git.ErrConfigNotFound) {
		return statusResult{"(PR: Not Submitted)", defaultRender}
	} else if errPRNum != nil {
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

	if !*clientInitialized {
		*clientInitialized = true // Mark that we've attempted initialization.
		// Use the clientErr passed by pointer to store any initialization error.
		// The ghClient itself is also passed by pointer and will be updated.

		remoteName := "origin" // Or make configurable
		remoteURL, errURL := git.GetRemoteURL(remoteName)
		if errURL != nil {
			*clientErr = fmt.Errorf("cannot get remote URL '%s': %w", remoteName, errURL)
		} else {
			owner, repoName, errParse := git.ParseOwnerAndRepo(remoteURL)
			if errParse != nil {
				*clientErr = fmt.Errorf("cannot parse owner/repo from '%s': %w", remoteURL, errParse)
			} else {
				// Attempt to create a new client
				client, errCli := gh.NewClient(ctx, owner, repoName)
				if errCli != nil {
					*clientErr = fmt.Errorf("GitHub client init failed: %w", errCli)
				} else {
					*ghClient = client // Update the shared client instance
				}
			}
		}
		// If there was an error during initialization, report it (once).
		if *clientErr != nil {
			_, _ = fmt.Fprintf(errW, ui.Colors.WarningStyle.Render("Warning: Cannot fetch PR status: %v\n"), *clientErr)
		}
	}

	// Check if client is still nil (e.g. init failed) or if there was a stored clientErr from the first attempt.
	if *ghClient == nil {
		// If client is nil and we have a clientErr, it means init failed on its first attempt.
		// If client is nil and no clientErr, it implies some other issue (should not happen if logic is correct).
		return statusResult{"(PR: Login/Setup Needed)", func(s string) string { return ui.Colors.WarningStyle.Render(s) }}
	}

	// ghClient should be usable now (or was already).
	semanticStatus, _, errGetStatus := (*ghClient).GetPullRequestStatus(prNumber)
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
