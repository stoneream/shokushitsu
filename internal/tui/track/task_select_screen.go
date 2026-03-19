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

func (screen *taskSelectScreen) Init(lib.Navigator) tea.Cmd {
	screen.app.taskSelect.filter.Focus()
	return nil
}

func (screen *taskSelectScreen) Update(msg tea.Msg, nav lib.Navigator) tea.Cmd {
	switch keyMsg := msg.(type) {
	case tea.KeyMsg:
		switch keyMsg.String() {
		case "ctrl+c":
			return nav.Quit()
		case "esc":
			screen.app.notice = ""
			screen.app.result = Result{Action: ActionReturnHome}
			return nav.Quit()
		case "up", "k":
			screen.app.taskSelect.cursor--
			screen.app.clampTaskCursor()
			return nil
		case "down", "j":
			screen.app.taskSelect.cursor++
			screen.app.clampTaskCursor()
			return nil
		case "enter":
			choices := screen.app.filteredTaskChoices()
			if len(choices) == 0 {
				return nil
			}

			choice := choices[screen.app.taskSelect.cursor]
			if choice.isNew {
				if err := screen.app.loadProjects(); err != nil {
					screen.app.runErr = err
					return nav.Quit()
				}
				screen.app.projectSelect.cursor = 0
				screen.app.notice = ""
				return nav.Replace(newProjectSelectScreen(screen.app))
			}

			if err := screen.app.startSession(choice.task.ProjectID, choice.task.TaskID, choice.task.ProjectName, choice.task.TaskName); err != nil {
				screen.app.runErr = err
				return nav.Quit()
			}
			return nav.Replace(newTrackingScreen(screen.app))
		}
	}

	var cmd tea.Cmd
	screen.app.taskSelect.filter, cmd = screen.app.taskSelect.filter.Update(msg)
	screen.app.clampTaskCursor()
	return cmd
}

func (screen *taskSelectScreen) View() string {
	var builder strings.Builder
	builder.WriteString(styleTitle.Render("タスク選択"))
	builder.WriteString("\n\n")
	builder.WriteString(screen.app.taskSelect.filter.View())
	builder.WriteString("\n\n")

	for index, choice := range screen.app.filteredTaskChoices() {
		cursor := "  "
		if index == screen.app.taskSelect.cursor {
			cursor = styleCursor.Render("> ")
		}

		if choice.isNew {
			builder.WriteString(cursor + styleNewAction.Render("[+] 新規タスクを作成") + "\n")
			continue
		}

		builder.WriteString(fmt.Sprintf(
			"%s[%s] %s",
			cursor,
			styleProjectName.Render(choice.task.ProjectName),
			styleTaskName.Render(choice.task.TaskName),
		) + "\n")
	}

	if screen.app.notice != "" {
		builder.WriteString("\n" + styleNotice.Render(screen.app.notice) + "\n")
	}

	builder.WriteString("\nEnter: 決定  ↑/↓: 移動  Esc: 戻る  Ctrl+C: 終了")
	return builder.String()
}
