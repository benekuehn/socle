package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

// Flag variables
var createMessage string

// --- ADD TESTING FLAGS ---
var (
	testBranchName      string // Bypass branch name prompt
	testStageChoice     string // Bypass stage choice prompt ("add-all", "add-p", "cancel")
	testAddPResultEmpty bool   // Simulate 'git add -p' staging nothing
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
			stdin:  os.Stdin, // Assuming Stdin is appropriate here

			// Populate config from flags
			createMessage: createMessage, // Use the package-level var bound to the flag
			branchNameArg: branchNameArg, // Pass the argument

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
	createCmd.Flags().StringVarP(&createMessage, "message", "m", "", "Commit message to use for uncommitted changes")
	createCmd.Flags().StringVar(&testBranchName, "test-branch-name", "", "Branch name to use (testing only)")
	createCmd.Flags().StringVar(&testStageChoice, "test-stage-choice", "", "Staging choice [add-all|add-p|cancel] (testing only)")
	createCmd.Flags().BoolVar(&testAddPResultEmpty, "test-add-p-empty", false, "Simulate 'git add -p' staging nothing (testing only)")
	_ = createCmd.Flags().MarkHidden("test-branch-name")
	_ = createCmd.Flags().MarkHidden("test-stage-choice")
	_ = createCmd.Flags().MarkHidden("test-add-p-empty")
}
