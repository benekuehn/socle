package cmdutils

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/benekuehn/socle/cli/so/internal/git"
)

// HandleStartupError manages errors from GetStackInfo for better command startup UX.
// It checks for specific conditions like being on the base branch or the branch being untracked.
// Returns:
// - handled: true if a standard message was printed, false otherwise
// - returnErr: the original error (potentially modified) that commands should handle
func HandleStartupError(err error, currentBranchAttempt string, outW io.Writer, errW io.Writer) (handled bool, returnErr error) {
	cb := currentBranchAttempt
	if cb == "" {
		cb, _ = git.GetCurrentBranch() // Best effort if initial attempt failed
	}

	// Check for untracked error conditions explicitly
	if err != nil && (errors.Is(err, git.ErrConfigNotFound) || strings.Contains(err.Error(), "not tracked by socle")) {
		// Just return the error - the command will handle the specific message format
		return false, fmt.Errorf("not tracked by socle")
	}

	// If there was no error, nothing to handle
	if err == nil {
		return false, nil
	}

	// For any other error, just return it
	return false, err
}
