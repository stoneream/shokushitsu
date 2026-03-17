package track

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stoneream/shokushitsu/internal/tui/lib"
)

type newTaskScreen struct {
	app *appState
}

func newNewTaskScreen(app *appState) *newTaskScreen {
	return &newTaskScreen{app: app}
}

func (screen *newTaskScreen) Init(lib.Navigator) tea.Cmd {
	screen.app.newTask.input.Focus()
	return nil
}

func (screen *newTaskScreen) Update(msg tea.Msg, nav lib.Navigator) tea.Cmd {
	switch keyMsg := msg.(type) {
	case tea.KeyMsg:
		switch keyMsg.String() {
		case "ctrl+c":
			return nav.Quit()
		case "esc":
			screen.app.notice = ""
			return nav.Replace(newProjectSelectScreen(screen.app))
		case "enter":
			name := strings.TrimSpace(screen.app.newTask.input.Value())
			if name == "" {
				screen.app.notice = "タスク名を入力してください。"
				return nil
			}
			if screen.app.selectedProject == nil {
				screen.app.notice = "プロジェクトを選択してください。"
				return nav.Replace(newProjectSelectScreen(screen.app))
			}

			task, err := screen.app.store.GetOrCreateTask(screen.app.ctx, screen.app.selectedProject.ID, name)
			if err != nil {
				screen.app.notice = "タスク作成に失敗しました。"
				return nil
			}

			if err := screen.app.startSession(screen.app.selectedProject.ID, task.ID, screen.app.selectedProject.Name, task.Name); err != nil {
				screen.app.runErr = err
				return nav.Quit()
			}
			return nav.Replace(newTrackingScreen(screen.app))
		}
	}

	var cmd tea.Cmd
	screen.app.newTask.input, cmd = screen.app.newTask.input.Update(msg)
	return cmd
}

func (screen *newTaskScreen) View() string {
	var builder strings.Builder
	builder.WriteString(styleNewAction.Render("新規タスク作成"))
	builder.WriteString("\n\n")
	if screen.app.selectedProject != nil {
		builder.WriteString(fmt.Sprintf("プロジェクト: %s\n\n", styleProjectName.Render(screen.app.selectedProject.Name)))
	}
	builder.WriteString(screen.app.newTask.input.View())
	if screen.app.notice != "" {
		builder.WriteString("\n\n" + styleNotice.Render(screen.app.notice))
	}
	builder.WriteString("\n\nEnter: 作成して開始  Esc: 戻る  Ctrl+C: 終了")
	return builder.String()
}
