package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Switch to the child of the current branch.",
	Long: `Navigates one level up the stack towards the tip.

The stack is determined by the tracking information set via 'so track'.
This command finds the immediate descendent of the current branch.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()

		runner := &upCmdRunner{
			logger: logger,
			stdout: cmd.OutOrStdout(),
			stderr: cmd.ErrOrStderr(),
		}

		return runner.run()
	},
}

func init() {
	AddCommand(upCmd)
}
