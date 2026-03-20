package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/api"
	"github.com/planitaicojp/moneyforward-cli/internal/config"
	cerrors "github.com/planitaicojp/moneyforward-cli/internal/errors"
)

var (
	logoutService string
	logoutAll     bool
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials",
	Long:  `Remove stored OAuth2 tokens for the current profile and specified service.`,
	RunE:  runLogout,
}

func init() {
	logoutCmd.Flags().StringVarP(&logoutService, "service", "s", "invoice", "service to log out from (invoice, expense, payable)")
	logoutCmd.Flags().BoolVar(&logoutAll, "all", false, "log out from all services")
}

func runLogout(cmd *cobra.Command, args []string) error {
	profile := cmdutil.GetProfile(cmd)

	tokens, err := config.LoadTokens()
	if err != nil {
		return err
	}

	if logoutAll {
		tokens.DeleteProfile(profile)
		if err := tokens.Save(); err != nil {
			return fmt.Errorf("saving tokens: %w", err)
		}
		fmt.Printf("Logged out from all services for profile %q\n", profile)
		return nil
	}

	if _, ok := api.Services[logoutService]; !ok {
		return &cerrors.ValidationError{Field: "service", Message: fmt.Sprintf("unknown service: %q", logoutService)}
	}

	existing, err := tokens.Get(profile, logoutService)
	if err != nil {
		return err
	}
	if existing == nil {
		fmt.Printf("Not logged in to %s for profile %q\n", logoutService, profile)
		return nil
	}

	tokens.Delete(profile, logoutService)
	if err := tokens.Save(); err != nil {
		return fmt.Errorf("saving tokens: %w", err)
	}

	fmt.Printf("Logged out from %s for profile %q\n", logoutService, profile)
	return nil
}
