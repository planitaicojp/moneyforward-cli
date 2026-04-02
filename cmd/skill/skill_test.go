package skill

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInstallCmd(t *testing.T) {
	t.Run("fails when git not found", func(t *testing.T) {
		t.Setenv("PATH", "/nonexistent")
		dir := t.TempDir()

		err := runInstall(dir)
		if err == nil {
			t.Fatal("expected error when git not found")
		}
		if err.Error() != "validation error: git is required to install skills" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails when already installed", func(t *testing.T) {
		dir := t.TempDir()
		skillDir := filepath.Join(dir, skillName)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}

		err := runInstall(dir)
		if err == nil {
			t.Fatal("expected error when already installed")
		}
		if err.Error() != "validation error: already installed, use 'mf skill update'" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("clones successfully", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git not available")
		}
		dir := t.TempDir()

		err := runInstall(dir)
		if err != nil {
			t.Skipf("skipping: remote repo not accessible: %v", err)
		}

		skillDir := filepath.Join(dir, skillName)
		if _, err := os.Stat(filepath.Join(skillDir, ".git")); os.IsNotExist(err) {
			t.Error("expected .git directory after install")
		}
	})
}

func TestRemoveCmd(t *testing.T) {
	t.Run("fails when not installed", func(t *testing.T) {
		dir := t.TempDir()

		err := runRemove(dir)
		if err == nil {
			t.Fatal("expected error when not installed")
		}
		if err.Error() != "validation error: not installed" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("removes successfully with no-input", func(t *testing.T) {
		dir := t.TempDir()
		skillDir := filepath.Join(dir, skillName)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("MF_NO_INPUT", "1")

		err := runRemove(dir)
		if err != nil {
			t.Fatalf("remove failed: %v", err)
		}

		if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
			t.Error("expected skill directory to be removed")
		}
	})
}

func TestUpdateCmd(t *testing.T) {
	t.Run("fails when not installed", func(t *testing.T) {
		dir := t.TempDir()

		err := runUpdate(dir)
		if err == nil {
			t.Fatal("expected error when not installed")
		}
		if err.Error() != "validation error: not installed, use 'mf skill install'" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails when not a git repo", func(t *testing.T) {
		dir := t.TempDir()
		skillDir := filepath.Join(dir, skillName)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}

		err := runUpdate(dir)
		if err == nil {
			t.Fatal("expected error when not a git repo")
		}
		if err.Error() != "validation error: not a git repository, remove and reinstall" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("pulls successfully", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git not available")
		}
		dir := t.TempDir()

		if err := runInstall(dir); err != nil {
			t.Skipf("install failed (remote repo may not exist): %v", err)
		}

		err := runUpdate(dir)
		if err != nil {
			t.Fatalf("update failed: %v", err)
		}
	})
}
