package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Switch to the parent of the current branch.",
	Long: `Navigates one level down the stack towards the base branch.

The stack is determined by the tracking information set via 'so track'.
This command finds the immediate parent of the current branch.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()

		runner := &downCmdRunner{
			logger: logger,
			stdout: cmd.OutOrStdout(),
			stderr: cmd.ErrOrStderr(),
		}

		return runner.run()
	},
}

func init() {
	AddCommand(downCmd)
}
