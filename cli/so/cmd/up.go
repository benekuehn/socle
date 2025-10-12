package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

// testSelectStackIndex is a hidden flag used only in tests to bypass interactive stack selection
// when on a base branch with multiple stacks. Value < 0 means disabled.
var testSelectStackIndex int = -1
// testSelectStackChild selects a stack by its first child branch name (test only)
var testSelectStackChild string = ""

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Switch to the child of the current branch.",
	Long: `Navigates one level up the stack towards the tip.

The stack is determined by the tracking information set via 'so track'.
This command finds the immediate descendent of the current branch.

If you are on a base branch with multiple stacks, you will be prompted to select which stack to navigate to.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()

		runner := &upCmdRunner{
			logger: logger,
			stdout: cmd.OutOrStdout(),
			stderr: cmd.ErrOrStderr(),
			stdin:  os.Stdin,
		}

		return runner.run()
	},
}

func init() {
	upCmd.Flags().IntVar(&testSelectStackIndex, "test-select-stack-index", -1, "(test only) select stack index without prompt")
	_ = upCmd.Flags().MarkHidden("test-select-stack-index")
	upCmd.Flags().StringVar(&testSelectStackChild, "test-select-stack-child", "", "(test only) select stack whose first child matches branch name")
	_ = upCmd.Flags().MarkHidden("test-select-stack-child")
	AddCommand(upCmd)
}
