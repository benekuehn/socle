package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Make command variables Exported if InitializeCobraApp needs them directly
// var TrackCmd = trackCmd // Example if needed, but better to use AddCommand

// SetupRootCommandForTest initializes the root command with all subcommands for testing.
// It returns a *new* instance each time to avoid test pollution.
func SetupRootCommandForTest() *cobra.Command {
	// Create a new root command instance each time
	var testDebugLogging bool // Local scope for the test instance flag
	testRootCmd := &cobra.Command{
		Use:           "so",
		Short:         "Socle helps streamline workflows involving stacked branches",
		Long:          `...`,
		SilenceErrors: true, // Good for tests
		SilenceUsage:  true,
		// Add PersistentPreRunE if its side effects are needed/tested,
		// otherwise maybe omit for simpler unit tests of RunE.
		// We might need a way to inject dependencies (like git client mocks later) here too.
	}
	testRootCmd.PersistentFlags().BoolVar(&testDebugLogging, "debug", false, "Enable debug logging output")

	// Use a local AddCommand that adds to testRootCmd
	addCmd := func(c *cobra.Command) { testRootCmd.AddCommand(c) }

	// Initialize all commands (they will call the local addCmd via their init)
	// This relies on the init() funcs in each command file calling AddCommand.
	// Make sure trackCmd, showCmd etc. are defined in this package.
	addCmd(trackCmd) // If trackCmd is lowercase, this works as it's same package
	addCmd(showCmd)
	addCmd(createCmd)
	addCmd(restackCmd)
	addCmd(submitCmd)
	// Add other commands...

	// Add test flags from trackCmd again to this instance
	// This is awkward - better if flags were attached differently,
	// but for now, re-add them.
	testRootCmd.Flags().AddFlagSet(trackCmd.Flags())

	return testRootCmd
}

// ResetTestFlags resets package-level test flag variables between tests.
func ResetTestFlags() {
	testSelectedParent = ""
	testAssumeBase = ""
	testBranchName = ""
	testStageChoice = ""
	testAddPResultEmpty = false
	// Reset other test flags from other commands here...
}

func initializeCobraAppForTest() (*cobra.Command, error) {
	// It can directly access trackCmd, showCmd etc. as they are in the same package
	var testDebugLogging bool
	testRootCmd := &cobra.Command{Use: "so", SilenceErrors: true, SilenceUsage: true}
	testRootCmd.PersistentFlags().BoolVar(&testDebugLogging, "debug", false, "Enable debug logging output")
	addCmd := func(c *cobra.Command) { testRootCmd.AddCommand(c) }
	addCmd(trackCmd)
	addCmd(showCmd)
	addCmd(createCmd)
	addCmd(restackCmd)
	addCmd(submitCmd)
	// Re-add test flags if needed (this part is still awkward)
	testRootCmd.Flags().AddFlagSet(trackCmd.Flags())
	return testRootCmd, nil
}

// runSoCommand remains largely the same, but calls the local initializer
// and resetter. No longer exported.
func runSoCommand(t *testing.T, args ...string) error {
	t.Helper()
	// Reset package-level test flags
	testSelectedParent = "" // Assuming these are defined in this package (e.g., track.go)
	testAssumeBase = ""

	testRootCmd, err := initializeCobraAppForTest() // Call local initializer
	if err != nil {
		t.Fatalf("Failed to initialize Cobra app for test: %v", err)
	}
	testRootCmd.SetArgs(args)

	t.Logf("Executing 'so %s'", strings.Join(args, " "))
	err = testRootCmd.Execute()
	t.Logf("Execution finished, returned error: %v", err)
	return err
}

func runSoCommandWithOutput(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	ResetTestFlags() // Reset flags before each command run

	// Create new buffer for stdout and stderr
	var outBuf, errBuf bytes.Buffer

	testRootCmd, initErr := initializeCobraAppForTest()
	if initErr != nil {
		t.Fatalf("Failed to initialize Cobra app for test: %v", initErr)
	}

	// Set output pipes
	testRootCmd.SetOut(&outBuf)
	testRootCmd.SetErr(&errBuf) // Capture stderr too

	testRootCmd.SetArgs(args)

	t.Logf("Executing 'so %s'", strings.Join(args, " "))
	err = testRootCmd.Execute() // Execute the command
	t.Logf("Execution finished, returned error: %v", err)

	stdout = outBuf.String()
	stderr = errBuf.String()

	// Log captured output for debugging tests
	t.Logf("Captured Stdout:\n%s", stdout)
	t.Logf("Captured Stderr:\n%s", stderr)

	return stdout, stderr, err // Return captured output and error
}

// Helper to create a file with content
func writeFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
	require.NoError(t, err, "Failed to write file %s", filename)
}

// Helper to read file content
func readFile(t *testing.T, dir, filename string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(dir, filename))
	require.NoError(t, err, "Failed to read file %s", filename)
	return string(content)
}
