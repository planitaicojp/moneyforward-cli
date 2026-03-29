package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/planitaicojp/moneyforward-cli/internal/api"
)

// waitBriefly sleeps 10ms, used when polling for async state in tests.
func waitBriefly() { time.Sleep(10 * time.Millisecond) }

// ── RefreshAccessToken ────────────────────────────────────────────────────────

func TestRefreshAccessToken_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.FormValue("grant_type") != "refresh_token" {
			http.Error(w, "wrong grant_type", http.StatusBadRequest)
			return
		}
		if r.FormValue("refresh_token") != "rt-abc" {
			http.Error(w, "wrong refresh_token", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.TokenResponse{
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
			ExpiresIn:    3600,
			TokenType:    "Bearer",
			Scope:        "mfc/invoice/data.read",
		})
	}))
	defer srv.Close()

	tr, err := api.RefreshAccessToken(srv.URL, "client-id", "client-secret", "rt-abc")
	if err != nil {
		t.Fatalf("RefreshAccessToken: %v", err)
	}
	if tr.AccessToken != "new-access" {
		t.Errorf("AccessToken = %q, want %q", tr.AccessToken, "new-access")
	}
	if tr.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want 3600", tr.ExpiresIn)
	}
}

func TestRefreshAccessToken_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := api.RefreshAccessToken(srv.URL, "id", "secret", "bad-token")
	if err == nil {
		t.Fatal("expected error on HTTP 401, got nil")
	}
}

func TestRefreshAccessToken_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	_, err := api.RefreshAccessToken(srv.URL, "id", "secret", "rt")
	if err == nil {
		t.Fatal("expected error on invalid JSON, got nil")
	}
}

// ── Authorize (PKCE callback flow) ───────────────────────────────────────────

// TestAuthorize_Success simulates the full PKCE flow:
//  1. Call Authorize with a fake token server.
//  2. Capture the auth URL via PrintURL.
//  3. Parse the redirect_uri and state from the auth URL.
//  4. GET the redirect_uri with ?code=test-code&state=<state> to simulate
//     the browser redirect.
func TestAuthorize_Success(t *testing.T) {
	// Mock token endpoint.
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.FormValue("grant_type") != "authorization_code" {
			http.Error(w, "wrong grant_type", http.StatusBadRequest)
			return
		}
		if r.FormValue("code") != "test-code" {
			http.Error(w, "wrong code", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.TokenResponse{
			AccessToken:  "access-xyz",
			RefreshToken: "refresh-xyz",
			ExpiresIn:    7200,
			TokenType:    "Bearer",
			Scope:        "mfc/invoice/data.read",
		})
	}))
	defer tokenSrv.Close()

	svc := api.ServiceConfig{
		Name:    "invoice",
		AuthURL: "https://example.invalid/authorize", // never actually fetched
		TokenURL: tokenSrv.URL,
	}

	var capturedAuthURL string
	done := make(chan *api.TokenResponse, 1)
	errCh := make(chan error, 1)

	go func() {
		tr, err := api.Authorize(svc, api.AuthorizeOptions{
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			Scope:        "mfc/invoice/data.read",
			PrintURL: func(u string) {
				capturedAuthURL = u
			},
		})
		if err != nil {
			errCh <- err
			return
		}
		done <- tr
	}()

	// Wait until PrintURL has been called so we can extract the redirect_uri and state.
	// Poll with a brief sleep to avoid a race.
	for i := 0; i < 100; i++ {
		if capturedAuthURL != "" {
			break
		}
		// 10ms × 100 = up to 1s
		select {
		case err := <-errCh:
			t.Fatalf("Authorize failed early: %v", err)
		default:
		}
		waitBriefly()
	}
	if capturedAuthURL == "" {
		t.Fatal("PrintURL was never called")
	}

	// Parse redirect_uri and state from the authorization URL.
	parsed, err := url.Parse(capturedAuthURL)
	if err != nil {
		t.Fatalf("parsing auth URL: %v", err)
	}
	q := parsed.Query()
	redirectURI := q.Get("redirect_uri")
	state := q.Get("state")
	if redirectURI == "" || state == "" {
		t.Fatalf("missing redirect_uri or state in auth URL: %s", capturedAuthURL)
	}

	// Simulate the browser redirect — GET the callback URL.
	callbackURL := redirectURI + "?code=test-code&state=" + url.QueryEscape(state)
	resp, err := http.Get(callbackURL) //nolint:noctx
	if err != nil {
		t.Fatalf("simulating browser callback: %v", err)
	}
	resp.Body.Close()

	// Wait for Authorize to return.
	select {
	case tr := <-done:
		if tr.AccessToken != "access-xyz" {
			t.Errorf("AccessToken = %q, want %q", tr.AccessToken, "access-xyz")
		}
	case err := <-errCh:
		t.Fatalf("Authorize returned error: %v", err)
	}
}

func TestAuthorize_StateMismatch(t *testing.T) {
	svc := api.ServiceConfig{
		Name:     "invoice",
		AuthURL:  "https://example.invalid/authorize",
		TokenURL: "https://example.invalid/token",
	}

	var capturedAuthURL string
	errCh := make(chan error, 1)

	go func() {
		_, err := api.Authorize(svc, api.AuthorizeOptions{
			ClientID: "id",
			Scope:    "read",
			PrintURL: func(u string) { capturedAuthURL = u },
		})
		errCh <- err
	}()

	for i := 0; i < 100; i++ {
		if capturedAuthURL != "" {
			break
		}
		select {
		case err := <-errCh:
			t.Fatalf("Authorize failed before callback: %v", err)
		default:
		}
		waitBriefly()
	}

	parsed, _ := url.Parse(capturedAuthURL)
	redirectURI := parsed.Query().Get("redirect_uri")

	// Send a wrong state.
	callbackURL := redirectURI + "?code=c&state=wrong-state"
	resp, err := http.Get(callbackURL) //nolint:noctx
	if err != nil {
		t.Fatalf("callback GET: %v", err)
	}
	resp.Body.Close()

	if err := <-errCh; err == nil {
		t.Error("expected error on state mismatch, got nil")
	}
}

func TestAuthorize_OAuthError(t *testing.T) {
	svc := api.ServiceConfig{
		Name:     "invoice",
		AuthURL:  "https://example.invalid/authorize",
		TokenURL: "https://example.invalid/token",
	}

	var capturedAuthURL string
	errCh := make(chan error, 1)

	go func() {
		_, err := api.Authorize(svc, api.AuthorizeOptions{
			ClientID: "id",
			Scope:    "read",
			PrintURL: func(u string) { capturedAuthURL = u },
		})
		errCh <- err
	}()

	for i := 0; i < 100; i++ {
		if capturedAuthURL != "" {
			break
		}
		select {
		case err := <-errCh:
			t.Fatalf("Authorize failed before callback: %v", err)
		default:
		}
		waitBriefly()
	}

	parsed, _ := url.Parse(capturedAuthURL)
	redirectURI := parsed.Query().Get("redirect_uri")

	// Send an error response.
	callbackURL := redirectURI + "?error=access_denied&error_description=User+denied"
	resp, err := http.Get(callbackURL) //nolint:noctx
	if err != nil {
		t.Fatalf("callback GET: %v", err)
	}
	resp.Body.Close()

	if err := <-errCh; err == nil {
		t.Error("expected error on OAuth error callback, got nil")
	}
}
