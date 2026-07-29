package cmd

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/gh"
	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/google/go-github/v83/github"
)

func TestTrackCommand(t *testing.T) {
	t.Run("Track new branch successfully", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup() // Ensure we cd back

		// Setup: Create feature/a branch
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature/a")

		// Action: Run 'so track', simulating selection of 'main'
		err := runSoCommand(t, "track", "--parent=main")

		// Assertion 1: Command should succeed
		if err != nil {
			t.Fatalf("so track failed unexpectedly: %v", err)
		}

		// Assertion 2: Check Git config
		parent, err := git.GetGitConfig("branch.feature/a.socle-parent")
		if err != nil {
			t.Fatalf("Failed to get socle-parent config after track: %v", err)
		}
		if parent != "main" {
			t.Errorf("Expected socle-parent to be 'main', but got '%s'", parent)
		}

		base, err := git.GetGitConfig("branch.feature/a.socle-base")
		if err != nil {
			t.Fatalf("Failed to get socle-base config after track: %v", err)
		}
		if base != "main" {
			t.Errorf("Expected socle-base to be 'main', but got '%s'", base)
		}
	})

	t.Run("Track requires parent when no TTY is available", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature/a")

		originalNonInteractive := nonInteractive
		nonInteractive = false
		t.Cleanup(func() { nonInteractive = originalNonInteractive })

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		runner := &trackCmdRunner{
			ctx:    context.Background(),
			logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
			stdout: &stdout,
			stderr: &stderr,
			stdin:  strings.NewReader(""),
		}

		err := runner.run()
		if err == nil || !strings.Contains(err.Error(), "so track --branch feature/a --parent <parent>") {
			t.Fatalf("expected actionable non-terminal error, got: %v", err)
		}
		if _, err := git.GetGitConfig("branch.feature/a.socle-parent"); !errors.Is(err, git.ErrConfigNotFound) {
			t.Fatalf("non-terminal failure wrote parent metadata: %v", err)
		}
	})

	t.Run("Attempt to track already tracked branch", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature/a")
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature/a.socle-parent", "main")
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature/a.socle-base", "main")

		// Action: Run 'so track' again
		err := runSoCommand(t, "track", "--parent=main")

		// Assertion 1: Command should succeed (informational exit)
		if err != nil {
			t.Fatalf("so track failed unexpectedly when branch already tracked: %v", err)
		}

		// Assertion 2: Config should remain unchanged
		parent, err := git.GetGitConfig("branch.feature/a.socle-parent")
		if err != nil {
			t.Fatalf("Failed to get socle-parent config: %v", err)
		}
		if parent != "main" {
			t.Errorf("socle-parent changed unexpectedly, got '%s'", parent)
		}
		base, err := git.GetGitConfig("branch.feature/a.socle-base")
		if err != nil {
			t.Fatalf("Failed to get socle-base config: %v", err)
		}
		if base != "main" {
			t.Errorf("socle-base changed unexpectedly, got '%s'", base)
		}
	})

	t.Run("Attempt to track base branch", func(t *testing.T) {
		_, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		err := runSoCommand(t, "track")

		expectedErrorMsg := "cannot track a base branch ('main') itself"
		if !strings.Contains(err.Error(), expectedErrorMsg) {
			t.Errorf("Expected error message containing '%s', but got: %v", expectedErrorMsg, err)
		}
	})

	t.Run("Discover remote pull request metadata", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()

		testutils.RunCommand(t, repoPath, "git", "remote", "add", "origin", "git@github.com:example/repo.git")
		testutils.RunCommand(t, repoPath, "git", "checkout", "-b", "feature/a")
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.feature/a.remote", "origin")

		mockClient := gh.NewMockClient()
		prNumber := 123
		prBase := "main"
		pr := &github.PullRequest{
			Number: github.Ptr(prNumber),
			Base: &github.PullRequestBranch{
				Ref: github.Ptr(prBase),
			},
		}
		mockClient.On("FindPullRequestByHead", "feature/a").Return(pr, nil)

		originalCreateClient := gh.CreateClient
		gh.CreateClient = func(ctx context.Context, owner, repo string) (gh.ClientInterface, error) {
			if owner != "example" || repo != "repo" {
				t.Fatalf("unexpected owner/repo: %s/%s", owner, repo)
			}
			return mockClient, nil
		}
		t.Cleanup(func() {
			gh.CreateClient = originalCreateClient
		})

		err := runSoCommand(t, "track", "--discover", "--parent=main")
		if err != nil {
			t.Fatalf("so track with discover failed unexpectedly: %v", err)
		}

		storedPR, err := git.GetGitConfig("branch.feature/a.socle-pr-number")
		if err != nil {
			t.Fatalf("failed to read stored PR number: %v", err)
		}
		if storedPR != "123" {
			t.Errorf("expected stored PR number '123', got '%s'", storedPR)
		}

		mockClient.AssertExpectations(t)
	})
}

func TestTrackExplicitRelationships(t *testing.T) {
	t.Run("configures stack without changing checkout", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()
		for _, branch := range []string{"feature/a", "feature/b", "feature/c"} {
			testutils.RunCommand(t, repoPath, "git", "branch", branch)
		}
		commands := [][]string{
			{"track", "--branch=feature/a", "--parent=main"},
			{"track", "--branch=feature/b", "--parent=feature/a"},
			{"track", "--branch=feature/c", "--parent=feature/b"},
		}
		for _, args := range commands {
			if err := runSoCommand(t, args...); err != nil {
				t.Fatalf("so %s failed: %v", strings.Join(args, " "), err)
			}
		}
		parents, err := git.GetAllSocleParents()
		if err != nil {
			t.Fatal(err)
		}
		expected := map[string]string{"feature/a": "main", "feature/b": "feature/a", "feature/c": "feature/b"}
		if len(parents) != len(expected) {
			t.Fatalf("expected exact parent graph %v, got %v", expected, parents)
		}
		for child, parent := range expected {
			if parents[child] != parent {
				t.Fatalf("expected %s -> %s, got %q", child, parent, parents[child])
			}
			base, err := git.GetGitConfig("branch." + child + ".socle-base")
			if err != nil || base != "main" {
				t.Fatalf("expected base main for %s, got %q (%v)", child, base, err)
			}
		}
		if current, err := git.GetCurrentBranch(); err != nil || current != "main" {
			t.Fatalf("checkout changed: branch=%q error=%v", current, err)
		}
	})

	t.Run("requires explicit base for untracked parent and rejects stale base", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()
		testutils.RunCommand(t, repoPath, "git", "branch", "parent")
		testutils.RunCommand(t, repoPath, "git", "branch", "child")
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.parent.socle-base", "main")
		err := runSoCommand(t, "track", "--branch=child", "--parent=parent")
		if err == nil || !strings.Contains(err.Error(), "--base <base>") {
			t.Fatalf("expected explicit-base retry, got %v", err)
		}
		if err := runSoCommand(t, "track", "--branch=child", "--parent=parent", "--base=main"); err != nil {
			t.Fatalf("explicit base failed: %v", err)
		}
	})

	t.Run("rejects cycles and unsafe base changes", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()
		for _, branch := range []string{"a", "b", "other"} {
			testutils.RunCommand(t, repoPath, "git", "branch", branch)
		}
		if err := runSoCommand(t, "track", "--branch=a", "--parent=main", "--base="); err != nil {
			t.Fatal(err)
		}
		if err := runSoCommand(t, "track", "--branch=b", "--parent=a", "--base="); err != nil {
			t.Fatal(err)
		}
		if err := runSoCommand(t, "track", "--branch=a", "--parent=b", "--base="); err == nil || !strings.Contains(err.Error(), "cycle") {
			t.Fatalf("expected cycle error, got %v", err)
		}
		if err := runSoCommand(t, "track", "--branch=other", "--parent=main", "--base="); err != nil {
			t.Fatal(err)
		}
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.other.socle-base", "other")
		if err := runSoCommand(t, "track", "--branch=a", "--parent=other", "--base=other"); err == nil || !strings.Contains(err.Error(), "tracked descendants") {
			t.Fatalf("expected descendant safety error, got %v", err)
		}
	})

	t.Run("rejects repairing a missing base above descendants", func(t *testing.T) {
		repoPath, cleanup := testutils.SetupGitRepo(t)
		defer cleanup()
		for _, branch := range []string{"parent", "child"} {
			testutils.RunCommand(t, repoPath, "git", "branch", branch)
		}
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.parent.socle-parent", "main")
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.child.socle-parent", "parent")
		testutils.RunCommand(t, repoPath, "git", "config", "--local", "branch.child.socle-base", "main")

		err := runSoCommand(t, "track", "--branch=parent", "--parent=main", "--base=")
		if err == nil || !strings.Contains(err.Error(), "from '<missing>'") || !strings.Contains(err.Error(), "tracked descendants") {
			t.Fatalf("expected missing-base descendant safety error, got %v", err)
		}
		if _, err := git.GetGitConfig("branch.parent.socle-base"); !errors.Is(err, git.ErrConfigNotFound) {
			t.Fatalf("failed validation wrote base metadata: %v", err)
		}
	})
}
