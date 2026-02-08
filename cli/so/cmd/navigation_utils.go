package cmd

import (
	"fmt"
	"strings"

	"github.com/benekuehn/socle/cli/so/internal/git"
)

// checkoutBranch wraps git.CheckoutBranch with common error message logic.
func checkoutBranch(target string, current string) error {
	if err := git.CheckoutBranch(target); err != nil {
		if strings.Contains(err.Error(), "Please commit your changes or stash them") {
			return fmt.Errorf("cannot checkout branch '%s': uncommitted changes detected in '%s'. Please commit or stash them first", target, current)
		}
		return fmt.Errorf("failed to checkout branch '%s': %w", target, err)
	}
	return nil
}
