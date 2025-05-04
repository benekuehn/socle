package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Checkout the parent branch of the current branch in the stack",
	Long: `Navigates one level up the stack towards the base branch.

The stack is determined by the tracking information set via 'so track'.
This command finds the immediate parent of the current branch.`,
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
	// No flags needed for this command yet
}
