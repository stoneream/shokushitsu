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
	lastPromptMinute     int
	hasPrompted          bool
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
		result: Result{
			Action: ActionQuit,
		},
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

func (app *appState) loadRecentTasks() error {
	tasks, err := app.store.ListRecentTasks(app.ctx)
	if err != nil {
		return fmt.Errorf("load recent tasks: %w", err)
	}
	app.tasks = tasks
	app.clampTaskCursor()
	return nil
}

func (app *appState) loadProjects() error {
	projects, err := app.store.ListProjects(app.ctx)
	if err != nil {
		return fmt.Errorf("load projects: %w", err)
	}
	app.projects = projects
	app.clampProjectCursor()
	return nil
}

func (app *appState) filteredTaskChoices() []taskChoice {
	query := strings.TrimSpace(strings.ToLower(app.taskSelect.filter.Value()))
	choices := []taskChoice{{isNew: true}}

	for _, recentTask := range app.tasks {
		if query == "" {
			choices = append(choices, taskChoice{task: recentTask})
			continue
		}

		searchTarget := strings.ToLower(recentTask.ProjectName + " " + recentTask.TaskName)
		if strings.Contains(searchTarget, query) {
			choices = append(choices, taskChoice{task: recentTask})
		}
	}

	return choices
}

func (app *appState) projectChoices() []projectChoice {
	choices := []projectChoice{{isNew: true}}
	for _, project := range app.projects {
		choices = append(choices, projectChoice{project: project})
	}
	return choices
}

func (app *appState) clampTaskCursor() {
	maxCursor := len(app.filteredTaskChoices()) - 1
	if maxCursor < 0 {
		app.taskSelect.cursor = 0
		return
	}
	if app.taskSelect.cursor < 0 {
		app.taskSelect.cursor = 0
	}
	if app.taskSelect.cursor > maxCursor {
		app.taskSelect.cursor = maxCursor
	}
}

func (app *appState) clampProjectCursor() {
	maxCursor := len(app.projectChoices()) - 1
	if maxCursor < 0 {
		app.projectSelect.cursor = 0
		return
	}
	if app.projectSelect.cursor < 0 {
		app.projectSelect.cursor = 0
	}
	if app.projectSelect.cursor > maxCursor {
		app.projectSelect.cursor = maxCursor
	}
}

func (app *appState) startSession(projectID, taskID int64, projectName, taskName string) error {
	now := time.Now().UTC()
	session, err := app.store.StartSession(app.ctx, projectID, taskID, now)
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}

	app.currentSession = &session
	app.tracking.now = now
	app.tracking.project = projectName
	app.tracking.task = taskName
	app.tracking.lastPromptMinute = 0
	app.tracking.hasPrompted = false
	app.tracking.continueCheckActive = false
	app.tracking.continueCheckDueTime = time.Time{}
	app.tracking.firstCtrlCAt = time.Time{}
	app.notice = ""
	return nil
}

func (app *appState) resetTracking() {
	app.currentSession = nil
	app.tracking.now = time.Time{}
	app.tracking.task = ""
	app.tracking.project = ""
	app.tracking.lastPromptMinute = 0
	app.tracking.hasPrompted = false
	app.tracking.continueCheckActive = false
	app.tracking.continueCheckDueTime = time.Time{}
	app.tracking.firstCtrlCAt = time.Time{}
}

func (app *appState) shouldTriggerContinueCheck() bool {
	if app.currentSession == nil || app.tracking.continueCheckActive {
		return false
	}

	elapsedMinutes := int(app.tracking.now.Sub(app.currentSession.StartedAt).Minutes())
	if elapsedMinutes <= 0 {
		return false
	}

	intervalMinutes := int(remindInterval / time.Minute)
	if elapsedMinutes%intervalMinutes != 0 {
		return false
	}

	if app.tracking.hasPrompted && elapsedMinutes == app.tracking.lastPromptMinute {
		return false
	}

	app.tracking.lastPromptMinute = elapsedMinutes
	app.tracking.hasPrompted = true
	return true
}
