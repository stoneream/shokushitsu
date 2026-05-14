package track

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stoneream/shokushitsu/internal/storage/sqlite"
	"github.com/stoneream/shokushitsu/internal/tui/lib"
)

func TestNewAppStartsAtProjectSelectWhenNoTasks(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer func() {
		_ = store.Close()
	}()

	app, initial, err := newApp(context.Background(), store, "")
	if err != nil {
		t.Fatalf("newApp failed: %v", err)
	}

	if _, ok := initial.(*projectSelectScreen); !ok {
		t.Fatalf("expected projectSelectScreen, got %T", initial)
	}
	if app.notice == "" {
		t.Fatal("expected initial notice when no tasks exist")
	}
	if app.result.Action != ActionQuit {
		t.Fatalf("expected default action %q, got %q", ActionQuit, app.result.Action)
	}
}

func TestTaskSelectEnterStartsTracking(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer func() {
		_ = store.Close()
	}()

	project, err := store.GetOrCreateProject(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("GetOrCreateProject failed: %v", err)
	}
	task, err := store.GetOrCreateTask(context.Background(), project.ID, "task-a")
	if err != nil {
		t.Fatalf("GetOrCreateTask failed: %v", err)
	}
	session, err := store.StartSession(context.Background(), project.ID, task.ID, time.Unix(1_700_001_000, 0).UTC())
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}
	if err := store.EndSession(context.Background(), session.ID, time.Unix(1_700_001_600, 0).UTC()); err != nil {
		t.Fatalf("EndSession failed: %v", err)
	}

	app, initial, err := newApp(context.Background(), store, "")
	if err != nil {
		t.Fatalf("newApp failed: %v", err)
	}
	app.taskSelect.cursor = 1

	model := lib.New(initial, nil)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(*lib.Model)

	if _, ok := next.Current().(*trackingScreen); !ok {
		t.Fatalf("expected trackingScreen, got %T", next.Current())
	}
	if app.currentSession == nil {
		t.Fatal("expected active session to be created")
	}
}

func TestProjectSelectEnterExistingMovesToNewTask(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer func() {
		_ = store.Close()
	}()

	project, err := store.GetOrCreateProject(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("GetOrCreateProject failed: %v", err)
	}

	app, initial, err := newApp(context.Background(), store, "")
	if err != nil {
		t.Fatalf("newApp failed: %v", err)
	}
	app.projectSelect.cursor = 1

	model := lib.New(initial, nil)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(*lib.Model)

	if _, ok := next.Current().(*newTaskScreen); !ok {
		t.Fatalf("expected newTaskScreen, got %T", next.Current())
	}
	if app.selectedProject == nil || app.selectedProject.ID != project.ID {
		t.Fatal("expected selected project to be set")
	}
}

func TestTaskSelectEscReturnsHome(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer func() {
		_ = store.Close()
	}()

	project, err := store.GetOrCreateProject(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("GetOrCreateProject failed: %v", err)
	}
	task, err := store.GetOrCreateTask(context.Background(), project.ID, "task-a")
	if err != nil {
		t.Fatalf("GetOrCreateTask failed: %v", err)
	}
	session, err := store.StartSession(context.Background(), project.ID, task.ID, time.Unix(1_700_001_000, 0).UTC())
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}
	if err := store.EndSession(context.Background(), session.ID, time.Unix(1_700_001_600, 0).UTC()); err != nil {
		t.Fatalf("EndSession failed: %v", err)
	}

	app, initial, err := newApp(context.Background(), store, "")
	if err != nil {
		t.Fatalf("newApp failed: %v", err)
	}

	model := lib.New(initial, nil)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	next := updated.(*lib.Model)

	if _, ok := next.Current().(*taskSelectScreen); !ok {
		t.Fatalf("expected taskSelectScreen, got %T", next.Current())
	}
	if app.result.Action != ActionReturnHome {
		t.Fatalf("expected action %q, got %q", ActionReturnHome, app.result.Action)
	}
}

func TestProjectSelectEscWithoutTasksReturnsHome(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer func() {
		_ = store.Close()
	}()

	app, initial, err := newApp(context.Background(), store, "")
	if err != nil {
		t.Fatalf("newApp failed: %v", err)
	}

	model := lib.New(initial, nil)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	next := updated.(*lib.Model)

	if _, ok := next.Current().(*projectSelectScreen); !ok {
		t.Fatalf("expected projectSelectScreen, got %T", next.Current())
	}
	if app.result.Action != ActionReturnHome {
		t.Fatalf("expected action %q, got %q", ActionReturnHome, app.result.Action)
	}
}

func TestProjectSelectViewPaginatesToWindowHeight(t *testing.T) {
	t.Parallel()

	app := &appState{
		windowHeight: 7,
		projects: []sqlite.Project{
			{ID: 1, Name: "alpha"},
			{ID: 2, Name: "beta"},
			{ID: 3, Name: "gamma"},
			{ID: 4, Name: "delta"},
		},
	}
	screen := newProjectSelectScreen(app)

	firstPage := screen.View()
	if !strings.Contains(firstPage, "alpha") || !strings.Contains(firstPage, "beta") {
		t.Fatalf("expected first page projects to be visible: %s", firstPage)
	}
	if strings.Contains(firstPage, "gamma") || strings.Contains(firstPage, "delta") {
		t.Fatalf("expected later page projects to be hidden: %s", firstPage)
	}

	app.projectSelect.cursor = 3
	secondPage := screen.View()
	if !strings.Contains(secondPage, "gamma") || !strings.Contains(secondPage, "delta") {
		t.Fatalf("expected second page projects to be visible: %s", secondPage)
	}
	if strings.Contains(secondPage, "alpha") || strings.Contains(secondPage, "beta") {
		t.Fatalf("expected first page projects to be hidden: %s", secondPage)
	}
	if !strings.Contains(secondPage, "(4-5/5)") {
		t.Fatalf("expected paging status to be shown: %s", secondPage)
	}
}

func TestTaskSelectViewPaginatesToWindowHeight(t *testing.T) {
	t.Parallel()

	filter := newAppFilterModel()
	app := &appState{
		windowHeight: 9,
		taskSelect: taskSelectState{
			filter: filter,
		},
		tasks: []sqlite.RecentTask{
			{ProjectName: "alpha", TaskName: "task-1"},
			{ProjectName: "alpha", TaskName: "task-2"},
			{ProjectName: "alpha", TaskName: "task-3"},
			{ProjectName: "alpha", TaskName: "task-4"},
		},
	}
	screen := newTaskSelectScreen(app)

	firstPage := screen.View()
	if !strings.Contains(firstPage, "task-1") || !strings.Contains(firstPage, "task-2") {
		t.Fatalf("expected first page tasks to be visible: %s", firstPage)
	}
	if strings.Contains(firstPage, "task-3") || strings.Contains(firstPage, "task-4") {
		t.Fatalf("expected later page tasks to be hidden: %s", firstPage)
	}

	app.taskSelect.cursor = 4
	secondPage := screen.View()
	if !strings.Contains(secondPage, "task-3") || !strings.Contains(secondPage, "task-4") {
		t.Fatalf("expected second page tasks to be visible: %s", secondPage)
	}
	if strings.Contains(secondPage, "task-1") || strings.Contains(secondPage, "task-2") {
		t.Fatalf("expected first page tasks to be hidden: %s", secondPage)
	}
	if !strings.Contains(secondPage, "(4-5/5)") {
		t.Fatalf("expected paging status to be shown: %s", secondPage)
	}
}

func newAppFilterModel() textinput.Model {
	filter := textinput.New()
	filter.Prompt = "検索> "
	filter.Placeholder = "タスク名・プロジェクト名で絞り込み"
	return filter
}

func openTestStore(t *testing.T) *sqlite.Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	return store
}
