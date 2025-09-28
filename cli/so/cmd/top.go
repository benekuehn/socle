package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var topCmd = &cobra.Command{
	Use:   "top",
	Short: "Switch to tip of the current stack",
	Long: `Navigates to the highest branch in the current stack.

The stack is determined by the tracking information set via 'so track'.
This command finds the last branch in the sequence starting from the base branch.

If you are on a base branch with multiple stacks, you will be prompted to select which stack to navigate to the top of.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()

		runner := &topCmdRunner{
			logger: logger,
			stdout: cmd.OutOrStdout(),
			stderr: cmd.ErrOrStderr(),
			stdin:  os.Stdin,
		}

		return runner.run()
	},
}

func init() {
	AddCommand(topCmd)
	// No flags needed for this command yet
}
