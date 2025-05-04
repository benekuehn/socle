package cmd

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/ui"
	// Assuming HandleSurveyInterrupt is there or moved to ui
)

type trackCmdRunner struct {
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader

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
		fmt.Fprintf(r.stdout, "Branch '%s' is already tracked.\n", currentBranch)
		fmt.Fprintf(r.stdout, "  Parent: %s\n", existingParent)
		fmt.Fprintf(r.stdout, "  Base:   %s\n", existingBase)
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

	// 4. Prompt user to select parent
	selectedParent := ""
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
				fmt.Fprintln(r.stdout, ui.Colors.WarningStyle.Render(fmt.Sprintf(
					"Warning: Parent branch '%s' is not tracked. Assuming stack base is '%s'.", selectedParent, defaultBaseBranch)))
				fmt.Fprintln(r.stdout, ui.Colors.WarningStyle.Render("Consider tracking the parent branch first for more accurate stack definitions."))
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
	fmt.Fprintf(r.stdout, "Tracking branch '%s' with parent '%s' and base '%s'.\n", currentBranch, selectedParent, selectedBase)

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

	fmt.Fprintln(r.stdout, ui.Colors.SuccessStyle.Render("Branch tracking information saved successfully."))
	return nil
}
