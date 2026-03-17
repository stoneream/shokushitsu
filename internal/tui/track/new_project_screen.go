package track

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stoneream/shokushitsu/internal/tui/lib"
)

type newProjectScreen struct {
	app *appState
}

func newNewProjectScreen(app *appState) *newProjectScreen {
	return &newProjectScreen{app: app}
}

func (s *newProjectScreen) Init(lib.Navigator) tea.Cmd {
	s.app.newProject.input.Focus()
	return nil
}

func (s *newProjectScreen) Update(msg tea.Msg, nav lib.Navigator) tea.Cmd {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "ctrl+c":
			return nav.Quit()
		case "esc":
			s.app.notice = ""
			return nav.Replace(newProjectSelectScreen(s.app))
		case "enter":
			name := strings.TrimSpace(s.app.newProject.input.Value())
			if name == "" {
				s.app.notice = "プロジェクト名を入力してください。"
				return nil
			}

			project, err := s.app.store.GetOrCreateProject(s.app.ctx, name)
			if err != nil {
				s.app.notice = "プロジェクト作成に失敗しました。"
				return nil
			}

			s.app.selectedProject = &project
			if err := s.app.loadProjects(); err != nil {
				s.app.runErr = err
				return nav.Quit()
			}
			s.app.newTask.input.SetValue("")
			s.app.newTask.input.Focus()
			s.app.notice = ""
			return nav.Replace(newNewTaskScreen(s.app))
		}
	}

	var cmd tea.Cmd
	s.app.newProject.input, cmd = s.app.newProject.input.Update(msg)
	return cmd
}

func (s *newProjectScreen) View() string {
	var b strings.Builder
	b.WriteString(styleNewAction.Render("新規プロジェクト作成"))
	b.WriteString("\n\n")
	b.WriteString(s.app.newProject.input.View())
	if s.app.notice != "" {
		b.WriteString("\n\n" + styleNotice.Render(s.app.notice))
	}
	b.WriteString("\n\nEnter: 作成  Esc: 戻る  Ctrl+C: 終了")
	return b.String()
}
