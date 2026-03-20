package config

import (
	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
)

// getProfile resolves the active profile, respecting --profile flag and MF_PROFILE env var.
func getProfile(cmd *cobra.Command) string {
	return cmdutil.GetProfile(cmd)
}
