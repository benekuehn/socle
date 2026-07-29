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

		discoverRemote, err := cmd.Flags().GetBool("discover")
		if err != nil {
			return err
		}

		runner := &trackCmdRunner{
			ctx:    cmd.Context(),
			logger: logger,
			stdout: cmd.OutOrStdout(),
			stderr: cmd.ErrOrStderr(),
			stdin:  os.Stdin,

			discoverRemote: discoverRemote,
			branch:         cmd.Flag("branch").Value.String(),
			parent:         cmd.Flag("parent").Value.String(),
			base:           cmd.Flag("base").Value.String(),
		}

		return runner.run()
	},
}

func init() {
	AddCommand(trackCmd)
	trackCmd.Flags().String("branch", "", "Child branch to track (defaults to the current branch)")
	trackCmd.Flags().String("parent", "", "Parent branch to track")
	trackCmd.Flags().String("base", "", "Expected stack base (validated against the parent's inherited base)")
	trackCmd.Flags().BoolP("discover", "d", false, "Discover remote metadata (e.g. existing pull requests) while tracking")
}
