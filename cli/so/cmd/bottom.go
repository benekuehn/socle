package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var bottomCmd = &cobra.Command{
	Use:   "bottom",
	Short: "Switch to the bottom-most branch (above base) of the current stack",
	Long: `Navigates to the first branch stacked directly on top of the base branch.

The stack is determined by the tracking information set via 'so track'.
This command finds the first branch after the base in the sequence leading to the top.

If you are on a base branch with multiple stacks, you will be prompted to select which stack to navigate to.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()

		runner := &bottomCmdRunner{
			logger: logger,
			stdout: cmd.OutOrStdout(),
			stderr: cmd.ErrOrStderr(),
			stdin:  os.Stdin,
		}

		return runner.run()
	},
}

func init() {
	AddCommand(bottomCmd)
}
