package config

import (
	"github.com/planitaicojp/moneyforward-cli/internal/config"
)

// getActiveProfile returns the active profile name from config.
func getActiveProfile(cfg *config.Config) string {
	if cfg.ActiveProfile != "" {
		return cfg.ActiveProfile
	}
	return "default"
}
