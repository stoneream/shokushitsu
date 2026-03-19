package track

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stoneream/shokushitsu/internal/storage/sqlite"
	"github.com/stoneream/shokushitsu/internal/tui/lib"
	"github.com/stoneream/shokushitsu/internal/tui/theme"
)

const (
	forceQuitWindow    = 5 * time.Second
	remindInterval     = 25 * time.Minute
	continueTimeout    = 15 * time.Minute
	notifyTitlePrompt  = "お仕事してるかね？"
	notifyTitleKeepRun = "shoku"
)

var (
	styleTitle       = lipgloss.NewStyle().Bold(true).Foreground(theme.Primary)
	styleCursor      = lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	styleProjectName = lipgloss.NewStyle().Foreground(theme.Primary)
	styleTaskName    = lipgloss.NewStyle().Foreground(theme.Accent)
	styleNewAction   = lipgloss.NewStyle().Bold(true).Foreground(theme.Attention)
	styleNotice      = lipgloss.NewStyle().Foreground(theme.Attention)
)

type Action string

const (
	ActionQuit       Action = "quit"
	ActionReturnHome Action = "return_home"
)

type Result struct {
	Action  Action
	Message string
}

func Run(ctx context.Context, store *sqlite.Store, notificationSoundPath string) (Result, error) {
	app, initialScreen, err := newApp(ctx, store, notificationSoundPath)
	if err != nil {
		return Result{}, err
	}

	model := lib.New(initialScreen, nil)
	if _, err := tea.NewProgram(model).Run(); err != nil {
		return Result{}, err
	}
	if app.runErr != nil {
		return Result{}, app.runErr
	}

	result := app.result
	if result.Action == "" {
		result.Action = ActionQuit
	}
	if result.Message == "" {
		result.Message = app.doneMessage
	}

	return result, nil
}
