package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/benekuehn/socle/cli/so/internal/git"
	"github.com/spf13/cobra"
)

var debugLogging bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "so",
	Short: "A CLI tool for managing stacked Git branches and GitHub PRs",
	Long: `Socle helps streamline workflows involving stacked branches
(sequences of dependent branches) on Git and GitHub.`,
	SilenceErrors: true,
	SilenceUsage:  true,

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// ---- CONFIGURE SLOG ----
		var logLevel slog.Level
		if debugLogging {
			logLevel = slog.LevelDebug
		} else {
			logLevel = slog.LevelInfo // Default level
		}

		handlerOpts := &slog.HandlerOptions{
			Level: logLevel,
		}
		handler := slog.NewTextHandler(os.Stderr, handlerOpts)

		slog.SetDefault(slog.New(handler))

		slog.Debug("Debug logging enabled")

		// Git repo check
		if !git.IsGitRepo() {
			// Use slog for this internal error message? Or keep direct print?
			// Let's keep direct print for this specific startup failure.
			return fmt.Errorf("error: not a git repository (or any of the parent directories)")
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err) // More user-friendly error
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debugLogging, "debug", false, "Enable debug logging output")
}

func AddCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}
