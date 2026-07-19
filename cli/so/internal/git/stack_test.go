package git

import (
	"testing"

	"github.com/benekuehn/socle/cli/so/internal/testutils"
	"github.com/stretchr/testify/require"
)

func TestStackInfoUsesCurrentSiblingLineage(t *testing.T) {
	repo, cleanup := testutils.SetupGitRepo(t)
	defer cleanup()
	trackStackBranches(t, repo, map[string]string{"A": "main", "B": "main"})
	testutils.RunCommand(t, repo, "git", "checkout", "A")

	info, err := GetStackInfo()
	require.NoError(t, err)
	require.Equal(t, []string{"main", "A"}, info.CurrentStack)
	require.Equal(t, []string{"main", "A"}, info.FullStack)
}

func TestStackInfoExtendsCurrentLineage(t *testing.T) {
	repo, cleanup := testutils.SetupGitRepo(t)
	defer cleanup()
	trackStackBranches(t, repo, map[string]string{"A": "main", "A2": "A"})
	testutils.RunCommand(t, repo, "git", "checkout", "A")

	info, err := GetStackInfo()
	require.NoError(t, err)
	require.Equal(t, []string{"main", "A", "A2"}, info.FullStack)
}

func TestStackInfoRejectsAmbiguousDescendants(t *testing.T) {
	repo, cleanup := testutils.SetupGitRepo(t)
	defer cleanup()
	trackStackBranches(t, repo, map[string]string{"A": "main", "A2": "A", "A3": "A"})
	testutils.RunCommand(t, repo, "git", "checkout", "A")

	_, err := GetStackInfo()
	require.EqualError(t, err, "ambiguous stack continuation from branch 'A': tracked children [A2 A3]")
}

func trackStackBranches(t *testing.T, repo string, parents map[string]string) {
	t.Helper()
	for branch, parent := range parents {
		testutils.RunCommand(t, repo, "git", "branch", branch)
		testutils.RunCommand(t, repo, "git", "config", "branch."+branch+".socle-parent", parent)
		testutils.RunCommand(t, repo, "git", "config", "branch."+branch+".socle-base", "main")
	}
}
