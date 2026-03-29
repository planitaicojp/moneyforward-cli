package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/config"
	cerrors "github.com/planitaicojp/moneyforward-cli/internal/errors"
)

var setCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Valid keys:
  active_profile   - set the active profile
  format           - set default output format (table, json, yaml, csv)
  client_id        - set OAuth2 client ID for the current profile`,
	Args: cmdutil.ExactArgs(2),
	RunE: runSet,
}

func runSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]

	profile := getProfile(cmd)

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch key {
	case "active_profile":
		cfg.ActiveProfile = value
	case "format":
		switch value {
		case "table", "json", "yaml", "csv":
			cfg.Defaults.Format = value
		default:
			return &cerrors.ValidationError{Field: "format", Message: fmt.Sprintf("invalid format %q; valid values: table, json, yaml, csv", value)}
		}
	case "client_id":
		if cfg.Profiles == nil {
			cfg.Profiles = make(map[string]config.Profile)
		}
		p := cfg.Profiles[profile]
		p.ClientID = value
		cfg.Profiles[profile] = p
	default:
		return &cerrors.ValidationError{Field: "key", Message: fmt.Sprintf("unknown config key: %q", key)}
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Set %s = %s\n", key, value)
	return nil
}
