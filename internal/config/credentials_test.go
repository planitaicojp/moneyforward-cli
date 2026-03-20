package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/planitaicojp/moneyforward-cli/internal/config"
)

// ── GetClientSecret / SetClientSecret ────────────────────────────────────────

func TestCredentials_SetGet(t *testing.T) {
	c := &config.Credentials{}
	c.SetClientSecret("default", "secret-abc")

	got := c.GetClientSecret("default")
	if got != "secret-abc" {
		t.Errorf("GetClientSecret = %q, want %q", got, "secret-abc")
	}
}

func TestCredentials_GetMissing(t *testing.T) {
	c := &config.Credentials{}
	got := c.GetClientSecret("nonexistent")
	if got != "" {
		t.Errorf("GetClientSecret(missing) = %q, want empty string", got)
	}
}

func TestCredentials_MultiProfile(t *testing.T) {
	c := &config.Credentials{}
	c.SetClientSecret("default", "secret-default")
	c.SetClientSecret("other", "secret-other")

	if got := c.GetClientSecret("default"); got != "secret-default" {
		t.Errorf("default = %q, want %q", got, "secret-default")
	}
	if got := c.GetClientSecret("other"); got != "secret-other" {
		t.Errorf("other = %q, want %q", got, "secret-other")
	}
}

// ── DeleteProfile ─────────────────────────────────────────────────────────────

func TestCredentials_DeleteProfile(t *testing.T) {
	c := &config.Credentials{}
	c.SetClientSecret("default", "secret-default")
	c.SetClientSecret("other", "secret-other")

	c.DeleteProfile("default")

	if got := c.GetClientSecret("default"); got != "" {
		t.Errorf("expected empty after DeleteProfile(default), got %q", got)
	}
	if got := c.GetClientSecret("other"); got != "secret-other" {
		t.Errorf("other profile should survive DeleteProfile(default), got %q", got)
	}
}

func TestCredentials_DeleteProfileNonexistent(t *testing.T) {
	c := &config.Credentials{}
	// Should not panic.
	c.DeleteProfile("nonexistent")
}

// ── Save / Load round-trip ───────────────────────────────────────────────────

func TestCredentials_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MF_CONFIG_DIR", dir)

	c := &config.Credentials{}
	c.SetClientSecret("default", "my-secret")
	c.SetClientSecret("other", "other-secret")

	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file permissions.
	info, err := os.Stat(filepath.Join(dir, "credentials.yaml"))
	if err != nil {
		t.Fatalf("Stat credentials.yaml: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("credentials.yaml permissions = %04o, want 0600", perm)
	}

	loaded, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}

	if got := loaded.GetClientSecret("default"); got != "my-secret" {
		t.Errorf("default secret = %q, want %q", got, "my-secret")
	}
	if got := loaded.GetClientSecret("other"); got != "other-secret" {
		t.Errorf("other secret = %q, want %q", got, "other-secret")
	}
}

func TestLoadCredentials_MissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MF_CONFIG_DIR", dir)

	c, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials on missing file should not error: %v", err)
	}
	if c == nil || c.Profiles == nil {
		t.Error("expected empty Credentials with initialized Profiles map")
	}
}
