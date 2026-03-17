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

func (s *menuScreen) Init(lib.Navigator) tea.Cmd {
	return nil
}

func (s *menuScreen) Update(msg tea.Msg, nav lib.Navigator) tea.Cmd {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "up", "k":
			s.app.menuCursor--
			if s.app.menuCursor < 0 {
				s.app.menuCursor = 0
			}
		case "down", "j":
			s.app.menuCursor++
			if s.app.menuCursor >= len(s.app.choices) {
				s.app.menuCursor = len(s.app.choices) - 1
			}
		case "enter":
			action := s.app.choices[s.app.menuCursor].action
			if action == ActionSummary {
				return nav.Replace(newSummaryDateScreen(s.app))
			}

			s.app.result = Result{Action: action}
			return nav.Quit()
		case "ctrl+c":
			s.app.result = Result{Action: ActionQuit}
			return nav.Quit()
		}
	}

	return nil
}

func (s *menuScreen) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("shoku"))
	b.WriteString("\n")
	b.WriteString(stylePromptText.Render("実行するコマンドを選択してください"))
	b.WriteString("\n\n")

	for i, c := range s.app.choices {
		cursor := "  "
		if i == s.app.menuCursor {
			cursor = styleCursor.Render("> ")
		}

		label := c.label
		switch c.action {
		case ActionTrack:
			label = styleTrack.Render(c.label)
		case ActionSummary:
			label = styleSummary.Render(c.label)
		case ActionQuit:
			label = styleQuit.Render(c.label)
		}

		line := fmt.Sprintf("%s%s  (%s)", cursor, label, styleGuide.Render(c.description))
		b.WriteString(line + "\n")
	}

	b.WriteString("\n" + styleGuide.Render("Enter: 決定  ↑/↓: 移動  Ctrl+C: 終了"))
	return b.String()
}

type summaryDateScreen struct {
	app    *appState
	cursor int
	notice string
}

func newSummaryDateScreen(app *appState) *summaryDateScreen {
	return &summaryDateScreen{app: app}
}

func (s *summaryDateScreen) Init(lib.Navigator) tea.Cmd {
	return nil
}

func (s *summaryDateScreen) Update(msg tea.Msg, nav lib.Navigator) tea.Cmd {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "up", "k":
			s.cursor--
			if s.cursor < 0 {
				s.cursor = 0
			}
		case "down", "j":
			s.cursor++
			if s.cursor >= len(s.app.summaryDates) {
				s.cursor = len(s.app.summaryDates) - 1
			}
			if s.cursor < 0 {
				s.cursor = 0
			}
		case "enter":
			if len(s.app.summaryDates) == 0 {
				s.notice = "選択可能な日付がありません"
				return nil
			}

			s.app.result = Result{
				Action:      ActionSummary,
				SummaryDate: s.app.summaryDates[s.cursor],
			}
			return nav.Quit()
		case "esc":
			return nav.Replace(newMenuScreen(s.app))
		case "ctrl+c":
			s.app.result = Result{Action: ActionQuit}
			return nav.Quit()
		}
	}

	return nil
}

func (s *summaryDateScreen) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("summary"))
	b.WriteString("\n")
	b.WriteString(stylePromptText.Render("集計日を選択してください（新しい順）"))
	b.WriteString("\n\n")

	if len(s.app.summaryDates) == 0 {
		b.WriteString(styleError.Render("サマリー対象の日付がありません"))
	} else {
		for i, d := range s.app.summaryDates {
			cursor := "  "
			if i == s.cursor {
				cursor = styleCursor.Render("> ")
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, styleSummary.Render(d)))
		}
	}

	if s.notice != "" {
		b.WriteString("\n")
		b.WriteString(styleError.Render(s.notice))
	}

	if len(s.app.summaryDates) == 0 {
		b.WriteString("\n\n" + styleGuide.Render("Esc: 戻る  Ctrl+C: 終了"))
	} else {
		b.WriteString("\n" + styleGuide.Render("Enter: 実行  Esc: 戻る  ↑/↓: 移動  Ctrl+C: 終了"))
	}
	return b.String()
}
