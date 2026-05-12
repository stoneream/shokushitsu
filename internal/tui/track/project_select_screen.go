package track

import (
	"fmt"
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
	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		screen.app.windowHeight = typedMsg.Height
		return nil
	case tea.KeyMsg:
		switch typedMsg.String() {
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
	choices := screen.app.projectChoices()
	pagination := lib.Paginate(len(choices), screen.app.projectSelect.cursor, screen.app.windowHeight, screen.projectSelectReservedLines())

	var builder strings.Builder
	builder.WriteString(styleTitle.Render("プロジェクト選択"))
	builder.WriteString("\n\n")

	for index := pagination.Start; index < pagination.End; index++ {
		choice := choices[index]
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

	builder.WriteString("\n" + screen.projectGuide(pagination, len(choices)))
	return builder.String()
}

func (screen *projectSelectScreen) projectSelectReservedLines() int {
	if screen.app.notice != "" {
		return 6
	}
	return 4
}

func (screen *projectSelectScreen) projectGuide(pagination lib.Pagination, total int) string {
	guide := "Enter: 決定  ↑/↓: 移動  Esc: 戻る  Ctrl+C: 終了"
	status := pagination.StatusLabel(total)
	if status == "" {
		return guide
	}
	return fmt.Sprintf("%s  (%s)", guide, status)
}
