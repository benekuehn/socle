package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"

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

type showCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
}

func (r *showCmdRunner) run(ctx context.Context) error {
	currentBranch, stack, baseBranch, err := git.GetCurrentStackInfo()
	handled, processedErr := cmdutils.HandleStartupError(err, currentBranch, r.stdout, r.stderr)
	if processedErr != nil {
		return processedErr
	}
	if handled {
		return nil
	}
	fmt.Fprintf(r.stdout, "Current branch: %s (Stack base: %s)\n", currentBranch, baseBranch)

	var ghClient *gh.Client
	var ghClientErr error
	var ghClientInitialized bool = false

	fmt.Fprintln(r.stdout, "\nCurrent Stack Status:")
	baseMarker := ""
	if baseBranch == currentBranch {
		baseMarker = " *"
	}
	fmt.Fprintf(r.stdout, "  %s (base)%s\n", baseBranch, baseMarker)

	ancestorNeedsRestack := false

	for i := 1; i < len(stack); i++ {
		branchName := stack[i]
		parentName := stack[i-1]
		rebaseStatus := getRebaseStatus(parentName, branchName, r.stderr)

		effectiveNeedsRestack := (rebaseStatus.text == "(Needs Restack)") || ancestorNeedsRestack
		if effectiveNeedsRestack && rebaseStatus.text == "(Up-to-date)" {
			rebaseStatus = statusResult{"(Needs Restack)", func(s string) string { return ui.Colors.WarningStyle.Render(s) }}
		}
		ancestorNeedsRestack = effectiveNeedsRestack

		prStatus := getPrStatusDisplay(ctx, &ghClient, &ghClientErr, &ghClientInitialized, branchName, r.stderr)
		marker := ""
		if branchName == currentBranch {
			marker = " *"
		}

		fmt.Fprintf(r.stdout, "  -> %s %s %s%s\n",
			branchName,
			rebaseStatus.render(rebaseStatus.text),
			prStatus.render(prStatus.text),
			marker,
		)
	}

	return nil
}

// getRebaseStatus determines the rebase status string and render function
func getRebaseStatus(parentName, branchName string, errW io.Writer) statusResult {
	needsRestack, errCheck := git.NeedsRestack(parentName, branchName)

	if errCheck != nil {
		fmt.Fprintf(errW, ui.Colors.WarningStyle.Render("  Warning: Could not check restack status for '%s': %v\n"), branchName, errCheck)
		return statusResult{"(Rebase: Error)", func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	} else if needsRestack {
		return statusResult{"(Needs Restack)", func(s string) string { return ui.Colors.WarningStyle.Render(s) }}
	} else {
		return statusResult{"(Up-to-date)", func(s string) string { return ui.Colors.SuccessStyle.Render(s) }}
	}
}

// getPrStatusDisplay reads config, calls gh service, and maps result to display statusResult
func getPrStatusDisplay(ctx context.Context, ghClient **gh.Client, clientErr *error, clientInitialized *bool, branchName string, errW io.Writer) statusResult {
	defaultRender := func(s string) string { return s }

	prNumberKey := fmt.Sprintf("branch.%s.socle-pr-number", branchName)
	prNumberStr, errPRNum := git.GetGitConfig(prNumberKey)

	if errors.Is(errPRNum, git.ErrConfigNotFound) {
		return statusResult{"(PR: Not Submitted)", defaultRender}
	} else if errPRNum != nil {
		fmt.Fprintf(errW, ui.Colors.WarningStyle.Render("  Warning: Could not read PR number config for '%s': %v\n"), branchName, errPRNum)
		return statusResult{"(PR: Config Err)", func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	}

	if prNumberStr == "" {
		return statusResult{"(PR: Not Submitted)", defaultRender}
	}

	prNumber, errParsePR := strconv.Atoi(prNumberStr)
	if errParsePR != nil {
		fmt.Fprintf(errW, ui.Colors.WarningStyle.Render("  Warning: Could not parse PR number '%s' for '%s': %v\n"), prNumberStr, branchName, errParsePR)
		return statusResult{"(PR: Invalid #)", func(s string) string { return ui.Colors.FailureStyle.Render(s) }}
	}

	if !*clientInitialized {
		*clientInitialized = true
		remoteName := "origin"
		remoteURL, errURL := git.GetRemoteURL(remoteName)
		if errURL != nil {
			*clientErr = fmt.Errorf("cannot get remote URL '%s': %w", remoteName, errURL)
		} else {
			owner, repoName, errParse := git.ParseOwnerAndRepo(remoteURL)
			if errParse != nil {
				*clientErr = fmt.Errorf("cannot parse owner/repo from '%s': %w", remoteURL, errParse)
			} else {
				client, errCli := gh.NewClient(ctx, owner, repoName)
				if errCli != nil {
					*clientErr = fmt.Errorf("GitHub client init failed: %w", errCli)
				} else {
					*ghClient = client
				}
			}
		}
		if *clientErr != nil {
			fmt.Fprintf(errW, ui.Colors.WarningStyle.Render("Warning: Cannot fetch PR status: %v\n"), *clientErr)
		}
	}

	if *ghClient == nil {
		return statusResult{"(PR: Login/Setup Needed)", func(s string) string { return ui.Colors.WarningStyle.Render(s) }}
	}

	semanticStatus, _, errGetStatus := (*ghClient).GetPullRequestStatus(prNumber)
	if errGetStatus != nil {
		fmt.Fprintf(errW, ui.Colors.WarningStyle.Render("  Warning: Could not fetch PR #%d for '%s': %v\n"), prNumber, branchName, errGetStatus)
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
