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

// DiscoverCompletedPRs finds the sole PR for branches missing local PR metadata.
// It refuses ambiguous matches, which can occur when a branch name has been reused.
func DiscoverCompletedPRs(ghClient ClientInterface, branches []string) (map[string]int, error) {
	discoveredPRs := make(map[string]int)
	for _, branch := range branches {
		existingPRNum, err := git.GetStoredPRNumber(branch)
		if err != nil {
			return nil, fmt.Errorf("read stored PR number for branch %q: %w", branch, err)
		}
		if existingPRNum > 0 {
			discoveredPRs[branch] = existingPRNum
			continue
		}

		prs, err := ghClient.FindPullRequestsForBranch(branch, "all")
		if err != nil {
			return nil, fmt.Errorf("find pull requests for branch %q: %w", branch, err)
		}
		if len(prs) > 1 {
			return nil, fmt.Errorf("multiple pull requests found for branch %q", branch)
		}
		if len(prs) == 1 {
			pr := prs[0]
			if pr.GetState() == "closed" {
				branchHead, err := git.GetCurrentBranchCommit(branch)
				if err != nil {
					return nil, fmt.Errorf("get current commit for branch %q: %w", branch, err)
				}
				if branchHead != pr.GetHead().GetSHA() {
					slog.Warn("Skipping closed pull request with a different branch head", "branch", branch, "pr_number", pr.GetNumber())
					continue
				}
			}

			prNumber := pr.GetNumber()
			discoveredPRs[branch] = prNumber
			if err := git.SetStoredPRNumber(branch, prNumber); err != nil {
				return nil, fmt.Errorf("store discovered PR number for branch %q: %w", branch, err)
			}
		}
	}
	return discoveredPRs, nil
}
