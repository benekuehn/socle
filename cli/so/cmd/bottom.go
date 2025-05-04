package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

var bottomCmd = &cobra.Command{
	Use:   "bottom",
	Short: "Checkout the bottom-most branch (above base) of the current stack",
	Long: `Navigates to the first branch stacked directly on top of the base branch.

The stack is determined by the tracking information set via 'so track'.
This command finds the first branch after the base in the sequence leading to the top.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()

		runner := &bottomCmdRunner{
			logger: logger,
			stdout: cmd.OutOrStdout(),
			stderr: cmd.ErrOrStderr(),
		}

		return runner.run()
	},
}

func init() {
	AddCommand(bottomCmd)
	// No flags needed for this command yet
}
