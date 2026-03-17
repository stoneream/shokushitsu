package track

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(at time.Time) tea.Msg {
		return tickMsg{at: at}
	})
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}

	total := int64(d.Seconds())
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func formatMMSS(d time.Duration) string {
	total := int64(d.Seconds())
	if total < 0 {
		total = 0
	}

	m := total / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
