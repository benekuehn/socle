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

		body := cmd.Flag("body").Value.String()
		bodyFile := cmd.Flag("body-file").Value.String()
		if body != "" && bodyFile != "" {
			return fmt.Errorf("--body and --body-file are mutually exclusive")
		}
		if bodyFile != "" {
			bodyBytes, err := os.ReadFile(bodyFile)
			if err != nil {
				return fmt.Errorf("failed to read --body-file %q: %w", bodyFile, err)
			}
			body = string(bodyBytes)
		}

		runner := &submitCmdRunner{
			logger:         logger,
			stdout:         cmd.OutOrStdout(),
			stderr:         cmd.ErrOrStderr(),
			nonInteractive: nonInteractive,

			// Populate config from flags
			forcePush:   cmd.Flag("force").Changed,
			noPush:      cmd.Flag("no-push").Changed,
			draft:       !cmd.Flag("no-draft").Changed,
			submitTitle: cmd.Flag("title").Value.String(),
			submitBody:  body,
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
}
