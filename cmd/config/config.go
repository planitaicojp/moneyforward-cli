package config

import "github.com/spf13/cobra"

// ConfigCmd is the parent config command.
var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long:  "Get, set, and list CLI configuration values.",
}

func init() {
	ConfigCmd.AddCommand(getCmd)
	ConfigCmd.AddCommand(setCmd)
	ConfigCmd.AddCommand(listCmd)
	ConfigCmd.AddCommand(pathCmd)
}
