package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func booleanFlagCommand(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{RunE: func(*cobra.Command, []string) error { return nil }}
	cmd.Flags().Bool("force", false, "")
	cmd.Flags().Bool("force-push", false, "")
	cmd.Flags().Bool("no-push", false, "")
	cmd.Flags().Bool("no-fetch", false, "")
	cmd.Flags().Bool("no-draft", false, "")
	cmd.Flags().Bool("no-restack", false, "")
	require.NoError(t, cmd.ParseFlags(args))
	return cmd
}

func TestBooleanFlagSubmitOptions(t *testing.T) {
	for _, tt := range []struct {
		name, flag           string
		force, noPush, draft bool
	}{
		{"force true", "--force", true, false, true},
		{"force false", "--force=false", false, false, true},
		{"no push true", "--no-push", false, true, true},
		{"no push false", "--no-push=false", false, false, true},
		{"no draft true", "--no-draft", false, false, false},
		{"no draft false", "--no-draft=false", false, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			force, noPush, draft := submitBooleanOptions(booleanFlagCommand(t, tt.flag))
			assert.Equal(t, tt.force, force)
			assert.Equal(t, tt.noPush, noPush)
			assert.Equal(t, tt.draft, draft)
		})
	}
}

func TestBooleanFlagRestackOptions(t *testing.T) {
	for _, tt := range []struct {
		name, flag                 string
		noFetch, forcePush, noPush bool
	}{
		{"no fetch true", "--no-fetch", true, false, false},
		{"no fetch false", "--no-fetch=false", false, false, false},
		{"force push true", "--force-push", false, true, false},
		{"force push false", "--force-push=false", false, false, false},
		{"no push true", "--no-push", false, false, true},
		{"no push false", "--no-push=false", false, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			noFetch, forcePush, noPush := restackBooleanOptions(booleanFlagCommand(t, tt.flag))
			assert.Equal(t, tt.noFetch, noFetch)
			assert.Equal(t, tt.forcePush, forcePush)
			assert.Equal(t, tt.noPush, noPush)
		})
	}
}

func TestBooleanFlagSyncOptions(t *testing.T) {
	for _, tt := range []struct {
		name, flag string
		doRestack  bool
	}{
		{"no restack true", "--no-restack", false},
		{"no restack false", "--no-restack=false", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.doRestack, syncBooleanOptions(booleanFlagCommand(t, tt.flag)))
		})
	}
}

func TestBooleanFlagRestackPushFlagsRemainExclusive(t *testing.T) {
	cmd := booleanFlagCommand(t)
	cmd.MarkFlagsMutuallyExclusive("force-push", "no-push")
	cmd.SetArgs([]string{"--force-push", "--no-push"})
	assert.Error(t, cmd.Execute())
}
