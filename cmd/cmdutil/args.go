package cmdutil

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/internal/config"
)

// ExactArgs returns a PositionalArgs that reports the command's Use line on mismatch.
func ExactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return fmt.Errorf("requires %d arg(s), received %d\n\nUsage:\n  %s", n, len(args), cmd.UseLine())
		}
		return nil
	}
}

// GetProfile resolves the active profile from (in order of precedence):
//  1. --profile flag on the root command
//  2. MF_PROFILE environment variable
//  3. active_profile in config.yaml
//  4. "default"
func GetProfile(cmd *cobra.Command) string {
	if f := cmd.Root().PersistentFlags().Lookup("profile"); f != nil && f.Value.String() != "" {
		return f.Value.String()
	}
	if p := config.EnvOr(config.EnvProfile, ""); p != "" {
		return p
	}
	cfg, err := config.Load()
	if err != nil {
		return "default"
	}
	if cfg.ActiveProfile != "" {
		return cfg.ActiveProfile
	}
	return "default"
}

// GetFormat resolves the output format from (in order of precedence):
//  1. --format flag on the root command
//  2. MF_FORMAT environment variable
//  3. defaults.format in config.yaml
//  4. "table"
func GetFormat(cmd *cobra.Command) string {
	if f := cmd.Root().PersistentFlags().Lookup("format"); f != nil && f.Value.String() != "" {
		return f.Value.String()
	}
	if f := config.EnvOr(config.EnvFormat, ""); f != "" {
		return f
	}
	cfg, err := config.Load()
	if err != nil {
		return config.DefaultFormat
	}
	if cfg.Defaults.Format != "" {
		return cfg.Defaults.Format
	}
	return config.DefaultFormat
}
