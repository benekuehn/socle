package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"

	"github.com/AlecAivazis/survey/v2"
	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
)

type trackCmdRunner struct {
	ctx            context.Context
	logger         *slog.Logger
	stdout         io.Writer
	stderr         io.Writer
	stdin          io.Reader
	discoverRemote bool
	branch         string
	parent         string
	base           string
}

func (r *trackCmdRunner) run() error {
	child := r.branch
	if child == "" {
		var err error
		child, err = git.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
	}
	if err := requireLocalBranch("child", child); err != nil {
		return err
	}
	knownTrunks := map[string]bool{"main": true, "master": true, "develop": true}
	if knownTrunks[child] {
		return fmt.Errorf("cannot track a base branch ('%s') itself", child)
	}
	parentKey := fmt.Sprintf("branch.%s.socle-parent", child)
	baseKey := fmt.Sprintf("branch.%s.socle-base", child)
	oldParent, err := configValue(parentKey)
	if err != nil {
		return fmt.Errorf("failed to read existing parent for '%s': %w", child, err)
	}
	oldBase, err := configValue(baseKey)
	if err != nil {
		return fmt.Errorf("failed to read existing base for '%s': %w", child, err)
	}

	parent := r.parent
	if parent == "" && oldParent != "" {
		parent = oldParent
	}
	var discovery *remoteDiscoveryResult
	if r.discoverRemote {
		var err error
		discovery, err = r.discoverRemoteInfo(child)
		if err != nil {
			_, _ = fmt.Fprintln(r.stderr, ui.Colors.WarningStyle.Render(fmt.Sprintf("Remote discovery skipped: %v", err)))
		} else if discovery != nil {
			r.emitDiscoverySummary(child, discovery)
		}
	}
	if parent == "" {
		if nonInteractive || !hasInteractiveSurveyTerminal(r.stdin, r.stderr) {
			return fmt.Errorf("parent is required without an interactive terminal; retry with: so track --branch %s --parent <parent> [--base <base>]", child)
		}
		branches, err := git.GetLocalBranches()
		if err != nil {
			return fmt.Errorf("failed to list local branches: %w", err)
		}
		choices := make([]string, 0, len(branches))
		for _, b := range branches {
			if b != child {
				choices = append(choices, b)
			}
		}
		sort.Strings(choices)
		if len(choices) == 0 {
			return fmt.Errorf("no other local branches are available as a parent")
		}
		prompt := &survey.Select{Message: fmt.Sprintf("Select the parent branch for '%s':", child), Options: choices}
		if discovery != nil {
			for _, b := range choices {
				if b == discovery.prBase {
					prompt.Default = b
				}
			}
		}
		err = survey.AskOne(prompt, &parent, survey.WithStdio(r.stdin.(*os.File), r.stderr.(*os.File), r.stderr.(*os.File)))
		if err != nil {
			return ui.HandleSurveyInterrupt(err, "Track command cancelled.")
		}
	}
	if err := requireLocalBranch("parent", parent); err != nil {
		return err
	}
	if child == parent {
		return fmt.Errorf("child '%s' cannot track itself; retry with: so track --branch %s --parent <different-parent>", child, child)
	}
	if r.base != "" {
		if err := requireLocalBranch("base", r.base); err != nil {
			return err
		}
		if !knownTrunks[r.base] {
			return fmt.Errorf("unsupported base branch '%s'; supported bases are main, master, and develop", r.base)
		}
	}

	parents, err := git.GetAllSocleParents()
	if err != nil {
		return fmt.Errorf("failed to read tracking metadata: %w", err)
	}
	for node, seen := parent, map[string]bool{}; node != ""; node = parents[node] {
		if node == child {
			return fmt.Errorf("tracking '%s' on '%s' would create a metadata cycle", child, parent)
		}
		if seen[node] {
			return fmt.Errorf("existing tracking metadata contains a cycle through '%s'; repair it before tracking '%s'", node, child)
		}
		seen[node] = true
	}
	if !knownTrunks[parent] {
		for _, existingChild := range git.BuildChildMap(parents)[parent] {
			if existingChild != child {
				return fmt.Errorf("parent '%s' already has child '%s'; non-base branches can only have one child", parent, existingChild)
			}
		}
	}

	resolvedBase := ""
	if knownTrunks[parent] {
		resolvedBase = parent
	} else {
		parentParent, readErr := configValue(fmt.Sprintf("branch.%s.socle-parent", parent))
		if readErr != nil {
			return fmt.Errorf("failed to read parent metadata for '%s': %w", parent, readErr)
		}
		if parentParent != "" {
			parentBase, baseErr := configValue(fmt.Sprintf("branch.%s.socle-base", parent))
			if baseErr != nil {
				return fmt.Errorf("failed to read base for parent '%s': %w", parent, baseErr)
			}
			resolvedBase = parentBase
		}
		if parentParent != "" && resolvedBase == "" {
			return fmt.Errorf("tracked parent '%s' has incomplete metadata (missing base); repair it before tracking child '%s'", parent, child)
		}
		if parentParent == "" {
			return fmt.Errorf("parent '%s' is untracked and is not a supported base; track the parent first", parent)
		}
	}
	if resolvedBase == "" {
		resolvedBase = r.base
	}
	if r.base != "" && resolvedBase != r.base {
		return fmt.Errorf("base '%s' conflicts with inherited base '%s' from parent '%s'", r.base, resolvedBase, parent)
	}

	if oldParent == parent && oldBase == resolvedBase {
		r.storeDiscoveredPR(child, discovery)
		_, _ = fmt.Fprintf(r.stdout, "Tracked child '%s': parent '%s', base '%s' (unchanged).\n", child, parent, resolvedBase)
		return nil
	}
	if oldBase != resolvedBase {
		desc := git.FindAllDescendants(child, git.BuildChildMap(parents))
		if len(desc) > 0 {
			sort.Strings(desc)
			previousBase := oldBase
			if previousBase == "" {
				previousBase = "<missing>"
			}
			return fmt.Errorf("cannot change base for '%s' from '%s' to '%s': tracked descendants %v would become inconsistent", child, previousBase, resolvedBase, desc)
		}
	}
	if err := git.SetGitConfig(parentKey, parent); err != nil {
		return fmt.Errorf("failed to write parent for '%s': %w", child, err)
	}
	if err := git.SetGitConfig(baseKey, resolvedBase); err != nil {
		restoreConfig(parentKey, oldParent)
		return fmt.Errorf("failed to write base for '%s' (prior parent restored): %w", child, err)
	}

	r.storeDiscoveredPR(child, discovery)
	_, _ = fmt.Fprintf(r.stdout, "Tracked child '%s': parent '%s', base '%s'.\n", child, parent, resolvedBase)
	return nil
}

func (r *trackCmdRunner) storeDiscoveredPR(branch string, discovery *remoteDiscoveryResult) {
	if discovery == nil || discovery.prNumber <= 0 {
		return
	}
	if err := git.SetStoredPRNumber(branch, discovery.prNumber); err != nil {
		_, _ = fmt.Fprintf(r.stderr, "Warning: failed to store discovered PR #%d: %v\n", discovery.prNumber, err)
	}
}

func requireLocalBranch(role, branch string) error {
	exists, err := git.BranchExists(branch)
	if err != nil {
		return fmt.Errorf("failed to verify %s branch '%s': %w", role, branch, err)
	}
	if !exists {
		return fmt.Errorf("%s branch '%s' does not exist locally", role, branch)
	}
	return nil
}

func configValue(key string) (string, error) {
	value, err := git.GetGitConfig(key)
	if errors.Is(err, git.ErrConfigNotFound) {
		return "", nil
	}
	return value, err
}

func restoreConfig(key, value string) {
	if value == "" {
		_ = git.UnsetGitConfig(key)
	} else {
		_ = git.SetGitConfig(key, value)
	}
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
