package track

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stoneream/shokushitsu/internal/tui/lib"
)

type projectSelectScreen struct {
	app *appState
}

func newProjectSelectScreen(app *appState) *projectSelectScreen {
	return &projectSelectScreen{app: app}
}

func (s *projectSelectScreen) Init(lib.Navigator) tea.Cmd {
	return nil
}

func (s *projectSelectScreen) Update(msg tea.Msg, nav lib.Navigator) tea.Cmd {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "ctrl+c":
			return nav.Quit()
		case "esc":
			if len(s.app.tasks) > 0 {
				s.app.notice = ""
				return nav.Replace(newTaskSelectScreen(s.app))
			}
			s.app.notice = ""
			s.app.result = Result{Action: ActionReturnHome}
			return nav.Quit()
		case "up", "k":
			s.app.projectSelect.cursor--
			s.app.clampProjectCursor()
			return nil
		case "down", "j":
			s.app.projectSelect.cursor++
			s.app.clampProjectCursor()
			return nil
		case "enter":
			choices := s.app.projectChoices()
			if len(choices) == 0 {
				return nil
			}

			choice := choices[s.app.projectSelect.cursor]
			s.app.notice = ""
			if choice.isNew {
				s.app.newProject.input.SetValue("")
				s.app.newProject.input.Focus()
				return nav.Replace(newNewProjectScreen(s.app))
			}

			project := choice.project
			s.app.selectedProject = &project
			s.app.newTask.input.SetValue("")
			s.app.newTask.input.Focus()
			return nav.Replace(newNewTaskScreen(s.app))
		}
	}

	return nil
}

func (s *projectSelectScreen) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("プロジェクト選択"))
	b.WriteString("\n\n")

	for i, c := range s.app.projectChoices() {
		cursor := "  "
		if i == s.app.projectSelect.cursor {
			cursor = styleCursor.Render("> ")
		}

		if c.isNew {
			b.WriteString(cursor + styleNewAction.Render("[+] 新規プロジェクトを作成") + "\n")
			continue
		}

		b.WriteString(cursor + styleProjectName.Render(c.project.Name) + "\n")
	}

	if s.app.notice != "" {
		b.WriteString("\n" + styleNotice.Render(s.app.notice) + "\n")
	}

	b.WriteString("\nEnter: 決定  ↑/↓: 移動  Esc: 戻る  Ctrl+C: 終了")
	return b.String()
}
