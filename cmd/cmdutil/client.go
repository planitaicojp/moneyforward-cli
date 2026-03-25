package cmdutil

import (
	"fmt"
	"os"
	"time"

	"github.com/planitaicojp/moneyforward-cli/internal/api"
	"github.com/planitaicojp/moneyforward-cli/internal/config"
	cerrors "github.com/planitaicojp/moneyforward-cli/internal/errors"
)

// EnsureValidToken returns a valid access token for the given profile and service,
// automatically refreshing if the stored token is expired.
func EnsureValidToken(profile string, svc api.ServiceConfig) (string, error) {
	tokens, err := config.LoadTokens()
	if err != nil {
		return "", err
	}

	ts, err := tokens.Get(profile, svc.Name)
	if err != nil {
		return "", err
	}
	if ts == nil {
		return "", &cerrors.AuthError{Message: fmt.Sprintf("not logged in to %s for profile %q; run 'mf auth login --service %s'", svc.Name, profile, svc.Name)}
	}

	if !ts.IsExpired() {
		return ts.AccessToken, nil
	}

	// Token is expired — attempt refresh.
	if ts.RefreshToken == "" {
		return "", &cerrors.AuthError{Message: fmt.Sprintf("token expired and no refresh token available; run 'mf auth login --service %s'", svc.Name)}
	}

	cfg, err := config.Load()
	if err != nil {
		return "", err
	}

	p, ok := cfg.Profiles[profile]
	if !ok || p.ClientID == "" {
		return "", &cerrors.ConfigError{Message: fmt.Sprintf("no client_id configured for profile %q", profile)}
	}

	creds, err := config.LoadCredentials()
	if err != nil {
		return "", err
	}
	clientSecret := creds.GetClientSecret(profile)
	if clientSecret == "" {
		return "", &cerrors.AuthError{Message: "client secret not found; re-authenticate with 'mf auth login'"}
	}

	tr, err := api.RefreshAccessToken(svc.TokenURL, p.ClientID, clientSecret, ts.RefreshToken)
	if err != nil {
		return "", &cerrors.AuthError{Message: fmt.Sprintf("refreshing token: %v", err)}
	}

	expiresAt := time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	newTS := &config.TokenSet{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    tr.TokenType,
		Scope:        tr.Scope,
	}
	// Preserve existing refresh token if the new response didn't include one.
	if newTS.RefreshToken == "" {
		newTS.RefreshToken = ts.RefreshToken
	}

	tokens.Set(profile, svc.Name, newTS)
	if err := tokens.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save refreshed token: %v\n", err)
	}

	return newTS.AccessToken, nil
}
