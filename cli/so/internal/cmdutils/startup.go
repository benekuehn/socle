package cmdutils

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/benekuehn/socle/cli/so/gitutils"
)

// HandleStartupError manages errors from GetCurrentStackInfo for better command startup UX.
// It checks for specific conditions like being on the base branch or the branch being untracked,
// printing helpful messages to the provided writers.
// Returns true if the condition was handled (message printed), false otherwise.
// Returns the original error if it was unexpected and not handled.
func HandleStartupError(err error, currentBranchAttempt string, outW io.Writer, errW io.Writer) (handled bool, returnErr error) {
	cb := currentBranchAttempt
	if cb == "" {
		cb, _ = gitutils.GetCurrentBranch() // Best effort if initial attempt failed
	}

	// Check for untracked error conditions explicitly
	isUntrackedError := false
	if err != nil {
		// Check specific error type or characteristic strings
		if errors.Is(err, gitutils.ErrConfigNotFound) || strings.Contains(err.Error(), "not tracked by socle") {
			isUntrackedError = true
		}
	}

	// TODO: Make base branches configurable instead of hardcoded map
	knownBases := map[string]bool{"main": true, "master": true, "develop": true}
	// Only consider it truly "on base" if GetCurrentStackInfo succeeded without error
	isOnBase := knownBases[cb] && err == nil

	if isOnBase {
		fmt.Fprintf(outW, "Currently on the base branch '%s'.\n", cb)
		return true, nil // Handled successfully
	} else if isUntrackedError {
		fmt.Fprintf(outW, "Branch '%s' is not currently tracked by socle.\n", cb)
		fmt.Fprintln(outW, "Use 'so track' to associate it with a parent branch and start a stack.")
		return true, nil // Handled successfully
	}

	// Situation wasn't handled here (either no error and not on base, or an unexpected error)
	return false, err // Return original error (might be nil)
}
