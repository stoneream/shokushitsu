package home

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

type viewState int

const (
	stateMenu viewState = iota
	stateSummaryDateSelect
)

var (
	styleTitle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	styleCursor     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	styleTrack      = lipgloss.NewStyle().Foreground(lipgloss.Color("111"))
	styleSummary    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleQuit       = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleGuide      = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
	stylePromptText = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	styleError      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

func Run(summaryDates []string) (Result, error) {
	m := &model{
		choices: []choice{
			{label: "track", action: ActionTrack, description: "作業時間の計測を開始する"},
			{label: "summary", action: ActionSummary, description: "作業時間のサマリーを表示する"},
			{label: "終了", action: ActionQuit, description: "終了する"},
		},
		state:        stateMenu,
		summaryDates: summaryDates,
	}

	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return Result{}, err
	}

	fm, ok := final.(*model)
	if !ok {
		return Result{}, fmt.Errorf("unexpected model type: %T", final)
	}

	return Result{
		Action:      fm.selected,
		SummaryDate: fm.summaryDate,
	}, nil
}

type choice struct {
	label       string
	action      Action
	description string
}

type model struct {
	choices  []choice
	cursor   int
	selected Action

	state         viewState
	summaryDates  []string
	summaryCursor int
	summaryDate   string
	notice        string
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateMenu:
		return m.updateMenu(msg)
	case stateSummaryDateSelect:
		return m.updateSummaryDateSelect(msg)
	default:
		return m, nil
	}
}

func (m *model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = 0
			}
		case "down", "j":
			m.cursor++
			if m.cursor >= len(m.choices) {
				m.cursor = len(m.choices) - 1
			}
		case "enter":
			action := m.choices[m.cursor].action
			if action == ActionSummary {
				m.notice = ""
				m.summaryCursor = 0
				m.state = stateSummaryDateSelect
				return m, nil
			}

			m.selected = action
			return m, tea.Quit
		case "ctrl+c":
			m.selected = ActionQuit
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *model) updateSummaryDateSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "up", "k":
			m.summaryCursor--
			if m.summaryCursor < 0 {
				m.summaryCursor = 0
			}
			return m, nil
		case "down", "j":
			m.summaryCursor++
			if m.summaryCursor >= len(m.summaryDates) {
				m.summaryCursor = len(m.summaryDates) - 1
			}
			if m.summaryCursor < 0 {
				m.summaryCursor = 0
			}
			return m, nil
		case "enter":
			if len(m.summaryDates) == 0 {
				m.notice = "選択可能な日付がありません"
				return m, nil
			}
			m.summaryDate = m.summaryDates[m.summaryCursor]
			m.selected = ActionSummary
			return m, tea.Quit
		case "esc":
			m.notice = ""
			m.state = stateMenu
			return m, nil
		case "ctrl+c":
			m.selected = ActionQuit
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *model) View() string {
	switch m.state {
	case stateMenu:
		return m.viewMenu()
	case stateSummaryDateSelect:
		return m.viewSummaryDateSelect()
	default:
		return ""
	}
}

func (m *model) viewMenu() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("shoku"))
	b.WriteString("\n")
	b.WriteString(stylePromptText.Render("実行するコマンドを選択してください"))
	b.WriteString("\n\n")

	for i, c := range m.choices {
		cursor := "  "
		if i == m.cursor {
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

	if m.notice != "" {
		b.WriteString("\n")
		b.WriteString(styleError.Render(m.notice))
	}

	b.WriteString("\n" + styleGuide.Render("Enter: 決定  ↑/↓: 移動  Ctrl+C: 終了"))
	return b.String()
}

func (m *model) viewSummaryDateSelect() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("summary"))
	b.WriteString("\n")
	b.WriteString(stylePromptText.Render("集計日を選択してください（新しい順）"))
	b.WriteString("\n\n")

	if len(m.summaryDates) == 0 {
		b.WriteString(styleError.Render("サマリー対象の日付がありません"))
	} else {
		for i, d := range m.summaryDates {
			cursor := "  "
			if i == m.summaryCursor {
				cursor = styleCursor.Render("> ")
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, styleSummary.Render(d)))
		}
	}

	if m.notice != "" {
		b.WriteString("\n")
		b.WriteString(styleError.Render(m.notice))
	}

	if len(m.summaryDates) == 0 {
		b.WriteString("\n\n" + styleGuide.Render("Esc: 戻る  Ctrl+C: 終了"))
	} else {
		b.WriteString("\n" + styleGuide.Render("Enter: 実行  Esc: 戻る  ↑/↓: 移動  Ctrl+C: 終了"))
	}
	return b.String()
}
