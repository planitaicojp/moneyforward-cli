package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/internal/api"
	"github.com/planitaicojp/moneyforward-cli/internal/config"
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

	profile := getProfile(cmd)

	svc, ok := api.Services[tokenService]
	if !ok {
		return &cerrors.ValidationError{Field: "service", Message: fmt.Sprintf("unknown service: %q", tokenService)}
	}

	tokens, err := config.LoadTokens()
	if err != nil {
		return err
	}

	ts, err := tokens.Get(profile, svc.Name)
	if err != nil {
		return err
	}
	if ts == nil {
		return &cerrors.AuthError{Message: fmt.Sprintf("not logged in to %s for profile %q; run 'mf auth login --service %s'", svc.Name, profile, svc.Name)}
	}

	// Auto-refresh if expired.
	if ts.IsExpired() {
		if ts.RefreshToken == "" {
			return &cerrors.AuthError{Message: fmt.Sprintf("token expired and no refresh token available; run 'mf auth login --service %s'", svc.Name)}
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		p, ok := cfg.Profiles[profile]
		if !ok || p.ClientID == "" {
			return &cerrors.ConfigError{Message: fmt.Sprintf("no client_id configured for profile %q", profile)}
		}

		creds, err := config.LoadCredentials()
		if err != nil {
			return err
		}
		clientSecret := creds.GetClientSecret(profile)
		if clientSecret == "" {
			return &cerrors.AuthError{Message: "client secret not found; re-authenticate with 'mf auth login'"}
		}

		tr, err := api.RefreshAccessToken(svc.TokenURL, p.ClientID, clientSecret, ts.RefreshToken)
		if err != nil {
			return &cerrors.AuthError{Message: fmt.Sprintf("refreshing token: %v", err)}
		}

		expiresAt := time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
		ts = &config.TokenSet{
			AccessToken:  tr.AccessToken,
			RefreshToken: tr.RefreshToken,
			ExpiresAt:    expiresAt,
			TokenType:    tr.TokenType,
			Scope:        tr.Scope,
		}
		// Preserve existing refresh token if the new response didn't include one.
		if ts.RefreshToken == "" {
			oldTS, _ := tokens.Get(profile, svc.Name)
			if oldTS != nil {
				ts.RefreshToken = oldTS.RefreshToken
			}
		}

		tokens.Set(profile, svc.Name, ts)
		if err := tokens.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save refreshed token: %v\n", err)
		}
	}

	fmt.Println(ts.AccessToken)
	return nil
}
