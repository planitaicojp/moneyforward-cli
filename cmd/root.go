package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/auth"
	cmdconfig "github.com/planitaicojp/moneyforward-cli/cmd/config"
	"github.com/planitaicojp/moneyforward-cli/cmd/expense"
	"github.com/planitaicojp/moneyforward-cli/cmd/invoice"
	"github.com/planitaicojp/moneyforward-cli/internal/config"
	cerrors "github.com/planitaicojp/moneyforward-cli/internal/errors"
)

var (
	version = "dev"

	flagProfile string
	flagFormat  string
	flagNoInput bool
	flagQuiet   bool
	flagVerbose bool
	flagNoColor bool
)

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:           "mf",
	Short:         "Money Forward Cloud CLI",
	Long:          "Command-line interface for Money Forward Cloud APIs",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if flagNoInput {
			_ = os.Setenv(config.EnvNoInput, "1")
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagProfile, "profile", "", "config profile to use")
	rootCmd.PersistentFlags().StringVar(&flagFormat, "format", "", "output format: table, json, yaml, csv")
	rootCmd.PersistentFlags().BoolVar(&flagNoInput, "no-input", false, "disable interactive prompts")
	rootCmd.PersistentFlags().BoolVar(&flagQuiet, "quiet", false, "suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "disable color output")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(auth.AuthCmd)
	rootCmd.AddCommand(cmdconfig.ConfigCmd)
	rootCmd.AddCommand(invoice.InvoiceCmd)
	rootCmd.AddCommand(expense.ExpenseCmd)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cerrors.GetExitCode(err))
	}
}

// GetFormat returns the output format.
func GetFormat() string {
	if flagFormat != "" {
		return flagFormat
	}
	if f := config.EnvOr(config.EnvFormat, ""); f != "" {
		return f
	}
	cfg, err := config.Load()
	if err != nil {
		return config.DefaultFormat
	}
	return cfg.Defaults.Format
}

// IsQuiet returns whether quiet mode is enabled.
func IsQuiet() bool {
	return flagQuiet
}

// IsVerbose returns whether verbose mode is enabled.
func IsVerbose() bool {
	return flagVerbose
}
