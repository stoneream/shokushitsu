package track

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/stoneream/shokushitsu/internal/storage/sqlite"
	"github.com/stoneream/shokushitsu/internal/tui/lib"
)

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

type taskSelectState struct {
	filter textinput.Model
	cursor int
}

type projectSelectState struct {
	cursor int
}

type newProjectState struct {
	input textinput.Model
}

type newTaskState struct {
	input textinput.Model
}

type trackingState struct {
	now                  time.Time
	task                 string
	project              string
	lastPromptMin        int
	continueCheckActive  bool
	continueCheckDueTime time.Time
	firstCtrlCAt         time.Time
}

type appState struct {
	ctx   context.Context
	store *sqlite.Store

	tasks            []sqlite.RecentTask
	projects         []sqlite.Project
	selectedProject  *sqlite.Project
	taskSelect       taskSelectState
	projectSelect    projectSelectState
	newProject       newProjectState
	newTask          newTaskState
	currentSession   *sqlite.Session
	tracking         trackingState
	notice           string
	result           Result
	doneMessage      string
	runErr           error
	notificationPath string
}

func newApp(ctx context.Context, store *sqlite.Store, notificationSoundPath string) (*appState, lib.Screen, error) {
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

	app := &appState{
		ctx:   ctx,
		store: store,
		taskSelect: taskSelectState{
			filter: taskFilter,
		},
		newProject: newProjectState{
			input: projectInput,
		},
		newTask: newTaskState{
			input: taskInput,
		},
		notificationPath: notificationSoundPath,
	}

	if err := app.loadRecentTasks(); err != nil {
		return nil, nil, err
	}

	initialScreen := lib.Screen(newTaskSelectScreen(app))
	if len(app.tasks) == 0 {
		if err := app.loadProjects(); err != nil {
			return nil, nil, err
		}
		app.notice = "タスクが未登録のため、新規作成から開始します。"
		initialScreen = newProjectSelectScreen(app)
	}

	return app, initialScreen, nil
}

func (a *appState) loadRecentTasks() error {
	tasks, err := a.store.ListRecentTasks(a.ctx)
	if err != nil {
		return fmt.Errorf("load recent tasks: %w", err)
	}
	a.tasks = tasks
	a.clampTaskCursor()
	return nil
}

func (a *appState) loadProjects() error {
	projects, err := a.store.ListProjects(a.ctx)
	if err != nil {
		return fmt.Errorf("load projects: %w", err)
	}
	a.projects = projects
	a.clampProjectCursor()
	return nil
}

func (a *appState) filteredTaskChoices() []taskChoice {
	query := strings.TrimSpace(strings.ToLower(a.taskSelect.filter.Value()))
	choices := []taskChoice{{isNew: true}}

	for _, t := range a.tasks {
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

func (a *appState) projectChoices() []projectChoice {
	choices := []projectChoice{{isNew: true}}
	for _, p := range a.projects {
		choices = append(choices, projectChoice{project: p})
	}
	return choices
}

func (a *appState) clampTaskCursor() {
	max := len(a.filteredTaskChoices()) - 1
	if max < 0 {
		a.taskSelect.cursor = 0
		return
	}
	if a.taskSelect.cursor < 0 {
		a.taskSelect.cursor = 0
	}
	if a.taskSelect.cursor > max {
		a.taskSelect.cursor = max
	}
}

func (a *appState) clampProjectCursor() {
	max := len(a.projectChoices()) - 1
	if max < 0 {
		a.projectSelect.cursor = 0
		return
	}
	if a.projectSelect.cursor < 0 {
		a.projectSelect.cursor = 0
	}
	if a.projectSelect.cursor > max {
		a.projectSelect.cursor = max
	}
}

func (a *appState) startSession(projectID, taskID int64, projectName, taskName string) error {
	now := time.Now().UTC()
	session, err := a.store.StartSession(a.ctx, projectID, taskID, now)
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}

	a.currentSession = &session
	a.tracking.now = now
	a.tracking.project = projectName
	a.tracking.task = taskName
	a.tracking.lastPromptMin = -1
	a.tracking.continueCheckActive = false
	a.tracking.continueCheckDueTime = time.Time{}
	a.tracking.firstCtrlCAt = time.Time{}
	a.notice = ""
	a.doneMessage = ""
	return nil
}

func (a *appState) resetTracking() {
	a.currentSession = nil
	a.tracking.now = time.Time{}
	a.tracking.task = ""
	a.tracking.project = ""
	a.tracking.lastPromptMin = -1
	a.tracking.continueCheckActive = false
	a.tracking.continueCheckDueTime = time.Time{}
	a.tracking.firstCtrlCAt = time.Time{}
}

func (a *appState) shouldTriggerContinueCheck() bool {
	if a.currentSession == nil || a.tracking.continueCheckActive {
		return false
	}

	elapsedMin := int(a.tracking.now.Sub(a.currentSession.StartedAt).Minutes())
	if elapsedMin <= 0 {
		return false
	}

	intervalMin := int(remindInterval / time.Minute)
	if elapsedMin%intervalMin != 0 {
		return false
	}

	if elapsedMin == a.tracking.lastPromptMin {
		return false
	}

	a.tracking.lastPromptMin = elapsedMin
	return true
}
