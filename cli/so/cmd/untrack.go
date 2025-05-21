package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var untrackCmd = &cobra.Command{
	Use:   "untrack",
	Short: "Remove a branch from the stack",
	Long: `Removes a branch from the stack by clearing its tracking information.
A branch can only be untracked if it has no children depending on it higher in the stack.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()

		runner := &untrackCmdRunner{
			logger: logger,
			stdout: cmd.OutOrStdout(),
			stderr: cmd.ErrOrStderr(),
			stdin:  os.Stdin,
		}

		return runner.run()
	},
}

func init() {
	AddCommand(untrackCmd)
}
