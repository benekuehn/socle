package cmd

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
)

type untrackCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader
}

func (r *untrackCmdRunner) run() error {
	// Get current branch
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Check if branch is a base branch
	knownBases := map[string]bool{"main": true, "master": true, "develop": true}
	if knownBases[currentBranch] {
		return fmt.Errorf("cannot untrack a base branch ('%s')", currentBranch)
	}

	// Get all local branches to check for children
	allBranches, err := git.GetLocalBranches()
	if err != nil {
		return fmt.Errorf("failed to list local branches: %w", err)
	}

	// Check if any branch has this branch as its parent
	var children []string
	for _, branch := range allBranches {
		if branch == currentBranch {
			continue
		}
		parentConfigKey := fmt.Sprintf("branch.%s.socle-parent", branch)
		parent, err := git.GetGitConfig(parentConfigKey)
		if err == nil && parent == currentBranch {
			children = append(children, branch)
		}
	}

	if len(children) > 0 {
		return fmt.Errorf("cannot untrack branch '%s' because it has children depending on it: %v", currentBranch, children)
	}

	// Clear tracking information
	parentConfigKey := fmt.Sprintf("branch.%s.socle-parent", currentBranch)
	baseConfigKey := fmt.Sprintf("branch.%s.socle-base", currentBranch)

	if err := git.UnsetGitConfig(parentConfigKey); err != nil {
		return fmt.Errorf("failed to clear parent tracking: %w", err)
	}

	if err := git.UnsetGitConfig(baseConfigKey); err != nil {
		// Try to restore parent config if base config fails
		_ = git.SetGitConfig(parentConfigKey, currentBranch)
		return fmt.Errorf("failed to clear base tracking: %w", err)
	}

	_, _ = fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render(fmt.Sprintf("Branch '%s' has been untracked.", currentBranch)))
	return nil
}
