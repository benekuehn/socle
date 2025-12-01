package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
	// Assuming HandleSurveyInterrupt is there or moved to ui
)

type trackCmdRunner struct {
	ctx    context.Context
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader

	discoverRemote bool

	// Test flags
	testSelectedParent string
	testAssumeBase     string
}

func (r *trackCmdRunner) run() error {
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	// Basic check: Don't track base branches like main/master/develop
	knownBases := map[string]bool{"main": true, "master": true, "develop": true} // TODO: Configurable
	if knownBases[currentBranch] {
		return fmt.Errorf("cannot track a base branch ('%s') itself", currentBranch)
	}

	// 2. Check if already tracked
	parentConfigKey := fmt.Sprintf("branch.%s.socle-parent", currentBranch)
	existingParent, errGetParent := git.GetGitConfig(parentConfigKey)
	if errGetParent == nil && existingParent != "" {
		baseConfigKey := fmt.Sprintf("branch.%s.socle-base", currentBranch)
		existingBase, _ := git.GetGitConfig(baseConfigKey)
		_, _ = fmt.Fprintf(r.stdout, "Branch '%s' is already tracked.\n", currentBranch)
		_, _ = fmt.Fprintf(r.stdout, "  Parent: %s\n", existingParent)
		_, _ = fmt.Fprintf(r.stdout, "  Base:   %s\n", existingBase)
		return nil
	} else if errGetParent != nil && !errors.Is(errGetParent, git.ErrConfigNotFound) {
		return fmt.Errorf("failed to check tracking status for branch '%s': %w", currentBranch, errGetParent) // Use actual error
	}

	// 3. Get potential parent branches
	allBranches, err := git.GetLocalBranches()
	if err != nil {
		return fmt.Errorf("failed to list local branches: %w", err)
	}

	potentialParents := []string{}
	for _, b := range allBranches {
		if b != currentBranch {
			potentialParents = append(potentialParents, b)
		}
	}

	if len(potentialParents) == 0 {
		return fmt.Errorf("no other local branches found to select as a parent")
	}

	var discovery *remoteDiscoveryResult
	if r.discoverRemote {
		var err error
		discovery, err = r.discoverRemoteInfo(currentBranch)
		if err != nil {
			_, _ = fmt.Fprintln(r.stderr, ui.Colors.WarningStyle.Render(fmt.Sprintf("Remote discovery skipped: %v", err)))
		} else if discovery != nil {
			r.logger.Debug("Remote discovery successful", "remoteName", discovery.remoteName)
			r.emitDiscoverySummary(currentBranch, discovery)
			if discovery.prBase != "" && discovery.prBase != currentBranch {
				found := false
				for _, candidate := range potentialParents {
					if candidate == discovery.prBase {
						found = true
						break
					}
				}
				if !found && knownBases[discovery.prBase] {
					potentialParents = append(potentialParents, discovery.prBase)
				}
			}
		}
	}

	// 4. Prompt user to select parent
	selectedParent := ""
	var defaultParent string
	if discovery != nil && discovery.prBase != "" && discovery.prBase != currentBranch {
		for _, candidate := range potentialParents {
			if candidate == discovery.prBase {
				defaultParent = discovery.prBase
				break
			}
		}
	}

	if r.testSelectedParent != "" {
		r.logger.Debug("Using parent branch from test flag", "testParent", r.testSelectedParent)
		found := false
		for _, p := range potentialParents {
			if p == r.testSelectedParent {
				found = true
				break
			}
		}
		if !found && !knownBases[r.testSelectedParent] {
			return fmt.Errorf("invalid test parent '%s': not found in potential parents %v or known bases", r.testSelectedParent, potentialParents)
		}
		selectedParent = r.testSelectedParent
	} else {
		// Use runner's stdio
		surveyOpts := survey.WithStdio(r.stdin.(*os.File), r.stderr.(*os.File), r.stderr.(*os.File))
		r.logger.Debug("Prompting user for parent branch")
		prompt := &survey.Select{Message: fmt.Sprintf("Select the parent branch for '%s':", currentBranch), Options: potentialParents}
		if defaultParent != "" {
			prompt.Default = defaultParent
		}
		err := survey.AskOne(prompt, &selectedParent, surveyOpts)
		if err != nil {
			// Use ui.HandleSurveyInterrupt which should be in internal/ui
			return ui.HandleSurveyInterrupt(err, "Track command cancelled.")
		}
		r.logger.Debug("Parent selected via prompt", "selectedParent", selectedParent)
	}

	// 5. Determine and store base branch
	selectedBase := ""
	if knownBases[selectedParent] {
		selectedBase = selectedParent
	} else {
		parentBaseKey := fmt.Sprintf("branch.%s.socle-base", selectedParent)
		inheritedBase, errGetBase := git.GetGitConfig(parentBaseKey)
		if errGetBase == nil && inheritedBase != "" {
			selectedBase = inheritedBase
			r.logger.Debug("Inheriting base from tracked parent", "base", selectedBase, "parent", selectedParent)
		} else if errors.Is(errGetBase, git.ErrConfigNotFound) {
			r.logger.Debug("Selected parent is not tracked", "parent", selectedParent)
			if r.testAssumeBase != "" {
				r.logger.Debug("Using base from test flag", "testBase", r.testAssumeBase)
				selectedBase = r.testAssumeBase
			} else {
				// Use runner's stdout/stderr
				_, _ = fmt.Fprintln(r.stdout, ui.Colors.WarningStyle.Render(fmt.Sprintf(
					"Warning: Parent branch '%s' is not tracked. Assuming stack base is '%s'.", selectedParent, defaultBaseBranch)))
				_, _ = fmt.Fprintln(r.stdout, ui.Colors.WarningStyle.Render("Consider tracking the parent branch first for more accurate stack definitions."))
				selectedBase = defaultBaseBranch
			}
		} else {
			return fmt.Errorf("failed to check tracking base for parent branch '%s': %w", selectedParent, errGetBase)
		}
	}

	if selectedBase == "" {
		return fmt.Errorf("could not determine base branch for the stack")
	}

	// 6. Store metadata in git config
	_, _ = fmt.Fprintf(r.stdout, "Tracking branch '%s' with parent '%s' and base '%s'.\n", currentBranch, selectedParent, selectedBase)

	err = git.SetGitConfig(parentConfigKey, selectedParent)
	if err != nil {
		return fmt.Errorf("failed to set socle-parent config: %w", err)
	}

	baseConfigKey := fmt.Sprintf("branch.%s.socle-base", currentBranch)
	err = git.SetGitConfig(baseConfigKey, selectedBase)
	if err != nil {
		_ = git.UnsetGitConfig(parentConfigKey)
		return fmt.Errorf("failed to set socle-base config: %w", err)
	}

	if discovery != nil && discovery.prNumber > 0 {
		if discovery.prBase != "" && discovery.prBase != selectedParent {
			_, _ = fmt.Fprintln(r.stderr, ui.Colors.WarningStyle.Render(fmt.Sprintf(
				"Warning: discovered PR #%d targets '%s', but you selected parent '%s'.",
				discovery.prNumber, discovery.prBase, selectedParent,
			)))
		}
		if errUnset := git.UnsetStoredPRNumber(currentBranch); errUnset != nil {
			r.logger.Debug("Failed to clear existing stored PR number before updating", "branch", currentBranch, "error", errUnset)
		}
		if errSet := git.SetStoredPRNumber(currentBranch, discovery.prNumber); errSet != nil {
			_, _ = fmt.Fprintln(r.stderr, ui.Colors.WarningStyle.Render(fmt.Sprintf(
				"Warning: failed to store discovered PR #%d locally: %v", discovery.prNumber, errSet,
			)))
		} else {
			_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render(fmt.Sprintf(
				"Stored discovered pull request #%d for '%s'.", discovery.prNumber, currentBranch,
			)))
		}
	} else if r.discoverRemote && discovery != nil {
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render("No open pull request discovered to store."))
	}

	_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render("Branch tracking information saved successfully."))
	return nil
}

type remoteDiscoveryResult struct {
	remoteName string
	remoteURL  string
	owner      string
	repo       string
	prNumber   int
	prBase     string
}

func (r *trackCmdRunner) discoverRemoteInfo(branch string) (*remoteDiscoveryResult, error) {
	remoteName := "origin"
	remoteKey := fmt.Sprintf("branch.%s.remote", branch)
	if remoteConfig, err := git.GetGitConfig(remoteKey); err == nil && remoteConfig != "" {
		remoteName = remoteConfig
	} else if err != nil && !errors.Is(err, git.ErrConfigNotFound) {
		return nil, fmt.Errorf("failed to read remote config for '%s': %w", branch, err)
	}

	remoteURL, err := git.GetRemoteURL(remoteName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve remote '%s' for branch '%s': %w", remoteName, branch, err)
	}

	owner, repo, err := git.ParseOwnerAndRepo(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse owner/repo from remote URL '%s': %w", remoteURL, err)
	}

	ctx := r.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	ghClient, err := gh.CreateClient(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client for %s/%s: %w", owner, repo, err)
	}

	pr, err := ghClient.FindPullRequestByHead(branch)
	if err != nil {
		return nil, fmt.Errorf("failed to discover pull request for branch '%s': %w", branch, err)
	}

	result := &remoteDiscoveryResult{
		remoteName: remoteName,
		remoteURL:  remoteURL,
		owner:      owner,
		repo:       repo,
	}
	if pr != nil {
		result.prNumber = pr.GetNumber()
		if base := pr.GetBase(); base != nil {
			result.prBase = base.GetRef()
		}
	}

	return result, nil
}

func (r *trackCmdRunner) emitDiscoverySummary(branch string, discovery *remoteDiscoveryResult) {
	if discovery == nil {
		return
	}

	_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render(fmt.Sprintf(
		"Discovered remote '%s' for branch '%s' (%s/%s).",
		discovery.remoteName, branch, discovery.owner, discovery.repo,
	)))

	if discovery.prNumber > 0 {
		baseInfo := discovery.prBase
		if baseInfo == "" {
			baseInfo = "unknown base"
		}
		_, _ = fmt.Fprintln(r.stdout, ui.Colors.InfoStyle.Render(fmt.Sprintf(
			"Found open pull request #%d targeting '%s'.", discovery.prNumber, baseInfo,
		)))
	}
}
