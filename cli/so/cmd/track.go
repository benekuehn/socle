package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var trackCmd = &cobra.Command{
	Use:   "track",
	Short: "Start tracking the current branch as part of a stack",
	Long: `Associates the current branch with a parent branch to define its position
within a stack. This allows 'socle show' to display the specific stack you are on.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()

		runner := &trackCmdRunner{
			logger: logger,
			stdout: cmd.OutOrStdout(),
			stderr: cmd.ErrOrStderr(),
			stdin:  os.Stdin,

			testSelectedParent: cmd.Flag("test-parent").Value.String(),
			testAssumeBase:     cmd.Flag("test-base").Value.String(),
		}

		return runner.run()
	},
}

const defaultBaseBranch = "main"

func init() {
	AddCommand(trackCmd)
	trackCmd.Flags().String("test-parent", "", "Parent branch to select (for testing only)")
	trackCmd.Flags().String("test-base", "", "Base branch to assume if parent is untracked (for testing only)")
	_ = trackCmd.Flags().MarkHidden("test-parent")
	_ = trackCmd.Flags().MarkHidden("test-base")
}
