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

func (screen *newProjectScreen) Init(lib.Navigator) tea.Cmd {
	screen.app.newProject.input.Focus()
	return nil
}

func (screen *newProjectScreen) Update(msg tea.Msg, nav lib.Navigator) tea.Cmd {
	switch keyMsg := msg.(type) {
	case tea.KeyMsg:
		switch keyMsg.String() {
		case "ctrl+c":
			return nav.Quit()
		case "esc":
			screen.app.notice = ""
			return nav.Replace(newProjectSelectScreen(screen.app))
		case "enter":
			name := strings.TrimSpace(screen.app.newProject.input.Value())
			if name == "" {
				screen.app.notice = "プロジェクト名を入力してください。"
				return nil
			}

			project, err := screen.app.store.GetOrCreateProject(screen.app.ctx, name)
			if err != nil {
				screen.app.notice = "プロジェクト作成に失敗しました。"
				return nil
			}

			screen.app.selectedProject = &project
			if err := screen.app.loadProjects(); err != nil {
				screen.app.runErr = err
				return nav.Quit()
			}
			screen.app.newTask.input.SetValue("")
			screen.app.newTask.input.Focus()
			screen.app.notice = ""
			return nav.Replace(newNewTaskScreen(screen.app))
		}
	}

	var cmd tea.Cmd
	screen.app.newProject.input, cmd = screen.app.newProject.input.Update(msg)
	return cmd
}

func (screen *newProjectScreen) View() string {
	var builder strings.Builder
	builder.WriteString(styleNewAction.Render("新規プロジェクト作成"))
	builder.WriteString("\n\n")
	builder.WriteString(screen.app.newProject.input.View())
	if screen.app.notice != "" {
		builder.WriteString("\n\n" + styleNotice.Render(screen.app.notice))
	}
	builder.WriteString("\n\nEnter: 作成  Esc: 戻る  Ctrl+C: 終了")
	return builder.String()
}
