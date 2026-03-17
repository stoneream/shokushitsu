package track

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stoneream/shokushitsu/internal/tui/lib"
)

type taskSelectScreen struct {
	app *appState
}

func newTaskSelectScreen(app *appState) *taskSelectScreen {
	return &taskSelectScreen{app: app}
}

func (s *taskSelectScreen) Init(lib.Navigator) tea.Cmd {
	s.app.taskSelect.filter.Focus()
	return nil
}

func (s *taskSelectScreen) Update(msg tea.Msg, nav lib.Navigator) tea.Cmd {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "ctrl+c":
			return nav.Quit()
		case "esc":
			s.app.notice = ""
			s.app.result = Result{Action: ActionReturnHome}
			return nav.Quit()
		case "up", "k":
			s.app.taskSelect.cursor--
			s.app.clampTaskCursor()
			return nil
		case "down", "j":
			s.app.taskSelect.cursor++
			s.app.clampTaskCursor()
			return nil
		case "enter":
			choices := s.app.filteredTaskChoices()
			if len(choices) == 0 {
				return nil
			}

			choice := choices[s.app.taskSelect.cursor]
			if choice.isNew {
				if err := s.app.loadProjects(); err != nil {
					s.app.runErr = err
					return nav.Quit()
				}
				s.app.projectSelect.cursor = 0
				s.app.notice = ""
				return nav.Replace(newProjectSelectScreen(s.app))
			}

			if err := s.app.startSession(choice.task.ProjectID, choice.task.TaskID, choice.task.ProjectName, choice.task.TaskName); err != nil {
				s.app.runErr = err
				return nav.Quit()
			}
			return nav.Replace(newTrackingScreen(s.app))
		}
	}

	var cmd tea.Cmd
	s.app.taskSelect.filter, cmd = s.app.taskSelect.filter.Update(msg)
	s.app.clampTaskCursor()
	return cmd
}

func (s *taskSelectScreen) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("タスク選択"))
	b.WriteString("\n\n")
	b.WriteString(s.app.taskSelect.filter.View())
	b.WriteString("\n\n")

	for i, c := range s.app.filteredTaskChoices() {
		cursor := "  "
		if i == s.app.taskSelect.cursor {
			cursor = styleCursor.Render("> ")
		}

		if c.isNew {
			b.WriteString(cursor + styleNewAction.Render("[+] 新規タスクを作成") + "\n")
			continue
		}

		b.WriteString(fmt.Sprintf(
			"%s[%s] %s",
			cursor,
			styleProjectName.Render(c.task.ProjectName),
			styleTaskName.Render(c.task.TaskName),
		) + "\n")
	}

	if s.app.notice != "" {
		b.WriteString("\n" + styleNotice.Render(s.app.notice) + "\n")
	}

	b.WriteString("\nEnter: 決定  ↑/↓: 移動  Esc: 戻る  Ctrl+C: 終了")
	return b.String()
}
