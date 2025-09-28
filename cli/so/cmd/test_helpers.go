package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// setupRepoWithStack creates a git repository with a stack of branches
func setupRepoWithStack(t *testing.T, branches []string) (repoPath string, cleanup func()) {
	t.Helper()
	repoPath, cleanup = testutils.SetupGitRepo(t) // Starts with main commit

	// Create and track branches sequentially
	for i := 1; i < len(branches); i++ {
		parent := branches[i-1]
		branch := branches[i]
		// Create branch off parent
		testutils.RunCommand(t, repoPath, "git", "checkout", parent)
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", branch)
		// Add a unique commit to distinguish the branch
		writeFile(t, repoPath, fmt.Sprintf("%s.txt", branch), branch)
		testutils.RunCommand(t, repoPath, "git", "add", ".")
		testutils.RunCommand(t, repoPath, "git", "commit", "-m", fmt.Sprintf("feat: commit on %s", branch))
		// Track it (using runSoCommand with test flags)
		err := runSoCommand(t, "track", fmt.Sprintf("--test-parent=%s", parent))
		require.NoError(t, err, "Setup: failed to track branch %s", branch)
	}
	// Go back to a known branch (e.g., the tip)
	testutils.RunCommand(t, repoPath, "git", "checkout", branches[len(branches)-1])
	return repoPath, cleanup
}

// runSoCommandWithOutput executes a so command and returns its output
func runSoCommandWithOutput(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	var outBuf, errBuf bytes.Buffer

	testRootCmd, initErr := initializeCobraAppForTest()
	if initErr != nil {
		t.Fatalf("Failed to initialize Cobra app for test: %v", initErr)
	}

	testRootCmd.SetOut(&outBuf)
	testRootCmd.SetErr(&errBuf)
	testRootCmd.SetArgs(args)

	t.Logf("Executing 'so %s'", strings.Join(args, " "))
	err = testRootCmd.Execute()
	t.Logf("Execution finished, returned error: %v", err)

	stdout = outBuf.String()
	stderr = errBuf.String()

	t.Logf("Captured Stdout:\n%s", stdout)
	t.Logf("Captured Stderr:\n%s", stderr)

	return stdout, stderr, err
}

// writeFile writes content to a file in the given directory
func writeFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
	require.NoError(t, err, "Failed to write file %s", filename)
}

// readFile reads content from a file in the given directory
func readFile(t *testing.T, dir, filename string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(dir, filename))
	require.NoError(t, err, "Failed to read file %s", filename)
	return string(content)
}

// trackBranch sets socle tracking metadata for a branch
func trackBranch(t *testing.T, repoPath, branch, parent, base string) {
	t.Helper()
	parentKey := fmt.Sprintf("branch.%s.socle-parent", branch)
	baseKey := fmt.Sprintf("branch.%s.socle-base", branch)
	testutils.RunCommand(t, repoPath, "git", "config", "--local", parentKey, parent)
	testutils.RunCommand(t, repoPath, "git", "config", "--local", baseKey, base)
}

// setupRepoWithMultipleStacks creates a git repository with multiple stacks from main
// Creates two stacks: main->feature-a->feature-b and main->feature-x->feature-y
func setupRepoWithMultipleStacks(t *testing.T) (repoPath string, cleanup func()) {
	t.Helper()
	repoPath, cleanup = testutils.SetupGitRepo(t) // Starts with main commit

	// Create first stack: main -> feature-a -> feature-b
	testutils.RunCommand(t, repoPath, "git", "checkout", "main")
	testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature-a")
	writeFile(t, repoPath, "feature-a.txt", "feature-a content")
	testutils.RunCommand(t, repoPath, "git", "add", ".")
	testutils.RunCommand(t, repoPath, "git", "commit", "-m", "feat: commit on feature-a")
	err := runSoCommand(t, "track", "--test-parent=main")
	require.NoError(t, err, "Setup: failed to track feature-a")

	testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature-b")
	writeFile(t, repoPath, "feature-b.txt", "feature-b content")
	testutils.RunCommand(t, repoPath, "git", "add", ".")
	testutils.RunCommand(t, repoPath, "git", "commit", "-m", "feat: commit on feature-b")
	err = runSoCommand(t, "track", "--test-parent=feature-a")
	require.NoError(t, err, "Setup: failed to track feature-b")

	// Create second stack: main -> feature-x -> feature-y
	testutils.RunCommand(t, repoPath, "git", "checkout", "main")
	testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature-x")
	writeFile(t, repoPath, "feature-x.txt", "feature-x content")
	testutils.RunCommand(t, repoPath, "git", "add", ".")
	testutils.RunCommand(t, repoPath, "git", "commit", "-m", "feat: commit on feature-x")
	err = runSoCommand(t, "track", "--test-parent=main")
	require.NoError(t, err, "Setup: failed to track feature-x")

	testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature-y")
	writeFile(t, repoPath, "feature-y.txt", "feature-y content")
	testutils.RunCommand(t, repoPath, "git", "add", ".")
	testutils.RunCommand(t, repoPath, "git", "commit", "-m", "feat: commit on feature-y")
	err = runSoCommand(t, "track", "--test-parent=feature-x")
	require.NoError(t, err, "Setup: failed to track feature-y")

	return repoPath, cleanup
}

// runSoCommand executes a so command
func runSoCommand(t *testing.T, args ...string) error {
	t.Helper()

	testRootCmd, err := initializeCobraAppForTest()
	if err != nil {
		t.Fatalf("Failed to initialize Cobra app for test: %v", err)
	}
	testRootCmd.SetArgs(args)

	t.Logf("Executing 'so %s'", strings.Join(args, " "))
	err = testRootCmd.Execute()
	t.Logf("Execution finished, returned error: %v", err)
	return err
}

// initializeCobraAppForTest initializes a Cobra command for testing
func initializeCobraAppForTest() (*cobra.Command, error) {
	var testDebugLogging bool
	testRootCmd := &cobra.Command{Use: "so", SilenceErrors: true, SilenceUsage: true}
	testRootCmd.PersistentFlags().BoolVar(&testDebugLogging, "debug", false, "Enable debug logging output")
	addCmd := func(c *cobra.Command) { testRootCmd.AddCommand(c) }
	addCmd(trackCmd)
	addCmd(logCmd)
	addCmd(createCmd)
	addCmd(restackCmd)
	addCmd(submitCmd)
	addCmd(topCmd)
	addCmd(bottomCmd)
	addCmd(upCmd)
	addCmd(downCmd)
	addCmd(untrackCmd)
	addCmd(syncCmd)
	testRootCmd.Flags().AddFlagSet(trackCmd.Flags())
	return testRootCmd, nil
}
