package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create [branch-name]",
	Short: "Create the next branch in the stack, optionally committing current changes",
	Long: `Creates a new branch stacked on top of the current branch.

If a [branch-name] is not provided, you will be prompted for one.

If there are uncommitted changes in the working directory:
  - They will be staged and committed onto the *new* branch.
  - You must provide a commit message via the -m flag, or you will be prompted.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()

		branchNameArg := ""
		if len(args) > 0 {
			branchNameArg = args[0]
		}

		runner := &createCmdRunner{
			logger: logger,
			stdout: cmd.OutOrStdout(),
			stderr: cmd.ErrOrStderr(),
			stdin:  os.Stdin,

			// Populate config from flags
			createMessage: cmd.Flag("message").Value.String(),
			branchNameArg: branchNameArg,

			// --- TESTING FLAGS ---
			testBranchName:      cmd.Flag("test-branch-name").Value.String(),
			testStageChoice:     cmd.Flag("test-stage-choice").Value.String(),
			testAddPResultEmpty: cmd.Flag("test-add-p-empty").Changed,
		}

		return runner.run()
	},
}

func init() {
	AddCommand(createCmd)
	createCmd.Flags().StringP("message", "m", "", "Commit message to use for uncommitted changes")

	createCmd.Flags().String("test-branch-name", "", "Branch name to use (testing only)")
	createCmd.Flags().String("test-stage-choice", "", "Staging choice [add-all|add-p|cancel] (testing only)")
	createCmd.Flags().Bool("test-add-p-empty", false, "Simulate 'git add -p' staging nothing (testing only)")
	_ = createCmd.Flags().MarkHidden("test-branch-name")
	_ = createCmd.Flags().MarkHidden("test-stage-choice")
	_ = createCmd.Flags().MarkHidden("test-add-p-empty")
}
