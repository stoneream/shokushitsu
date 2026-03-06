package track

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stoneream/shokushitsu/internal/notify"
	"github.com/stoneream/shokushitsu/internal/storage/sqlite"
)

const (
	forceQuitWindow    = 5 * time.Second
	remindInterval     = 25 * time.Minute
	continueTimeout    = 15 * time.Minute
	notifyTitlePrompt  = "お仕事してるかね？"
	notifyTitleKeepRun = "shoku"
)

var (
	styleTitle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	styleCursor      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	styleProjectName = lipgloss.NewStyle().Foreground(lipgloss.Color("111"))
	styleTaskName    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleNewAction   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("141"))
	styleNotice      = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
)

type screenState int

const (
	stateTaskSelect screenState = iota
	stateProjectSelect
	stateNewProject
	stateNewTask
	stateTracking
)

type model struct {
	ctx   context.Context
	store *sqlite.Store

	state screenState

	tasks      []sqlite.RecentTask
	taskFilter textinput.Model
	taskCursor int

	projects      []sqlite.Project
	projectCursor int

	projectInput textinput.Model
	taskInput    textinput.Model

	selectedProject *sqlite.Project

	currentSession *sqlite.Session
	trackingNow    time.Time
	trackingTask   string
	trackingProj   string
	lastPromptMin  int

	continueCheckActive  bool
	continueCheckDueTime time.Time

	firstCtrlCAt time.Time
	notice       string

	doneMessage string
	runErr      error

	notificationSoundPath string
}

type tickMsg struct {
	at time.Time
}

type taskChoice struct {
	isNew bool
	task  sqlite.RecentTask
}

type projectChoice struct {
	isNew   bool
	project sqlite.Project
}

func Run(ctx context.Context, store *sqlite.Store, notificationSoundPath string) (string, error) {
	m, err := newModel(ctx, store, notificationSoundPath)
	if err != nil {
		return "", err
	}

	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return "", err
	}

	fm, ok := final.(*model)
	if !ok {
		return "", fmt.Errorf("unexpected model type: %T", final)
	}
	if fm.runErr != nil {
		return "", fm.runErr
	}

	return fm.doneMessage, nil
}

func newModel(ctx context.Context, store *sqlite.Store, notificationSoundPath string) (*model, error) {
	taskFilter := textinput.New()
	taskFilter.Prompt = "検索> "
	taskFilter.Placeholder = "タスク名・プロジェクト名で絞り込み"
	taskFilter.Focus()
	taskFilter.CharLimit = 200

	projectInput := textinput.New()
	projectInput.Prompt = "プロジェクト名> "
	projectInput.CharLimit = 200

	taskInput := textinput.New()
	taskInput.Prompt = "タスク名> "
	taskInput.CharLimit = 200

	m := &model{
		ctx:                   ctx,
		store:                 store,
		state:                 stateTaskSelect,
		taskFilter:            taskFilter,
		projectInput:          projectInput,
		taskInput:             taskInput,
		notificationSoundPath: notificationSoundPath,
	}

	if err := m.loadRecentTasks(); err != nil {
		return nil, err
	}

	if len(m.tasks) == 0 {
		if err := m.loadProjects(); err != nil {
			return nil, err
		}
		m.state = stateProjectSelect
		m.notice = "タスクが未登録のため、新規作成から開始します。"
	}

	return m, nil
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateTaskSelect:
		return m.updateTaskSelect(msg)
	case stateProjectSelect:
		return m.updateProjectSelect(msg)
	case stateNewProject:
		return m.updateNewProject(msg)
	case stateNewTask:
		return m.updateNewTask(msg)
	case stateTracking:
		return m.updateTracking(msg)
	default:
		return m, nil
	}
}

func (m *model) updateTaskSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			m.taskCursor--
			m.clampTaskCursor()
			return m, nil
		case "down", "j":
			m.taskCursor++
			m.clampTaskCursor()
			return m, nil
		case "enter":
			choices := m.filteredTaskChoices()
			if len(choices) == 0 {
				return m, nil
			}

			choice := choices[m.taskCursor]
			if choice.isNew {
				if err := m.loadProjects(); err != nil {
					m.runErr = err
					return m, tea.Quit
				}
				m.projectCursor = 0
				m.notice = ""
				m.state = stateProjectSelect
				return m, nil
			}

			if err := m.startSession(choice.task.ProjectID, choice.task.TaskID, choice.task.ProjectName, choice.task.TaskName); err != nil {
				m.runErr = err
				return m, tea.Quit
			}
			return m, tickCmd()
		}
	}

	var cmd tea.Cmd
	m.taskFilter, cmd = m.taskFilter.Update(msg)
	m.clampTaskCursor()
	return m, cmd
}

func (m *model) updateProjectSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if len(m.tasks) > 0 {
				m.state = stateTaskSelect
				m.notice = ""
			}
			return m, nil
		case "up", "k":
			m.projectCursor--
			m.clampProjectCursor()
			return m, nil
		case "down", "j":
			m.projectCursor++
			m.clampProjectCursor()
			return m, nil
		case "enter":
			choices := m.projectChoices()
			if len(choices) == 0 {
				return m, nil
			}

			choice := choices[m.projectCursor]
			m.notice = ""
			if choice.isNew {
				m.projectInput.SetValue("")
				m.projectInput.Focus()
				m.state = stateNewProject
				return m, nil
			}

			project := choice.project
			m.selectedProject = &project
			m.taskInput.SetValue("")
			m.taskInput.Focus()
			m.state = stateNewTask
			return m, nil
		}
	}

	return m, nil
}

func (m *model) updateNewProject(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.notice = ""
			m.state = stateProjectSelect
			return m, nil
		case "enter":
			name := strings.TrimSpace(m.projectInput.Value())
			if name == "" {
				m.notice = "プロジェクト名を入力してください。"
				return m, nil
			}

			project, err := m.store.GetOrCreateProject(m.ctx, name)
			if err != nil {
				m.notice = "プロジェクト作成に失敗しました。"
				return m, nil
			}

			m.selectedProject = &project
			if err := m.loadProjects(); err != nil {
				m.runErr = err
				return m, tea.Quit
			}
			m.taskInput.SetValue("")
			m.taskInput.Focus()
			m.notice = ""
			m.state = stateNewTask
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.projectInput, cmd = m.projectInput.Update(msg)
	return m, cmd
}

func (m *model) updateNewTask(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.notice = ""
			m.state = stateProjectSelect
			return m, nil
		case "enter":
			name := strings.TrimSpace(m.taskInput.Value())
			if name == "" {
				m.notice = "タスク名を入力してください。"
				return m, nil
			}
			if m.selectedProject == nil {
				m.notice = "プロジェクトを選択してください。"
				m.state = stateProjectSelect
				return m, nil
			}

			task, err := m.store.GetOrCreateTask(m.ctx, m.selectedProject.ID, name)
			if err != nil {
				m.notice = "タスク作成に失敗しました。"
				return m, nil
			}

			if err := m.startSession(m.selectedProject.ID, task.ID, m.selectedProject.Name, task.Name); err != nil {
				m.runErr = err
				return m, tea.Quit
			}
			return m, tickCmd()
		}
	}

	var cmd tea.Cmd
	m.taskInput, cmd = m.taskInput.Update(msg)
	return m, cmd
}

func (m *model) updateTracking(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tickMsg:
		m.trackingNow = v.at
		if !m.firstCtrlCAt.IsZero() && m.trackingNow.Sub(m.firstCtrlCAt) > forceQuitWindow {
			m.firstCtrlCAt = time.Time{}
			if !m.continueCheckActive {
				m.notice = ""
			}
		}

		if m.continueCheckActive && !m.continueCheckDueTime.IsZero() && !m.trackingNow.Before(m.continueCheckDueTime) {
			m.continueCheckActive = false
			m.continueCheckDueTime = time.Time{}
			m.notice = "15分間操作がなかったため、作業を継続します。"
			notify.SendAsync(
				notifyTitleKeepRun,
				fmt.Sprintf("引き続き [%s] %s を継続します。", m.trackingProj, m.trackingTask),
				m.notificationSoundPath,
			)
		}

		if m.shouldTriggerContinueCheck() {
			m.continueCheckActive = true
			m.continueCheckDueTime = m.trackingNow.Add(continueTimeout)
			m.notice = "25分経過しました。cで継続、eで終了します。"
			notify.SendAsync(
				notifyTitlePrompt,
				"25分経過しました。作業を継続しますか？",
				m.notificationSoundPath,
			)
		}

		return m, tickCmd()
	case tea.KeyMsg:
		switch v.String() {
		case "e":
			if m.currentSession == nil {
				m.runErr = fmt.Errorf("active session is missing")
				return m, tea.Quit
			}

			endedAt := time.Now().UTC()
			if err := m.store.EndSession(m.ctx, m.currentSession.ID, endedAt); err != nil {
				m.runErr = fmt.Errorf("end session: %w", err)
				return m, tea.Quit
			}

			message := fmt.Sprintf(
				"セッションを終了しました: %s / %s (%s)",
				m.trackingProj,
				m.trackingTask,
				formatDuration(endedAt.Sub(m.currentSession.StartedAt)),
			)
			if err := m.loadRecentTasks(); err != nil {
				m.runErr = err
				return m, tea.Quit
			}

			m.currentSession = nil
			m.trackingNow = time.Time{}
			m.trackingTask = ""
			m.trackingProj = ""
			m.lastPromptMin = -1
			m.continueCheckActive = false
			m.continueCheckDueTime = time.Time{}
			m.firstCtrlCAt = time.Time{}
			m.doneMessage = ""
			m.notice = message
			m.state = stateTaskSelect
			m.taskFilter.Focus()
			m.clampTaskCursor()
			return m, nil
		case "ctrl+c":
			now := time.Now()
			if m.firstCtrlCAt.IsZero() || now.Sub(m.firstCtrlCAt) > forceQuitWindow {
				m.firstCtrlCAt = now
				m.notice = "再度 Ctrl+C を押すと強制終了します。"
				return m, nil
			}

			m.doneMessage = "強制終了しました。セッションは未終了のままです。"
			return m, tea.Quit
		case "c":
			if !m.continueCheckActive {
				return m, nil
			}
			m.continueCheckActive = false
			m.continueCheckDueTime = time.Time{}
			m.notice = "作業を継続します。"
			notify.SendAsync(
				notifyTitleKeepRun,
				fmt.Sprintf("引き続き [%s] %s を継続します。", m.trackingProj, m.trackingTask),
				m.notificationSoundPath,
			)
			return m, nil
		}
	}

	return m, nil
}

func (m *model) View() string {
	switch m.state {
	case stateTaskSelect:
		return m.viewTaskSelect()
	case stateProjectSelect:
		return m.viewProjectSelect()
	case stateNewProject:
		return m.viewNewProject()
	case stateNewTask:
		return m.viewNewTask()
	case stateTracking:
		return m.viewTracking()
	default:
		return ""
	}
}

func (m *model) viewTaskSelect() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("タスク選択"))
	b.WriteString("\n\n")
	b.WriteString(m.taskFilter.View())
	b.WriteString("\n\n")

	for i, c := range m.filteredTaskChoices() {
		cursor := "  "
		if i == m.taskCursor {
			cursor = styleCursor.Render("> ")
		}

		if c.isNew {
			b.WriteString(cursor + styleNewAction.Render("[+] 新規タスクを作成") + "\n")
			continue
		}

		b.WriteString(fmt.Sprintf(
			"%s[%s] %s",
			cursor,
			styleProjectName.Render(c.task.ProjectName),
			styleTaskName.Render(c.task.TaskName),
		) + "\n")
	}

	if m.notice != "" {
		b.WriteString("\n" + styleNotice.Render(m.notice) + "\n")
	}

	b.WriteString("\nEnter: 決定  ↑/↓: 移動  Ctrl+C: 終了")
	return b.String()
}

func (m *model) viewProjectSelect() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("プロジェクト選択"))
	b.WriteString("\n\n")

	for i, c := range m.projectChoices() {
		cursor := "  "
		if i == m.projectCursor {
			cursor = styleCursor.Render("> ")
		}

		if c.isNew {
			b.WriteString(cursor + styleNewAction.Render("[+] 新規プロジェクトを作成") + "\n")
			continue
		}

		b.WriteString(cursor + styleProjectName.Render(c.project.Name) + "\n")
	}

	if m.notice != "" {
		b.WriteString("\n" + styleNotice.Render(m.notice) + "\n")
	}

	b.WriteString("\nEnter: 決定  ↑/↓: 移動  Esc: 戻る  Ctrl+C: 終了")
	return b.String()
}

func (m *model) viewNewProject() string {
	var b strings.Builder
	b.WriteString(styleNewAction.Render("新規プロジェクト作成"))
	b.WriteString("\n\n")
	b.WriteString(m.projectInput.View())
	if m.notice != "" {
		b.WriteString("\n\n" + styleNotice.Render(m.notice))
	}
	b.WriteString("\n\nEnter: 作成  Esc: 戻る  Ctrl+C: 終了")
	return b.String()
}

func (m *model) viewNewTask() string {
	var b strings.Builder
	b.WriteString(styleNewAction.Render("新規タスク作成"))
	b.WriteString("\n\n")
	if m.selectedProject != nil {
		b.WriteString(fmt.Sprintf("プロジェクト: %s\n\n", styleProjectName.Render(m.selectedProject.Name)))
	}
	b.WriteString(m.taskInput.View())
	if m.notice != "" {
		b.WriteString("\n\n" + styleNotice.Render(m.notice))
	}
	b.WriteString("\n\nEnter: 作成して開始  Esc: 戻る  Ctrl+C: 終了")
	return b.String()
}

func (m *model) viewTracking() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("計測中"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("プロジェクト: %s\n", styleProjectName.Render(m.trackingProj)))
	b.WriteString(fmt.Sprintf("タスク: %s\n", styleTaskName.Render(m.trackingTask)))
	if m.currentSession != nil {
		b.WriteString(fmt.Sprintf("開始時刻: %s\n", m.currentSession.StartedAt.Local().Format("2006-01-02 15:04:05")))
		if !m.trackingNow.IsZero() {
			b.WriteString(fmt.Sprintf("経過時間: %s\n", formatDuration(m.trackingNow.Sub(m.currentSession.StartedAt))))
		}
	}
	if m.notice != "" {
		b.WriteString("\n" + styleNotice.Render(m.notice) + "\n")
	}
	if m.continueCheckActive {
		remaining := m.continueCheckDueTime.Sub(m.trackingNow)
		if remaining < 0 {
			remaining = 0
		}
		b.WriteString(fmt.Sprintf("\n継続確認中: cで継続 / eで終了（自動継続まで %s）", formatMMSS(remaining)))
	} else {
		b.WriteString("\n'e': 正常終了  Ctrl+C: 強制終了")
	}
	return b.String()
}

func (m *model) startSession(projectID, taskID int64, projectName, taskName string) error {
	now := time.Now().UTC()
	session, err := m.store.StartSession(m.ctx, projectID, taskID, now)
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}

	m.currentSession = &session
	m.trackingNow = now
	m.trackingProj = projectName
	m.trackingTask = taskName
	m.lastPromptMin = -1
	m.continueCheckActive = false
	m.continueCheckDueTime = time.Time{}
	m.firstCtrlCAt = time.Time{}
	m.notice = ""
	m.doneMessage = ""
	m.state = stateTracking
	return nil
}

func (m *model) loadRecentTasks() error {
	tasks, err := m.store.ListRecentTasks(m.ctx)
	if err != nil {
		return fmt.Errorf("load recent tasks: %w", err)
	}
	m.tasks = tasks
	m.clampTaskCursor()
	return nil
}

func (m *model) loadProjects() error {
	projects, err := m.store.ListProjects(m.ctx)
	if err != nil {
		return fmt.Errorf("load projects: %w", err)
	}
	m.projects = projects
	m.clampProjectCursor()
	return nil
}

func (m *model) filteredTaskChoices() []taskChoice {
	query := strings.TrimSpace(strings.ToLower(m.taskFilter.Value()))
	choices := []taskChoice{{isNew: true}}

	for _, t := range m.tasks {
		if query == "" {
			choices = append(choices, taskChoice{task: t})
			continue
		}

		target := strings.ToLower(t.ProjectName + " " + t.TaskName)
		if strings.Contains(target, query) {
			choices = append(choices, taskChoice{task: t})
		}
	}

	return choices
}

func (m *model) projectChoices() []projectChoice {
	choices := []projectChoice{{isNew: true}}
	for _, p := range m.projects {
		choices = append(choices, projectChoice{project: p})
	}
	return choices
}

func (m *model) clampTaskCursor() {
	max := len(m.filteredTaskChoices()) - 1
	if max < 0 {
		m.taskCursor = 0
		return
	}
	if m.taskCursor < 0 {
		m.taskCursor = 0
	}
	if m.taskCursor > max {
		m.taskCursor = max
	}
}

func (m *model) clampProjectCursor() {
	max := len(m.projectChoices()) - 1
	if max < 0 {
		m.projectCursor = 0
		return
	}
	if m.projectCursor < 0 {
		m.projectCursor = 0
	}
	if m.projectCursor > max {
		m.projectCursor = max
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(at time.Time) tea.Msg {
		return tickMsg{at: at}
	})
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}

	total := int64(d.Seconds())
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func formatMMSS(d time.Duration) string {
	total := int64(d.Seconds())
	if total < 0 {
		total = 0
	}

	m := total / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

func (m *model) shouldTriggerContinueCheck() bool {
	if m.currentSession == nil || m.continueCheckActive {
		return false
	}

	elapsedMin := int(m.trackingNow.Sub(m.currentSession.StartedAt).Minutes())
	if elapsedMin <= 0 {
		return false
	}

	intervalMin := int(remindInterval / time.Minute)
	if elapsedMin%intervalMin != 0 {
		return false
	}

	if elapsedMin == m.lastPromptMin {
		return false
	}

	m.lastPromptMin = elapsedMin
	return true
}
