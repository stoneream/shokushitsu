package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing.toml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load should not fail for missing file: %v", err)
	}
	if cfg.NotificationSoundPath() != "" {
		t.Fatalf("expected empty sound path, got %q", cfg.NotificationSoundPath())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("config should be auto-created: %v", err)
	}
	if !strings.Contains(string(data), "[notification]") {
		t.Fatalf("default config content is unexpected: %q", string(data))
	}
}

func TestLoadNotificationSoundFromTable(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
[notification]
sound_file = "/tmp/custom.wav"
`
	if err := writeFile(path, content); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got := cfg.NotificationSoundPath(); got != "/tmp/custom.wav" {
		t.Fatalf("unexpected NotificationSoundPath: %q", got)
	}
}

func TestLoadNotificationSoundFromFlatKey(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	content := `notification_sound_file = '/tmp/flat.wav'`
	if err := writeFile(path, content); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got := cfg.NotificationSoundPath(); got != "/tmp/flat.wav" {
		t.Fatalf("unexpected NotificationSoundPath: %q", got)
	}
}

func TestTableValueTakesPrecedence(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
notification_sound_file = "/tmp/flat.wav"
[notification]
sound_file = "/tmp/table.wav"
`
	if err := writeFile(path, content); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got := cfg.NotificationSoundPath(); got != "/tmp/table.wav" {
		t.Fatalf("expected table key precedence, got %q", got)
	}
}

func TestResolveNotificationSoundPath(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join("/tmp", "shoku-home", "config.toml")
	relative := "sounds/notify.wav"
	got := ResolveNotificationSoundPath(configPath, relative)
	want := filepath.Clean("/tmp/shoku-home/sounds/notify.wav")
	if got != want {
		t.Fatalf("relative path resolution mismatch: got %q, want %q", got, want)
	}
}

func TestLoadWithInlineComment(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	content := `notification_sound_file = "/tmp/flat.wav" # comment`
	if err := writeFile(path, content); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got := cfg.NotificationSoundPath(); got != "/tmp/flat.wav" {
		t.Fatalf("unexpected NotificationSoundPath: %q", got)
	}
}

func TestInvalidTargetValueReturnsError(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	content := `notification_sound_file = "unterminated`
	if err := writeFile(path, content); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "invalid quoted string") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadDoesNotOverwriteExistingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	content := `notification_sound_file = "/tmp/existing.wav"`
	if err := writeFile(path, content); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got := cfg.NotificationSoundPath(); got != "/tmp/existing.wav" {
		t.Fatalf("unexpected NotificationSoundPath: %q", got)
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
