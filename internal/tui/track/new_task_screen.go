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

func (s *newTaskScreen) Init(lib.Navigator) tea.Cmd {
	s.app.newTask.input.Focus()
	return nil
}

func (s *newTaskScreen) Update(msg tea.Msg, nav lib.Navigator) tea.Cmd {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "ctrl+c":
			return nav.Quit()
		case "esc":
			s.app.notice = ""
			return nav.Replace(newProjectSelectScreen(s.app))
		case "enter":
			name := strings.TrimSpace(s.app.newTask.input.Value())
			if name == "" {
				s.app.notice = "タスク名を入力してください。"
				return nil
			}
			if s.app.selectedProject == nil {
				s.app.notice = "プロジェクトを選択してください。"
				return nav.Replace(newProjectSelectScreen(s.app))
			}

			task, err := s.app.store.GetOrCreateTask(s.app.ctx, s.app.selectedProject.ID, name)
			if err != nil {
				s.app.notice = "タスク作成に失敗しました。"
				return nil
			}

			if err := s.app.startSession(s.app.selectedProject.ID, task.ID, s.app.selectedProject.Name, task.Name); err != nil {
				s.app.runErr = err
				return nav.Quit()
			}
			return nav.Replace(newTrackingScreen(s.app))
		}
	}

	var cmd tea.Cmd
	s.app.newTask.input, cmd = s.app.newTask.input.Update(msg)
	return cmd
}

func (s *newTaskScreen) View() string {
	var b strings.Builder
	b.WriteString(styleNewAction.Render("新規タスク作成"))
	b.WriteString("\n\n")
	if s.app.selectedProject != nil {
		b.WriteString(fmt.Sprintf("プロジェクト: %s\n\n", styleProjectName.Render(s.app.selectedProject.Name)))
	}
	b.WriteString(s.app.newTask.input.View())
	if s.app.notice != "" {
		b.WriteString("\n\n" + styleNotice.Render(s.app.notice))
	}
	b.WriteString("\n\nEnter: 作成して開始  Esc: 戻る  Ctrl+C: 終了")
	return b.String()
}
