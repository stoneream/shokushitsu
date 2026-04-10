package track

import (
	"context"
	"path/filepath"
	"testing"
	"time"

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

func openTestStore(t *testing.T) *sqlite.Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	return store
}
