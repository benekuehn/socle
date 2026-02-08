package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync branches with remote and clean up merged/closed PRs",
	Long: `Syncs all branches with remote, prompting to delete any branches for PRs that have been merged or closed.
Restacks all branches in your repository that can be restacked without conflicts.
If trunk cannot be fast-forwarded to match remote, overwrites trunk with the remote version.

Process:
1. Fetches all branches from remote
2. Checks PR status for each branch
3. Prompts to delete branches with merged/closed PRs
4. Restacks branches that can be restacked without conflicts
5. Updates trunk to match remote if needed`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()

		noFetch, _ := cmd.Flags().GetBool("test-no-fetch")
		noSurvey, _ := cmd.Flags().GetBool("test-no-survey")

		runner := &syncCmdRunner{
			logger:         logger,
			stdout:         cmd.OutOrStdout(),
			stderr:         cmd.ErrOrStderr(),
			stdin:          os.Stdin, // Needed for prompts
			nonInteractive: nonInteractive,

			// Populate config from flags
			doRestack: !cmd.Flag("no-restack").Changed,
			noFetch:   noFetch,
			noSurvey:  noSurvey,
		}

		return runner.run(cmd)
	},
}

func init() {
	AddCommand(syncCmd)
	syncCmd.Flags().Bool("no-restack", false, "Skip restacking branches")
	syncCmd.Flags().Bool("test-no-fetch", false, "TESTING: Skip fetching from remote")
	syncCmd.Flags().Bool("test-no-survey", false, "TESTING: Auto-answer yes to all prompts")
	_ = syncCmd.Flags().MarkHidden("test-no-fetch")
	_ = syncCmd.Flags().MarkHidden("test-no-survey")
}
