package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"
)

// ServiceConfig holds OAuth2 configuration for a Money Forward service.
type ServiceConfig struct {
	Name         string
	AuthURL      string
	TokenURL     string
	DefaultScope string
}

// Services maps service name to its OAuth2 configuration.
var Services = map[string]ServiceConfig{
	"invoice": {
		Name:         "invoice",
		AuthURL:      "https://api.biz.moneyforward.com/authorize",
		TokenURL:     "https://api.biz.moneyforward.com/token",
		DefaultScope: "mfc/invoice/data.read mfc/invoice/data.write",
	},
	"expense": {
		Name:         "expense",
		AuthURL:      "https://expense.moneyforward.com/oauth/authorize",
		TokenURL:     "https://expense.moneyforward.com/oauth/token",
		DefaultScope: "office_setting:write transaction:write report:write user_setting:write account:write public_resource:read",
	},
	"payable": {
		Name:         "payable",
		AuthURL:      "https://payable.moneyforward.com/oauth/authorize",
		TokenURL:     "https://payable.moneyforward.com/oauth/token",
		DefaultScope: "office_setting:write transaction:write report:write user_setting:write account:write public_resource:read",
	},
}

// TokenResponse is returned by the token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

const pkceChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"

// generateCodeVerifier creates a 43-character PKCE code verifier.
func generateCodeVerifier() (string, error) {
	buf := make([]byte, 43)
	for i := range buf {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(pkceChars))))
		if err != nil {
			return "", err
		}
		buf[i] = pkceChars[n.Int64()]
	}
	return string(buf), nil
}

// codeChallenge derives the S256 code challenge from a verifier.
func codeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// generateState creates a random state string for CSRF protection.
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// openBrowser attempts to open the given URL in the default browser.
func openBrowser(rawURL string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	_ = cmd.Start()
}

// AuthorizeOptions holds options for the authorization flow.
type AuthorizeOptions struct {
	ClientID     string
	ClientSecret string
	Scope        string
	// PrintURL is called with the authorization URL so the caller can display it.
	PrintURL func(authURL string)
}

// Authorize runs the OAuth2 PKCE Authorization Code flow.
// It starts a local callback server, opens the browser, and waits for the code.
func Authorize(svc ServiceConfig, opts AuthorizeOptions) (*TokenResponse, error) {
	verifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("generating code verifier: %w", err)
	}
	challenge := codeChallenge(verifier)

	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("generating state: %w", err)
	}

	// Find a free local port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("starting callback server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

	// Build authorization URL.
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", opts.ClientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", opts.Scope)
	params.Set("state", state)
	params.Set("code_challenge", challenge)
	params.Set("code_challenge_method", "S256")
	authURL := svc.AuthURL + "?" + params.Encode()

	if opts.PrintURL != nil {
		opts.PrintURL(authURL)
	}
	openBrowser(authURL)

	// Wait for callback.
	type callbackResult struct {
		code string
		err  error
	}
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		if errParam := q.Get("error"); errParam != "" {
			desc := q.Get("error_description")
			fmt.Fprintf(w, "<html><body><h2>Authentication failed</h2><p>%s: %s</p><p>You may close this window.</p></body></html>", errParam, desc)
			resultCh <- callbackResult{err: fmt.Errorf("oauth error %s: %s", errParam, desc)}
			return
		}

		if q.Get("state") != state {
			fmt.Fprintf(w, "<html><body><h2>Authentication failed</h2><p>State mismatch.</p></body></html>")
			resultCh <- callbackResult{err: fmt.Errorf("state mismatch in OAuth callback")}
			return
		}

		code := q.Get("code")
		if code == "" {
			fmt.Fprintf(w, "<html><body><h2>Authentication failed</h2><p>No code received.</p></body></html>")
			resultCh <- callbackResult{err: fmt.Errorf("no authorization code in callback")}
			return
		}

		fmt.Fprintf(w, "<html><body><h2>Authentication successful</h2><p>You may close this window and return to the terminal.</p></body></html>")
		resultCh <- callbackResult{code: code}
	})

	go func() {
		_ = srv.Serve(listener)
	}()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	select {
	case res := <-resultCh:
		if res.err != nil {
			return nil, res.err
		}
		return exchangeCode(svc.TokenURL, opts.ClientID, opts.ClientSecret, res.code, redirectURI, verifier)
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("timed out waiting for OAuth2 callback")
	}
}

// exchangeCode exchanges an authorization code for tokens.
func exchangeCode(tokenURL, clientID, clientSecret, code, redirectURI, verifier string) (*TokenResponse, error) {
	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("redirect_uri", redirectURI)
	params.Set("client_id", clientID)
	params.Set("client_secret", clientSecret)
	params.Set("code_verifier", verifier)

	resp, err := http.PostForm(tokenURL, params)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned HTTP %d", resp.StatusCode)
	}

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}
	return &tr, nil
}

// RefreshAccessToken uses a refresh token to obtain a new access token.
func RefreshAccessToken(tokenURL, clientID, clientSecret, refreshToken string) (*TokenResponse, error) {
	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", refreshToken)
	params.Set("client_id", clientID)
	params.Set("client_secret", clientSecret)

	resp, err := http.PostForm(tokenURL, params)
	if err != nil {
		return nil, fmt.Errorf("refresh token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh token endpoint returned HTTP %d", resp.StatusCode)
	}

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("parsing refresh token response: %w", err)
	}
	return &tr, nil
}
