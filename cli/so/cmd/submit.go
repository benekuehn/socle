// cli/cmd/submit.go
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Create or update GitHub Pull Requests for the current stack",
	Long: `Pushes branches in the current stack to the remote ('origin' by default)
and creates or updates corresponding GitHub Pull Requests.

- Requires GITHUB_TOKEN environment variable with 'repo' scope or auth setup via 'gh auth login'.
- Reads PR templates from .github/ or root directory.
- Creates Draft PRs by default (use --no-draft to override).
- Stores PR numbers locally in '.git/config' for future updates.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()

		body, _ := cmd.Flags().GetString("body")
		bodyFile, _ := cmd.Flags().GetString("body-file")
		// Note: mutual exclusivity is enforced via MarkFlagsMutuallyExclusive in init()
		if bodyFile != "" {
			bodyBytes, err := os.ReadFile(bodyFile)
			if err != nil {
				return fmt.Errorf("failed to read --body-file %q: %w", bodyFile, err)
			}
			body = string(bodyBytes)
		}

		title, _ := cmd.Flags().GetString("title")
		forcePush, _ := cmd.Flags().GetBool("force")
		noPush, _ := cmd.Flags().GetBool("no-push")
		noDraft, _ := cmd.Flags().GetBool("no-draft")

		runner := &submitCmdRunner{
			logger:         logger,
			stdout:         cmd.OutOrStdout(),
			stderr:         cmd.ErrOrStderr(),
			nonInteractive: nonInteractive,

			// Populate config from flags
			forcePush:   forcePush,
			noPush:      noPush,
			draft:       !noDraft,
			submitTitle: title,
			submitBody:  body,
			// --- TESTING FLAGS ---
			testSubmitTitle:       mustGetString(cmd, "test-title"),
			testSubmitBody:        mustGetString(cmd, "test-body"),
			testSubmitEditConfirm: mustGetBool(cmd, "test-edit-confirm"),
		}

		return runner.run(context.Background(), cmd)
	},
}

func init() {
	rootCmd.AddCommand(submitCmd)
	submitCmd.Flags().Bool("force", false, "Force push branches")
	submitCmd.Flags().Bool("no-push", false, "Skip pushing branches to remote")
	submitCmd.Flags().Bool("no-draft", false, "Create non-draft Pull Requests")
	submitCmd.Flags().String("title", "", "PR title to use when creating pull requests")
	submitCmd.Flags().String("body", "", "PR body (markdown) to use when creating pull requests")
	submitCmd.Flags().String("body-file", "", "Path to file containing PR body markdown")

	// --- TESTING FLAGS ---
	submitCmd.Flags().String("test-title", "", "TESTING: Override PR title")
	submitCmd.Flags().String("test-body", "", "TESTING: Override PR body")
	submitCmd.Flags().Bool("test-edit-confirm", false, "TESTING: Confirm edit prompt")
	_ = submitCmd.Flags().MarkHidden("test-title")
	_ = submitCmd.Flags().MarkHidden("test-body")
	_ = submitCmd.Flags().MarkHidden("test-edit-confirm")

	submitCmd.MarkFlagsMutuallyExclusive("body", "body-file")
}

// mustGetString is a helper that panics if the flag doesn't exist (programming error).
func mustGetString(cmd *cobra.Command, name string) string {
	v, err := cmd.Flags().GetString(name)
	if err != nil {
		panic(fmt.Sprintf("flag %q not defined: %v", name, err))
	}
	return v
}

// mustGetBool is a helper that panics if the flag doesn't exist (programming error).
func mustGetBool(cmd *cobra.Command, name string) bool {
	v, err := cmd.Flags().GetBool(name)
	if err != nil {
		panic(fmt.Sprintf("flag %q not defined: %v", name, err))
	}
	return v
}
