package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Notification Notification `toml:"notification"`

	// Backward-compatible flat key support.
	NotificationSoundFile string `toml:"notification_sound_file"`
}

type Notification struct {
	SoundFile string `toml:"sound_file"`
}

const defaultConfigToml = `# shoku 設定ファイル
#
# 通知音ファイルを設定したい場合のみ指定してください。
# 未設定またはファイルが見つからない場合は、通知音を鳴らしません。
[notification]
sound_file = ""
`

// Load reads config.toml. If missing, it creates a default config file.
func Load(path string) (Config, error) {
	var config Config

	if path == "" {
		return config, fmt.Errorf("config path is required")
	}

	if err := EnsureFile(path); err != nil {
		return config, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("read config %q: %w", path, err)
	}

	if err := parseConfigToml(string(data), &config); err != nil {
		return config, fmt.Errorf("parse config %q: %w", path, err)
	}

	return config, nil
}

// EnsureFile creates config.toml with default content when it does not exist.
// If already present, it does nothing.
func EnsureFile(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory %q: %w", dir, err)
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("create config %q: %w", path, err)
	}
	defer file.Close()

	if _, err := file.WriteString(defaultConfigToml); err != nil {
		return fmt.Errorf("write default config %q: %w", path, err)
	}

	return nil
}

func (c Config) NotificationSoundPath() string {
	if notificationSoundPath := strings.TrimSpace(c.Notification.SoundFile); notificationSoundPath != "" {
		return notificationSoundPath
	}

	return strings.TrimSpace(c.NotificationSoundFile)
}

// ResolveNotificationSoundPath expands ~/ and resolves relative paths against config file directory.
func ResolveNotificationSoundPath(configPath, rawPath string) string {
	notificationSoundPath := strings.TrimSpace(rawPath)
	if notificationSoundPath == "" {
		return ""
	}

	notificationSoundPath = expandHome(notificationSoundPath)
	if filepath.IsAbs(notificationSoundPath) {
		return filepath.Clean(notificationSoundPath)
	}

	baseDir := "."
	if configPath != "" {
		baseDir = filepath.Dir(configPath)
	}

	return filepath.Clean(filepath.Join(baseDir, notificationSoundPath))
}

func parseConfigToml(data string, config *Config) error {
	currentTable := ""
	lines := strings.Split(data, "\n")
	for lineIndex, line := range lines {
		content := strings.TrimSpace(stripInlineComment(line))
		if content == "" {
			continue
		}

		if strings.HasPrefix(content, "[") && strings.HasSuffix(content, "]") {
			currentTable = strings.TrimSpace(content[1 : len(content)-1])
			continue
		}

		key, value, ok := strings.Cut(content, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch {
		case isNotificationSoundFileKey(currentTable, key):
			soundFile, err := parseTomlString(value)
			if err != nil {
				return fmt.Errorf("line %d: %w", lineIndex+1, err)
			}
			config.NotificationSoundFile = soundFile
		case isNotificationSoundTableKey(currentTable, key):
			soundFile, err := parseTomlString(value)
			if err != nil {
				return fmt.Errorf("line %d: %w", lineIndex+1, err)
			}
			config.Notification.SoundFile = soundFile
		}
	}

	return nil
}

func isNotificationSoundFileKey(table, key string) bool {
	normalizedTable := strings.ToLower(strings.TrimSpace(table))
	normalizedKey := strings.ToLower(strings.TrimSpace(key))

	return normalizedTable == "" &&
		(normalizedKey == "notification_sound_file" || normalizedKey == "notification.sound_file")
}

func isNotificationSoundTableKey(table, key string) bool {
	normalizedTable := strings.ToLower(strings.TrimSpace(table))
	normalizedKey := strings.ToLower(strings.TrimSpace(key))

	return normalizedTable == "notification" && normalizedKey == "sound_file"
}

func parseTomlString(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	if strings.HasPrefix(raw, "\"") {
		unquotedValue, err := strconv.Unquote(raw)
		if err != nil {
			return "", fmt.Errorf("invalid quoted string %q", raw)
		}
		return unquotedValue, nil
	}

	if strings.HasPrefix(raw, "'") && strings.HasSuffix(raw, "'") && len(raw) >= 2 {
		return raw[1 : len(raw)-1], nil
	}

	// Fallback for unquoted values.
	return raw, nil
}

func stripInlineComment(s string) string {
	inSingle := false
	inDouble := false
	escaped := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}

		switch ch {
		case '\\':
			if inDouble {
				escaped = true
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '#':
			if !inSingle && !inDouble {
				return s[:i]
			}
		}
	}

	return s
}

func expandHome(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}

	if !strings.HasPrefix(path, "~/") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return filepath.Join(home, strings.TrimPrefix(path, "~/"))
}
