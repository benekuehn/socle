// cli/cmd/submit.go
package cmd

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"
)

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Create or update GitHub Pull Requests for the current stack",
	Long: `Pushes branches in the current stack to the remote ('origin' by default)
and creates or updates corresponding GitHub Pull Requests.

- Requires GITHUB_TOKEN environment variable with 'repo' scope.
- Reads PR templates from .github/ or root directory.
- Creates Draft PRs by default (use --no-draft to override).
- Stores PR numbers locally in '.git/config' for future updates.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()

		runner := &submitCmdRunner{
			logger: logger,
			stdout: cmd.OutOrStdout(),
			stderr: cmd.ErrOrStderr(),

			// Populate config from flags
			forcePush: cmd.Flag("force").Changed,
			noPush:    cmd.Flag("no-push").Changed,
			draft:     !cmd.Flag("no-draft").Changed,
			// --- TESTING FLAGS ---
			testSubmitTitle:       cmd.Flag("test-title").Value.String(),
			testSubmitBody:        cmd.Flag("test-body").Value.String(),
			testSubmitEditConfirm: cmd.Flag("test-edit-confirm").Changed, // Assuming bool flag
		}

		return runner.run(context.Background(), cmd)
	},
}

func init() {
	rootCmd.AddCommand(submitCmd)
	submitCmd.Flags().Bool("force", false, "Force push branches")
	submitCmd.Flags().Bool("no-push", false, "Skip pushing branches to remote")
	submitCmd.Flags().Bool("no-draft", false, "Create non-draft Pull Requests")

	// --- TESTING FLAGS ---
	submitCmd.Flags().String("test-title", "", "TESTING: Override PR title")
	submitCmd.Flags().String("test-body", "", "TESTING: Override PR body")
	submitCmd.Flags().Bool("test-edit-confirm", false, "TESTING: Confirm edit prompt")
	_ = submitCmd.Flags().MarkHidden("test-title")
	_ = submitCmd.Flags().MarkHidden("test-body")
	_ = submitCmd.Flags().MarkHidden("test-edit-confirm")
}
