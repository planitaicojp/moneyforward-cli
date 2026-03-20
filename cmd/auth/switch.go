package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/config"
)

var switchCmd = &cobra.Command{
	Use:   "switch <profile-name>",
	Short: "Switch active profile",
	Long:  `Set the active profile used by default for all commands.`,
	Args:  cmdutil.ExactArgs(1),
	RunE:  runSwitch,
}

func runSwitch(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cfg.ActiveProfile = name

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Switched to profile %q\n", name)
	return nil
}
