package home

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stoneream/shokushitsu/internal/tui/lib"
	"github.com/stoneream/shokushitsu/internal/tui/theme"
)

type Action string

const (
	ActionTrack   Action = "track"
	ActionSummary Action = "summary"
	ActionQuit    Action = "quit"
)

type Result struct {
	Action      Action
	SummaryDate string
}

var (
	styleTitle      = lipgloss.NewStyle().Bold(true).Foreground(theme.Primary)
	styleCursor     = lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	styleTrack      = lipgloss.NewStyle().Foreground(theme.Accent)
	styleSummary    = lipgloss.NewStyle().Foreground(theme.Primary)
	styleQuit       = lipgloss.NewStyle().Foreground(theme.Muted)
	styleGuide      = lipgloss.NewStyle().Foreground(theme.Muted)
	stylePromptText = lipgloss.NewStyle().Foreground(theme.Attention)
	styleError      = lipgloss.NewStyle().Foreground(theme.Attention)
)

func Run(summaryDates []string) (Result, error) {
	app := &appState{
		choices: []choice{
			{label: "track", action: ActionTrack, description: "作業時間の計測を開始する"},
			{label: "summary", action: ActionSummary, description: "作業時間のサマリーを表示する"},
			{label: "終了", action: ActionQuit, description: "終了する"},
		},
		summaryDates: summaryDates,
	}

	model := lib.New(newMenuScreen(app), nil)
	if _, err := tea.NewProgram(model).Run(); err != nil {
		return Result{}, err
	}

	return app.result, nil
}

type choice struct {
	label       string
	action      Action
	description string
}

type appState struct {
	choices      []choice
	menuCursor   int
	summaryDates []string
	result       Result
}

type menuScreen struct {
	app *appState
}

func newMenuScreen(app *appState) *menuScreen {
	return &menuScreen{app: app}
}

func (screen *menuScreen) Init(lib.Navigator) tea.Cmd {
	return nil
}

func (screen *menuScreen) Update(msg tea.Msg, nav lib.Navigator) tea.Cmd {
	switch keyMsg := msg.(type) {
	case tea.KeyMsg:
		switch keyMsg.String() {
		case "up", "k":
			screen.app.menuCursor--
			if screen.app.menuCursor < 0 {
				screen.app.menuCursor = 0
			}
		case "down", "j":
			screen.app.menuCursor++
			if screen.app.menuCursor >= len(screen.app.choices) {
				screen.app.menuCursor = len(screen.app.choices) - 1
			}
		case "enter":
			action := screen.app.choices[screen.app.menuCursor].action
			if action == ActionSummary {
				return nav.Replace(newSummaryDateScreen(screen.app))
			}

			screen.app.result = Result{Action: action}
			return nav.Quit()
		case "ctrl+c":
			screen.app.result = Result{Action: ActionQuit}
			return nav.Quit()
		}
	}

	return nil
}

func (screen *menuScreen) View() string {
	var builder strings.Builder
	builder.WriteString(styleTitle.Render("shoku"))
	builder.WriteString("\n")
	builder.WriteString(stylePromptText.Render("実行するコマンドを選択してください"))
	builder.WriteString("\n\n")

	for index, choice := range screen.app.choices {
		cursor := "  "
		if index == screen.app.menuCursor {
			cursor = styleCursor.Render("> ")
		}

		label := choice.label
		switch choice.action {
		case ActionTrack:
			label = styleTrack.Render(choice.label)
		case ActionSummary:
			label = styleSummary.Render(choice.label)
		case ActionQuit:
			label = styleQuit.Render(choice.label)
		}

		line := fmt.Sprintf("%s%s  (%s)", cursor, label, styleGuide.Render(choice.description))
		builder.WriteString(line + "\n")
	}

	builder.WriteString("\n" + styleGuide.Render("Enter: 決定  ↑/↓: 移動  Ctrl+C: 終了"))
	return builder.String()
}

type summaryDateScreen struct {
	app    *appState
	cursor int
	notice string
}

func newSummaryDateScreen(app *appState) *summaryDateScreen {
	return &summaryDateScreen{app: app}
}

func (screen *summaryDateScreen) Init(lib.Navigator) tea.Cmd {
	return nil
}

func (screen *summaryDateScreen) Update(msg tea.Msg, nav lib.Navigator) tea.Cmd {
	switch keyMsg := msg.(type) {
	case tea.KeyMsg:
		switch keyMsg.String() {
		case "up", "k":
			screen.cursor--
			if screen.cursor < 0 {
				screen.cursor = 0
			}
		case "down", "j":
			screen.cursor++
			if screen.cursor >= len(screen.app.summaryDates) {
				screen.cursor = len(screen.app.summaryDates) - 1
			}
			if screen.cursor < 0 {
				screen.cursor = 0
			}
		case "enter":
			if len(screen.app.summaryDates) == 0 {
				screen.notice = "選択可能な日付がありません"
				return nil
			}

			screen.app.result = Result{
				Action:      ActionSummary,
				SummaryDate: screen.app.summaryDates[screen.cursor],
			}
			return nav.Quit()
		case "esc":
			return nav.Replace(newMenuScreen(screen.app))
		case "ctrl+c":
			screen.app.result = Result{Action: ActionQuit}
			return nav.Quit()
		}
	}

	return nil
}

func (screen *summaryDateScreen) View() string {
	var builder strings.Builder
	builder.WriteString(styleTitle.Render("summary"))
	builder.WriteString("\n")
	builder.WriteString(stylePromptText.Render("集計日を選択してください（新しい順）"))
	builder.WriteString("\n\n")

	if len(screen.app.summaryDates) == 0 {
		builder.WriteString(styleError.Render("サマリー対象の日付がありません"))
	} else {
		for index, summaryDate := range screen.app.summaryDates {
			cursor := "  "
			if index == screen.cursor {
				cursor = styleCursor.Render("> ")
			}
			builder.WriteString(fmt.Sprintf("%s%s\n", cursor, styleSummary.Render(summaryDate)))
		}
	}

	if screen.notice != "" {
		builder.WriteString("\n")
		builder.WriteString(styleError.Render(screen.notice))
	}

	if len(screen.app.summaryDates) == 0 {
		builder.WriteString("\n\n" + styleGuide.Render("Esc: 戻る  Ctrl+C: 終了"))
	} else {
		builder.WriteString("\n" + styleGuide.Render("Enter: 実行  Esc: 戻る  ↑/↓: 移動  Ctrl+C: 終了"))
	}
	return builder.String()
}
