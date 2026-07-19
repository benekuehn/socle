package gh

import (
	"fmt"
	"log/slog"

	"github.com/benekuehn/socle/cli/so/internal/git"
)

// DiscoverPRs iterates through a list of branches, finds their corresponding open PRs on GitHub,
// and stores the PR numbers in the local Git config.
func DiscoverPRs(ghClient ClientInterface, branches []string) (map[string]int, error) {
	slog.Debug("Starting PR discovery", "branch_count", len(branches))
	discoveredPRs := make(map[string]int)

	for _, branch := range branches {
		// Skip if PR number is already known
		existingPRNum, err := git.GetStoredPRNumber(branch)
		if err != nil {
			return nil, fmt.Errorf("read stored PR number for branch %q: %w", branch, err)
		}
		if existingPRNum > 0 {
			slog.Debug("Skipping discovery for branch with existing PR number", "branch", branch, "pr_number", existingPRNum)
			discoveredPRs[branch] = existingPRNum
			continue
		}

		slog.Debug("Searching for open PR for branch", "branch", branch)
		pr, err := ghClient.FindOpenPullRequestForBranch(branch)
		if err != nil {
			return nil, fmt.Errorf("find open pull request for branch %q: %w", branch, err)
		}

		if pr != nil {
			prNumber := pr.GetNumber()
			slog.Info("Discovered existing PR", "branch", branch, "pr_number", prNumber, "pr_title", pr.GetTitle())
			discoveredPRs[branch] = prNumber

			// Store the discovered PR number in the local git config
			if err := git.SetStoredPRNumber(branch, prNumber); err != nil {
				return nil, fmt.Errorf("store discovered PR number for branch %q: %w", branch, err)
			}
		} else {
			slog.Debug("No open PR found for branch", "branch", branch)
		}
	}

	slog.Debug("Finished PR discovery", "discovered_count", len(discoveredPRs))
	return discoveredPRs, nil
}
