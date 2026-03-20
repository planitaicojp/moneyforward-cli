package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/internal/config"
)

var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the configuration directory path",
	Long:  `Print the path to the configuration directory (~/.config/mf by default).`,
	RunE:  runPath,
}

func runPath(cmd *cobra.Command, args []string) error {
	fmt.Println(config.DefaultConfigDir())
	return nil
}
