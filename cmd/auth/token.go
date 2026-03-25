package auth

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/api"
	cerrors "github.com/planitaicojp/moneyforward-cli/internal/errors"
)

var tokenService string

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Print the access token",
	Long: `Print the access token for the current profile and service to stdout.

If the MF_ACCESS_TOKEN environment variable is set, it is printed directly.
If the stored token is expired and a refresh token is available, the token
is automatically refreshed before printing.`,
	RunE: runToken,
}

func init() {
	tokenCmd.Flags().StringVarP(&tokenService, "service", "s", "invoice", "service to get token for (invoice, expense, payable)")
}

func runToken(cmd *cobra.Command, args []string) error {
	// Environment variable override.
	if t := os.Getenv("MF_ACCESS_TOKEN"); t != "" {
		fmt.Println(t)
		return nil
	}

	profile := cmdutil.GetProfile(cmd)

	svc, ok := api.Services[tokenService]
	if !ok {
		return &cerrors.ValidationError{Field: "service", Message: fmt.Sprintf("unknown service: %q", tokenService)}
	}

	accessToken, err := cmdutil.EnsureValidToken(profile, svc)
	if err != nil {
		return err
	}

	fmt.Println(accessToken)
	return nil
}
