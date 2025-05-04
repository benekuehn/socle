package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var restackCmd = &cobra.Command{
	Use:   "restack",
	Short: "Rebase the current stack onto the latest base branch",
	Long: `Updates the current stack by rebasing each branch sequentially onto its updated parent.
Handles remote 'origin' automatically.

Process:
1. Checks for clean state & existing Git rebase.
2. Fetches the base branch from 'origin' (unless --no-fetch).
3. Rebases each branch in the stack onto the latest commit of its parent.
   - Skips branches that are already up-to-date.
4. If conflicts occur:
   - Stops and instructs you to use standard Git commands (status, add, rebase --continue / --abort).
   - Run 'so restack' again after resolving or aborting the Git rebase.
5. If successful:
   - Prompts to force-push updated branches to 'origin' (use --force-push or --no-push to skip prompt).`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()

		runner := &restackCmdRunner{
			logger: logger,
			stdout: cmd.OutOrStdout(),
			stderr: cmd.ErrOrStderr(),
			stdin:  os.Stdin, // Needed for push prompt

			// Populate config from flags
			noFetch:   cmd.Flag("no-fetch").Changed,
			forcePush: cmd.Flag("force-push").Changed,
			noPush:    cmd.Flag("no-push").Changed,
		}

		return runner.run(cmd)
	},
}

func init() {
	AddCommand(restackCmd)
	// Define flags without binding to global vars
	restackCmd.Flags().Bool("no-fetch", false, "Skip fetching the remote base branch")
	restackCmd.Flags().Bool("force-push", false, "Force push rebased branches without prompting")
	restackCmd.Flags().Bool("no-push", false, "Do not push branches after successful rebase")
	// Flags that decide push behavior are mutually exclusive
	restackCmd.MarkFlagsMutuallyExclusive("force-push", "no-push")
}
