package auth

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/api"
	"github.com/planitaicojp/moneyforward-cli/internal/config"
	cerrors "github.com/planitaicojp/moneyforward-cli/internal/errors"
	"github.com/planitaicojp/moneyforward-cli/internal/prompt"
)

var (
	loginService  string
	loginClientID string
	loginScopes   string
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Money Forward Cloud",
	Long: `Authenticate with Money Forward Cloud using OAuth2 PKCE flow.

Opens a browser window to complete the authentication. The access token
is stored in ~/.config/mf/tokens.yaml.`,
	RunE: runLogin,
}

func init() {
	loginCmd.Flags().StringVarP(&loginService, "service", "s", "invoice", "service to authenticate (invoice, expense, payable)")
	loginCmd.Flags().StringVar(&loginClientID, "client-id", "", "OAuth2 client ID")
	loginCmd.Flags().StringVar(&loginScopes, "scopes", "", "OAuth2 scopes (comma-separated, overrides default)")
}

func runLogin(cmd *cobra.Command, args []string) error {
	profile := cmdutil.GetProfile(cmd)
	noInput := config.IsNoInput()

	// Validate service.
	svc, ok := api.Services[loginService]
	if !ok {
		names := make([]string, 0, len(api.Services))
		for k := range api.Services {
			names = append(names, k)
		}
		return &cerrors.ValidationError{Field: "service", Message: fmt.Sprintf("unknown service %q, valid values: %s", loginService, strings.Join(names, ", "))}
	}

	// Load config (used for client ID resolution and saving).
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Resolve client ID.
	clientID := loginClientID
	if clientID == "" {
		if p, ok := cfg.Profiles[profile]; ok {
			clientID = p.ClientID
		}
	}
	if clientID == "" {
		if noInput {
			return &cerrors.ConfigError{Message: "client ID is required; set it with --client-id or config set client_id"}
		}
		var err error
		clientID, err = prompt.Input("Enter OAuth2 Client ID", "")
		if err != nil {
			return err
		}
		if clientID == "" {
			return &cerrors.ValidationError{Field: "client_id", Message: "client ID cannot be empty"}
		}
	}

	// Resolve client secret.
	creds, err := config.LoadCredentials()
	if err != nil {
		return err
	}
	clientSecret := creds.GetClientSecret(profile)
	if clientSecret == "" {
		if noInput {
			return &cerrors.ConfigError{Message: "client secret is required; run without --no-input to provide it interactively"}
		}
		clientSecret, err = prompt.Password("Enter OAuth2 Client Secret")
		if err != nil {
			return err
		}
		if clientSecret == "" {
			return &cerrors.ValidationError{Field: "client_secret", Message: "client secret cannot be empty"}
		}
		// Offer to save.
		save, err := prompt.Confirm("Save client secret to credentials file")
		if err != nil {
			return err
		}
		if save {
			creds.SetClientSecret(profile, clientSecret)
			if saveErr := creds.Save(); saveErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not save credentials: %v\n", saveErr)
			}
		}
	}

	// Determine scopes.
	scope := svc.DefaultScope
	if loginScopes != "" {
		scope = strings.ReplaceAll(loginScopes, ",", " ")
	}

	// Save client_id to config if not already there.
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]config.Profile)
	}
	p := cfg.Profiles[profile]
	if p.ClientID == "" {
		p.ClientID = clientID
		cfg.Profiles[profile] = p
		if saveErr := cfg.Save(); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save config: %v\n", saveErr)
		}
	}

	fmt.Fprintf(os.Stderr, "Opening browser for %s authentication...\n", svc.Name)
	fmt.Fprintf(os.Stderr, "If the browser does not open, copy the URL below:\n\n")

	tokenResp, err := api.Authorize(svc, api.AuthorizeOptions{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scope:        scope,
		PrintURL: func(authURL string) {
			fmt.Fprintln(os.Stderr, authURL)
			fmt.Fprintln(os.Stderr)
		},
	})
	if err != nil {
		return &cerrors.AuthError{Message: err.Error()}
	}

	// Store tokens.
	tokens, err := config.LoadTokens()
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	tokens.Set(profile, svc.Name, &config.TokenSet{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    tokenResp.TokenType,
		Scope:        tokenResp.Scope,
	})

	if err := tokens.Save(); err != nil {
		return fmt.Errorf("saving tokens: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\nLogged in to %s as profile %q\n", svc.Name, profile)
	return nil
}


