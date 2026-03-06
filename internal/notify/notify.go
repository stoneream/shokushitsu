package notify

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/gen2brain/beeep"
)

func SendAsync(title, message, soundPath string) {
	go func() {
		_ = Send(title, message, soundPath)
	}()
}

func Send(title, message, soundPath string) error {
	if err := beeep.Notify(title, message, ""); err != nil {
		return err
	}

	if soundPath == "" {
		return nil
	}

	if _, err := os.Stat(soundPath); err != nil {
		// Path configured but file is missing/unreadable -> no custom sound.
		return nil
	}

	if err := playSound(soundPath); err != nil {
		return fmt.Errorf("play notification sound: %w", err)
	}

	return nil
}

func playSound(path string) error {
	switch runtime.GOOS {
	case "darwin":
		player, err := exec.LookPath("afplay")
		if err != nil {
			return nil
		}
		return exec.Command(player, path).Start()
	case "linux":
		if player, err := exec.LookPath("paplay"); err == nil {
			return exec.Command(player, path).Start()
		}
		if player, err := exec.LookPath("aplay"); err == nil {
			return exec.Command(player, path).Start()
		}
		return nil
	case "windows":
		player, err := exec.LookPath("powershell")
		if err != nil {
			return nil
		}
		escapedPath := strings.ReplaceAll(path, "'", "''")
		script := fmt.Sprintf("(New-Object Media.SoundPlayer '%s').PlaySync()", escapedPath)
		return exec.Command(player, "-NoProfile", "-NonInteractive", "-Command", script).Start()
	default:
		return nil
	}
}
