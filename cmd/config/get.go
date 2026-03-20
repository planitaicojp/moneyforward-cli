package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/config"
	cerrors "github.com/planitaicojp/moneyforward-cli/internal/errors"
)

var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long: `Get a single configuration value by key.

Valid keys:
  active_profile   - currently active profile
  format           - default output format (table, json, yaml, csv)
  client_id        - OAuth2 client ID for the current profile`,
	Args: cmdutil.ExactArgs(1),
	RunE: runGet,
}

func runGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	profile := getProfile(cmd)

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch key {
	case "active_profile":
		fmt.Println(profile)
	case "format":
		fmt.Println(cfg.Defaults.Format)
	case "client_id":
		if p, ok := cfg.Profiles[profile]; ok {
			fmt.Println(p.ClientID)
		} else {
			fmt.Println("")
		}
	default:
		return &cerrors.ValidationError{Field: "key", Message: fmt.Sprintf("unknown config key: %q", key)}
	}

	return nil
}
