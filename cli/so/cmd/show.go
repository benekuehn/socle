package cmd

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Display the current tracked stack of branches",
	Long: `Shows the sequence of tracked branches leading from the stack's base
branch to the current branch, based on metadata set by 'socle track'.
Includes status indicating if a branch needs rebasing onto its parent.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		runner := &showCmdRunner{
			logger: slog.Default(),
			stdout: cmd.OutOrStdout(),
			stderr: cmd.ErrOrStderr(),
		}
		return runner.run(context.Background())
	},
}

func init() {
	AddCommand(showCmd)
}
