package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/config"
	cerrors "github.com/planitaicojp/moneyforward-cli/internal/errors"
	"github.com/planitaicojp/moneyforward-cli/internal/prompt"
)

var removeCmd = &cobra.Command{
	Use:   "remove <profile-name>",
	Short: "Remove a profile",
	Long:  `Remove a profile and all its stored tokens and credentials.`,
	Args:  cmdutil.ExactArgs(1),
	RunE:  runRemove,
}

func runRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	noInput := config.IsNoInput()

	if !noInput {
		ok, err := prompt.Confirm(fmt.Sprintf("Remove profile %q and all its tokens?", name))
		if err != nil {
			return err
		}
		if !ok {
			return &cerrors.ValidationError{Message: "aborted"}
		}
	}

	// Remove from config.
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	delete(cfg.Profiles, name)
	if cfg.ActiveProfile == name {
		cfg.ActiveProfile = ""
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Remove tokens.
	tokens, err := config.LoadTokens()
	if err != nil {
		return err
	}
	tokens.DeleteProfile(name)
	if err := tokens.Save(); err != nil {
		return fmt.Errorf("saving tokens: %w", err)
	}

	// Remove credentials.
	creds, err := config.LoadCredentials()
	if err != nil {
		return err
	}
	creds.DeleteProfile(name)
	if err := creds.Save(); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	fmt.Printf("Removed profile %q\n", name)
	return nil
}
