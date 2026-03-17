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

func (s *trackingScreen) Init(lib.Navigator) tea.Cmd {
	return tickCmd()
}

func (s *trackingScreen) Update(msg tea.Msg, nav lib.Navigator) tea.Cmd {
	switch v := msg.(type) {
	case tickMsg:
		s.app.tracking.now = v.at
		if !s.app.tracking.firstCtrlCAt.IsZero() && s.app.tracking.now.Sub(s.app.tracking.firstCtrlCAt) > forceQuitWindow {
			s.app.tracking.firstCtrlCAt = time.Time{}
			if !s.app.tracking.continueCheckActive {
				s.app.notice = ""
			}
		}

		if s.app.tracking.continueCheckActive &&
			!s.app.tracking.continueCheckDueTime.IsZero() &&
			!s.app.tracking.now.Before(s.app.tracking.continueCheckDueTime) {
			s.app.tracking.continueCheckActive = false
			s.app.tracking.continueCheckDueTime = time.Time{}
			s.app.notice = "15分間操作がなかったため、作業を継続します。"
			notify.SendAsync(
				notifyTitleKeepRun,
				fmt.Sprintf("引き続き [%s] %s を継続します。", s.app.tracking.project, s.app.tracking.task),
				s.app.notificationPath,
			)
		}

		if s.app.shouldTriggerContinueCheck() {
			s.app.tracking.continueCheckActive = true
			s.app.tracking.continueCheckDueTime = s.app.tracking.now.Add(continueTimeout)
			s.app.notice = "25分経過しました。cで継続、eで終了します。"
			notify.SendAsync(
				notifyTitlePrompt,
				"25分経過しました。作業を継続しますか？",
				s.app.notificationPath,
			)
		}

		return tickCmd()
	case tea.KeyMsg:
		switch v.String() {
		case "e":
			if s.app.currentSession == nil {
				s.app.runErr = fmt.Errorf("active session is missing")
				return nav.Quit()
			}

			endedAt := time.Now().UTC()
			if err := s.app.store.EndSession(s.app.ctx, s.app.currentSession.ID, endedAt); err != nil {
				s.app.runErr = fmt.Errorf("end session: %w", err)
				return nav.Quit()
			}

			message := fmt.Sprintf(
				"セッションを終了しました: %s / %s (%s)",
				s.app.tracking.project,
				s.app.tracking.task,
				formatDuration(endedAt.Sub(s.app.currentSession.StartedAt)),
			)
			if err := s.app.loadRecentTasks(); err != nil {
				s.app.runErr = err
				return nav.Quit()
			}

			s.app.resetTracking()
			s.app.doneMessage = ""
			s.app.notice = message
			s.app.taskSelect.filter.Focus()
			s.app.clampTaskCursor()
			return nav.Replace(newTaskSelectScreen(s.app))
		case "ctrl+c":
			now := time.Now()
			if s.app.tracking.firstCtrlCAt.IsZero() || now.Sub(s.app.tracking.firstCtrlCAt) > forceQuitWindow {
				s.app.tracking.firstCtrlCAt = now
				s.app.notice = "再度 Ctrl+C を押すと強制終了します。"
				return nil
			}

			s.app.doneMessage = "強制終了しました。セッションは未終了のままです。"
			return nav.Quit()
		case "c":
			if !s.app.tracking.continueCheckActive {
				return nil
			}
			s.app.tracking.continueCheckActive = false
			s.app.tracking.continueCheckDueTime = time.Time{}
			s.app.notice = "作業を継続します。"
			notify.SendAsync(
				notifyTitleKeepRun,
				fmt.Sprintf("引き続き [%s] %s を継続します。", s.app.tracking.project, s.app.tracking.task),
				s.app.notificationPath,
			)
			return nil
		}
	}

	return nil
}

func (s *trackingScreen) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("計測中"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("プロジェクト: %s\n", styleProjectName.Render(s.app.tracking.project)))
	b.WriteString(fmt.Sprintf("タスク: %s\n", styleTaskName.Render(s.app.tracking.task)))
	if s.app.currentSession != nil {
		b.WriteString(fmt.Sprintf("開始時刻: %s\n", s.app.currentSession.StartedAt.Local().Format("2006-01-02 15:04:05")))
		if !s.app.tracking.now.IsZero() {
			b.WriteString(fmt.Sprintf("経過時間: %s\n", formatDuration(s.app.tracking.now.Sub(s.app.currentSession.StartedAt))))
		}
	}
	if s.app.notice != "" {
		b.WriteString("\n" + styleNotice.Render(s.app.notice) + "\n")
	}
	if s.app.tracking.continueCheckActive {
		remaining := s.app.tracking.continueCheckDueTime.Sub(s.app.tracking.now)
		if remaining < 0 {
			remaining = 0
		}
		b.WriteString(fmt.Sprintf("\n継続確認中: cで継続 / eで終了（自動継続まで %s）", formatMMSS(remaining)))
	} else {
		b.WriteString("\n'e': 正常終了  Ctrl+C: 強制終了")
	}
	return b.String()
}
