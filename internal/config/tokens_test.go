package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/planitaicojp/moneyforward-cli/internal/config"
)

// ── IsExpired ────────────────────────────────────────────────────────────────

func TestTokenSet_IsExpired_ZeroTime(t *testing.T) {
	ts := &config.TokenSet{}
	if !ts.IsExpired() {
		t.Error("zero ExpiresAt should be expired")
	}
}

func TestTokenSet_IsExpired_Past(t *testing.T) {
	ts := &config.TokenSet{ExpiresAt: time.Now().Add(-1 * time.Hour)}
	if !ts.IsExpired() {
		t.Error("past ExpiresAt should be expired")
	}
}

func TestTokenSet_IsExpired_Within60s(t *testing.T) {
	// Token that expires in 30s is within the 60s safety buffer.
	ts := &config.TokenSet{ExpiresAt: time.Now().Add(30 * time.Second)}
	if !ts.IsExpired() {
		t.Error("token expiring within 60s should be treated as expired")
	}
}

func TestTokenSet_IsExpired_Future(t *testing.T) {
	ts := &config.TokenSet{ExpiresAt: time.Now().Add(10 * time.Minute)}
	if ts.IsExpired() {
		t.Error("future ExpiresAt should not be expired")
	}
}

// ── Set / Get ────────────────────────────────────────────────────────────────

func TestTokens_SetGet(t *testing.T) {
	toks := &config.Tokens{}
	ts := &config.TokenSet{
		AccessToken:  "access-abc",
		RefreshToken: "refresh-xyz",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		Scope:        "mfc/invoice/data.read",
	}

	toks.Set("default", "invoice", ts)

	got, err := toks.Get("default", "invoice")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected token, got nil")
	}
	if got.AccessToken != "access-abc" {
		t.Errorf("AccessToken = %q, want %q", got.AccessToken, "access-abc")
	}
	if got.Scope != "mfc/invoice/data.read" {
		t.Errorf("Scope = %q, want %q", got.Scope, "mfc/invoice/data.read")
	}
}

func TestTokens_GetMissingProfile(t *testing.T) {
	toks := &config.Tokens{}
	got, err := toks.Get("nonexistent", "invoice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing profile, got %+v", got)
	}
}

func TestTokens_GetMissingService(t *testing.T) {
	toks := &config.Tokens{}
	toks.Set("default", "invoice", &config.TokenSet{AccessToken: "x"})

	got, err := toks.Get("default", "expense")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for missing service")
	}
}

// ── Multi-service / Multi-profile ────────────────────────────────────────────

func TestTokens_MultiServiceIsolation(t *testing.T) {
	toks := &config.Tokens{}
	toks.Set("default", "invoice", &config.TokenSet{AccessToken: "inv"})
	toks.Set("default", "expense", &config.TokenSet{AccessToken: "exp"})
	toks.Set("other", "invoice", &config.TokenSet{AccessToken: "other-inv"})

	cases := []struct {
		profile, service, want string
	}{
		{"default", "invoice", "inv"},
		{"default", "expense", "exp"},
		{"other", "invoice", "other-inv"},
	}
	for _, tc := range cases {
		got, err := toks.Get(tc.profile, tc.service)
		if err != nil {
			t.Fatalf("Get(%q, %q): %v", tc.profile, tc.service, err)
		}
		if got == nil || got.AccessToken != tc.want {
			t.Errorf("Get(%q, %q).AccessToken = %q, want %q",
				tc.profile, tc.service, got.AccessToken, tc.want)
		}
	}
}

// ── Delete ───────────────────────────────────────────────────────────────────

func TestTokens_Delete(t *testing.T) {
	toks := &config.Tokens{}
	toks.Set("default", "invoice", &config.TokenSet{AccessToken: "x"})
	toks.Set("default", "expense", &config.TokenSet{AccessToken: "y"})

	toks.Delete("default", "invoice")

	if got, _ := toks.Get("default", "invoice"); got != nil {
		t.Error("expected nil after Delete(invoice)")
	}
	if got, _ := toks.Get("default", "expense"); got == nil {
		t.Error("expense token should survive Delete(invoice)")
	}
}

func TestTokens_DeleteProfile(t *testing.T) {
	toks := &config.Tokens{}
	toks.Set("default", "invoice", &config.TokenSet{AccessToken: "x"})
	toks.Set("other", "invoice", &config.TokenSet{AccessToken: "y"})

	toks.DeleteProfile("default")

	if got, _ := toks.Get("default", "invoice"); got != nil {
		t.Error("expected nil after DeleteProfile(default)")
	}
	if got, _ := toks.Get("other", "invoice"); got == nil {
		t.Error("other profile should survive DeleteProfile(default)")
	}
}

// ── ListServices ─────────────────────────────────────────────────────────────

func TestTokens_ListServices(t *testing.T) {
	toks := &config.Tokens{}
	toks.Set("default", "invoice", &config.TokenSet{})
	toks.Set("default", "expense", &config.TokenSet{})

	svcs := toks.ListServices("default")
	if len(svcs) != 2 {
		t.Errorf("ListServices returned %d services, want 2", len(svcs))
	}

	if svcs := toks.ListServices("unknown"); svcs != nil {
		t.Errorf("ListServices(unknown) = %v, want nil", svcs)
	}
}

// ── Save / Load round-trip ───────────────────────────────────────────────────

func TestTokens_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MF_CONFIG_DIR", dir)

	expiry := time.Now().Add(1 * time.Hour).Truncate(time.Second)

	toks := &config.Tokens{}
	toks.Set("default", "invoice", &config.TokenSet{
		AccessToken:  "access-abc",
		RefreshToken: "refresh-xyz",
		ExpiresAt:    expiry,
		TokenType:    "Bearer",
		Scope:        "mfc/invoice/data.read",
	})
	toks.Set("default", "expense", &config.TokenSet{AccessToken: "exp-token"})

	if err := toks.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file permissions.
	info, err := os.Stat(filepath.Join(dir, "tokens.yaml"))
	if err != nil {
		t.Fatalf("Stat tokens.yaml: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("tokens.yaml permissions = %04o, want 0600", perm)
	}

	loaded, err := config.LoadTokens()
	if err != nil {
		t.Fatalf("LoadTokens: %v", err)
	}

	got, _ := loaded.Get("default", "invoice")
	if got == nil {
		t.Fatal("invoice token missing after reload")
	}
	if got.AccessToken != "access-abc" {
		t.Errorf("AccessToken = %q, want %q", got.AccessToken, "access-abc")
	}
	if got.RefreshToken != "refresh-xyz" {
		t.Errorf("RefreshToken = %q, want %q", got.RefreshToken, "refresh-xyz")
	}
	if !got.ExpiresAt.Equal(expiry) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, expiry)
	}

	got2, _ := loaded.Get("default", "expense")
	if got2 == nil || got2.AccessToken != "exp-token" {
		t.Error("expense token not preserved in round-trip")
	}
}

func TestLoadTokens_MissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MF_CONFIG_DIR", dir)

	toks, err := config.LoadTokens()
	if err != nil {
		t.Fatalf("LoadTokens on missing file should not error: %v", err)
	}
	if toks == nil || toks.Profiles == nil {
		t.Error("expected empty Tokens with initialized Profiles map")
	}
}
