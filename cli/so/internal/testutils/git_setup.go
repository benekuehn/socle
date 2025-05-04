// cli/so/internal/testutils/git_setup.go
package testutils

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// RunCommand runs an OS command in a specific directory and fails the test on error.
func RunCommand(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command '%s %s' failed in dir '%s':\nError: %v\nOutput:\n%s", name, strings.Join(args, " "), dir, err, string(outBytes))
	}
	return string(outBytes)
}

// SetupGitRepo initializes a new Git repo in a temporary directory.
// It creates an initial commit on 'main'.
// It changes the CWD to the repo dir and returns the path and a cleanup func to cd back.
func SetupGitRepo(t *testing.T) (repoPath string, cleanup func()) {
	t.Helper()
	repoPath = t.TempDir()
	t.Logf("Created temp repo at: %s", repoPath)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	if err := os.Chdir(repoPath); err != nil {
		t.Fatalf("Failed to change directory to repo path '%s': %v", repoPath, err)
	}

	RunCommand(t, repoPath, "git", "init", "-b", "main")

	RunCommand(t, repoPath, "git", "config", "user.email", "test@example.com")
	RunCommand(t, repoPath, "git", "config", "user.name", "Test User")
	// Set config needed for GPG signing tests later, if any (avoids prompts)
	RunCommand(t, repoPath, "git", "config", "commit.gpgsign", "false")
	RunCommand(t, repoPath, "touch", "README.md")
	RunCommand(t, repoPath, "git", "add", "README.md")
	RunCommand(t, repoPath, "git", "commit", "-m", "Initial commit")

	cleanup = func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Errorf("Failed to change directory back to '%s': %v", originalWD, err)
		}
	}
	return repoPath, cleanup
}
