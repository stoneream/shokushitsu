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

func (screen *projectSelectScreen) Init(lib.Navigator) tea.Cmd {
	return nil
}

func (screen *projectSelectScreen) Update(msg tea.Msg, nav lib.Navigator) tea.Cmd {
	switch keyMsg := msg.(type) {
	case tea.KeyMsg:
		switch keyMsg.String() {
		case "ctrl+c":
			return nav.Quit()
		case "esc":
			if len(screen.app.tasks) > 0 {
				screen.app.notice = ""
				return nav.Replace(newTaskSelectScreen(screen.app))
			}
			screen.app.notice = ""
			screen.app.result = Result{Action: ActionReturnHome}
			return nav.Quit()
		case "up", "k":
			screen.app.projectSelect.cursor--
			screen.app.clampProjectCursor()
			return nil
		case "down", "j":
			screen.app.projectSelect.cursor++
			screen.app.clampProjectCursor()
			return nil
		case "enter":
			choices := screen.app.projectChoices()
			if len(choices) == 0 {
				return nil
			}

			choice := choices[screen.app.projectSelect.cursor]
			screen.app.notice = ""
			if choice.isNew {
				screen.app.newProject.input.SetValue("")
				screen.app.newProject.input.Focus()
				return nav.Replace(newNewProjectScreen(screen.app))
			}

			project := choice.project
			screen.app.selectedProject = &project
			screen.app.newTask.input.SetValue("")
			screen.app.newTask.input.Focus()
			return nav.Replace(newNewTaskScreen(screen.app))
		}
	}

	return nil
}

func (screen *projectSelectScreen) View() string {
	var builder strings.Builder
	builder.WriteString(styleTitle.Render("プロジェクト選択"))
	builder.WriteString("\n\n")

	for index, choice := range screen.app.projectChoices() {
		cursor := "  "
		if index == screen.app.projectSelect.cursor {
			cursor = styleCursor.Render("> ")
		}

		if choice.isNew {
			builder.WriteString(cursor + styleNewAction.Render("[+] 新規プロジェクトを作成") + "\n")
			continue
		}

		builder.WriteString(cursor + styleProjectName.Render(choice.project.Name) + "\n")
	}

	if screen.app.notice != "" {
		builder.WriteString("\n" + styleNotice.Render(screen.app.notice) + "\n")
	}

	builder.WriteString("\nEnter: 決定  ↑/↓: 移動  Esc: 戻る  Ctrl+C: 終了")
	return builder.String()
}
