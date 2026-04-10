package track

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stoneream/shokushitsu/internal/notify"
	"github.com/stoneream/shokushitsu/internal/tui/lib"
)

type trackingScreen struct {
	app *appState
}

func newTrackingScreen(app *appState) *trackingScreen {
	return &trackingScreen{app: app}
}

func (screen *trackingScreen) Init(lib.Navigator) tea.Cmd {
	return tickCmd()
}

func (screen *trackingScreen) Update(msg tea.Msg, nav lib.Navigator) tea.Cmd {
	switch message := msg.(type) {
	case tickMsg:
		screen.app.tracking.now = message.at
		if !screen.app.tracking.firstCtrlCAt.IsZero() && screen.app.tracking.now.Sub(screen.app.tracking.firstCtrlCAt) > forceQuitWindow {
			screen.app.tracking.firstCtrlCAt = time.Time{}
			if !screen.app.tracking.continueCheckActive {
				screen.app.notice = ""
			}
		}

		if screen.app.tracking.continueCheckActive &&
			!screen.app.tracking.continueCheckDueTime.IsZero() &&
			!screen.app.tracking.now.Before(screen.app.tracking.continueCheckDueTime) {
			screen.app.tracking.continueCheckActive = false
			screen.app.tracking.continueCheckDueTime = time.Time{}
			screen.app.notice = "15分間操作がなかったため、作業を継続します。"
			notify.SendAsync(
				notifyTitleKeepRun,
				fmt.Sprintf("引き続き [%s] %s を継続します。", screen.app.tracking.project, screen.app.tracking.task),
				"",
			)
		}

		if screen.app.shouldTriggerContinueCheck() {
			screen.app.tracking.continueCheckActive = true
			screen.app.tracking.continueCheckDueTime = screen.app.tracking.now.Add(continueTimeout)
			screen.app.notice = "25分経過しました。cで継続、eで終了します。"
			notify.SendAsync(
				notifyTitlePrompt,
				"25分経過しました。作業を継続しますか？",
				screen.app.notificationPath,
			)
		}

		return tickCmd()
	case tea.KeyMsg:
		switch message.String() {
		case "e":
			if screen.app.currentSession == nil {
				screen.app.runErr = fmt.Errorf("active session is missing")
				return nav.Quit()
			}

			endedAt := time.Now().UTC()
			if err := screen.app.store.EndSession(screen.app.ctx, screen.app.currentSession.ID, endedAt); err != nil {
				screen.app.runErr = fmt.Errorf("end session: %w", err)
				return nav.Quit()
			}

			message := fmt.Sprintf(
				"セッションを終了しました: %s / %s (%s)",
				screen.app.tracking.project,
				screen.app.tracking.task,
				formatDuration(endedAt.Sub(screen.app.currentSession.StartedAt)),
			)
			if err := screen.app.loadRecentTasks(); err != nil {
				screen.app.runErr = err
				return nav.Quit()
			}

			screen.app.resetTracking()
			screen.app.doneMessage = ""
			screen.app.notice = message
			screen.app.taskSelect.filter.Focus()
			screen.app.clampTaskCursor()
			return nav.Replace(newTaskSelectScreen(screen.app))
		case "ctrl+c":
			pressedAt := time.Now()
			if screen.app.tracking.firstCtrlCAt.IsZero() || pressedAt.Sub(screen.app.tracking.firstCtrlCAt) > forceQuitWindow {
				screen.app.tracking.firstCtrlCAt = pressedAt
				screen.app.notice = "再度 Ctrl+C を押すと強制終了します。"
				return nil
			}

			screen.app.doneMessage = "強制終了しました。セッションは未終了のままです。"
			return nav.Quit()
		case "c":
			if !screen.app.tracking.continueCheckActive {
				return nil
			}
			screen.app.tracking.continueCheckActive = false
			screen.app.tracking.continueCheckDueTime = time.Time{}
			screen.app.notice = "作業を継続します。"
			notify.SendAsync(
				notifyTitleKeepRun,
				fmt.Sprintf("引き続き [%s] %s を継続します。", screen.app.tracking.project, screen.app.tracking.task),
				screen.app.notificationPath,
			)
			return nil
		}
	}

	return nil
}

func (screen *trackingScreen) View() string {
	var builder strings.Builder
	builder.WriteString(styleTitle.Render("計測中"))
	builder.WriteString("\n\n")
	builder.WriteString(fmt.Sprintf("プロジェクト: %s\n", styleProjectName.Render(screen.app.tracking.project)))
	builder.WriteString(fmt.Sprintf("タスク: %s\n", styleTaskName.Render(screen.app.tracking.task)))
	if screen.app.currentSession != nil {
		builder.WriteString(fmt.Sprintf("開始時刻: %s\n", screen.app.currentSession.StartedAt.Local().Format("2006-01-02 15:04:05")))
		if !screen.app.tracking.now.IsZero() {
			builder.WriteString(fmt.Sprintf("経過時間: %s\n", formatDuration(screen.app.tracking.now.Sub(screen.app.currentSession.StartedAt))))
		}
	}
	if screen.app.notice != "" {
		builder.WriteString("\n" + styleNotice.Render(screen.app.notice) + "\n")
	}
	if screen.app.tracking.continueCheckActive {
		remaining := screen.app.tracking.continueCheckDueTime.Sub(screen.app.tracking.now)
		if remaining < 0 {
			remaining = 0
		}
		builder.WriteString(fmt.Sprintf("\n継続確認中: cで継続 / eで終了（自動継続まで %s）", formatMMSS(remaining)))
	} else {
		builder.WriteString("\n'e': 正常終了  Ctrl+C: 強制終了")
	}
	return builder.String()
}
