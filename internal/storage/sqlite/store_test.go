package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenAppliesPragmasAndMigrations(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer func() {
		_ = store.Close()
	}()

	ctx := context.Background()

	assertTableExists(t, store, "schema_migrations")
	assertTableExists(t, store, "projects")
	assertTableExists(t, store, "tasks")
	assertTableExists(t, store, "sessions")

	var foreignKeys int
	if err := store.db.QueryRowContext(ctx, "PRAGMA foreign_keys;").Scan(&foreignKeys); err != nil {
		t.Fatalf("read pragma foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("unexpected foreign_keys pragma: got %d", foreignKeys)
	}

	var journalMode string
	if err := store.db.QueryRowContext(ctx, "PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		t.Fatalf("read pragma journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("unexpected journal_mode pragma: got %s", journalMode)
	}
}

func TestRepositoryFlow(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer func() {
		_ = store.Close()
	}()

	ctx := context.Background()

	project, err := store.GetOrCreateProject(ctx, "project-a")
	if err != nil {
		t.Fatalf("GetOrCreateProject failed: %v", err)
	}

	project2, err := store.GetOrCreateProject(ctx, "project-a")
	if err != nil {
		t.Fatalf("GetOrCreateProject second call failed: %v", err)
	}
	if project.ID != project2.ID {
		t.Fatalf("project ID mismatch: %d != %d", project.ID, project2.ID)
	}

	task, err := store.GetOrCreateTask(ctx, project.ID, "task-a")
	if err != nil {
		t.Fatalf("GetOrCreateTask failed: %v", err)
	}

	task2, err := store.GetOrCreateTask(ctx, project.ID, "task-a")
	if err != nil {
		t.Fatalf("GetOrCreateTask second call failed: %v", err)
	}
	if task.ID != task2.ID {
		t.Fatalf("task ID mismatch: %d != %d", task.ID, task2.ID)
	}

	start1 := time.Unix(1_700_000_000, 0).UTC()
	session1, err := store.StartSession(ctx, project.ID, task.ID, start1)
	if err != nil {
		t.Fatalf("StartSession #1 failed: %v", err)
	}

	start2 := time.Unix(1_700_000_600, 0).UTC()
	session2, err := store.StartSession(ctx, project.ID, task.ID, start2)
	if err != nil {
		t.Fatalf("StartSession #2 failed: %v", err)
	}

	active, err := store.GetActiveSessions(ctx)
	if err != nil {
		t.Fatalf("GetActiveSessions failed: %v", err)
	}
	if len(active) != 2 {
		t.Fatalf("unexpected active session count: got %d", len(active))
	}

	end1 := time.Unix(1_700_000_900, 0).UTC()
	if err := store.EndSession(ctx, session1.ID, end1); err != nil {
		t.Fatalf("EndSession failed: %v", err)
	}

	active, err = store.GetActiveSessions(ctx)
	if err != nil {
		t.Fatalf("GetActiveSessions after end failed: %v", err)
	}
	if len(active) != 1 || active[0].ID != session2.ID {
		t.Fatalf("unexpected active sessions after end: %+v", active)
	}

	items, err := store.ListSessionsByRange(
		ctx,
		time.Unix(1_699_999_000, 0).UTC(),
		time.Unix(1_700_010_000, 0).UTC(),
	)
	if err != nil {
		t.Fatalf("ListSessionsByRange failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected range result count: got %d", len(items))
	}
}

func TestEndSessionErrors(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer func() {
		_ = store.Close()
	}()

	ctx := context.Background()

	project, err := store.GetOrCreateProject(ctx, "project-a")
	if err != nil {
		t.Fatalf("GetOrCreateProject failed: %v", err)
	}
	task, err := store.GetOrCreateTask(ctx, project.ID, "task-a")
	if err != nil {
		t.Fatalf("GetOrCreateTask failed: %v", err)
	}
	session, err := store.StartSession(ctx, project.ID, task.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	if err := store.EndSession(ctx, session.ID, time.Now().UTC()); err != nil {
		t.Fatalf("EndSession first call failed: %v", err)
	}

	err = store.EndSession(ctx, session.ID, time.Now().UTC())
	if !errors.Is(err, ErrSessionAlreadyEnded) {
		t.Fatalf("expected ErrSessionAlreadyEnded, got %v", err)
	}

	err = store.EndSession(ctx, 999999, time.Now().UTC())
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestListProjectsAndRecentTasks(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer func() {
		_ = store.Close()
	}()

	ctx := context.Background()

	projectA, err := store.GetOrCreateProject(ctx, "alpha")
	if err != nil {
		t.Fatalf("GetOrCreateProject alpha failed: %v", err)
	}
	projectB, err := store.GetOrCreateProject(ctx, "beta")
	if err != nil {
		t.Fatalf("GetOrCreateProject beta failed: %v", err)
	}

	taskA, err := store.GetOrCreateTask(ctx, projectA.ID, "task-a")
	if err != nil {
		t.Fatalf("GetOrCreateTask task-a failed: %v", err)
	}
	_, err = store.GetOrCreateTask(ctx, projectB.ID, "task-b")
	if err != nil {
		t.Fatalf("GetOrCreateTask task-b failed: %v", err)
	}

	session, err := store.StartSession(ctx, projectA.ID, taskA.ID, time.Unix(1_700_001_000, 0).UTC())
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}
	if err := store.EndSession(ctx, session.ID, time.Unix(1_700_001_600, 0).UTC()); err != nil {
		t.Fatalf("EndSession failed: %v", err)
	}

	projects, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("unexpected project count: got %d", len(projects))
	}
	if projects[0].Name != "alpha" || projects[1].Name != "beta" {
		t.Fatalf("unexpected project order: %+v", projects)
	}

	tasks, err := store.ListRecentTasks(ctx)
	if err != nil {
		t.Fatalf("ListRecentTasks failed: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("unexpected task count: got %d", len(tasks))
	}
	if tasks[0].TaskName != "task-a" {
		t.Fatalf("unexpected first task: %+v", tasks[0])
	}
	if tasks[0].UseCount != 1 || tasks[0].LastUsedAt == nil {
		t.Fatalf("unexpected first task usage: %+v", tasks[0])
	}
	if tasks[1].TaskName != "task-b" {
		t.Fatalf("unexpected second task: %+v", tasks[1])
	}
	if tasks[1].UseCount != 0 || tasks[1].LastUsedAt != nil {
		t.Fatalf("unexpected second task usage: %+v", tasks[1])
	}
}

func TestDailySummaryByStartedRange(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer func() {
		_ = store.Close()
	}()

	ctx := context.Background()

	project, err := store.GetOrCreateProject(ctx, "alpha")
	if err != nil {
		t.Fatalf("GetOrCreateProject failed: %v", err)
	}
	taskA, err := store.GetOrCreateTask(ctx, project.ID, "task-a")
	if err != nil {
		t.Fatalf("GetOrCreateTask task-a failed: %v", err)
	}
	taskB, err := store.GetOrCreateTask(ctx, project.ID, "task-b")
	if err != nil {
		t.Fatalf("GetOrCreateTask task-b failed: %v", err)
	}

	loc := time.FixedZone("JST", 9*60*60)
	from := time.Date(2026, 3, 6, 0, 0, 0, 0, loc)
	to := from.AddDate(0, 0, 1)

	sessionA1, err := store.StartSession(ctx, project.ID, taskA.ID, time.Date(2026, 3, 6, 9, 0, 0, 0, loc).UTC())
	if err != nil {
		t.Fatalf("StartSession task-a #1 failed: %v", err)
	}
	if err := store.EndSession(ctx, sessionA1.ID, time.Date(2026, 3, 6, 10, 30, 0, 0, loc).UTC()); err != nil {
		t.Fatalf("EndSession task-a #1 failed: %v", err)
	}

	sessionA2, err := store.StartSession(ctx, project.ID, taskA.ID, time.Date(2026, 3, 6, 11, 0, 0, 0, loc).UTC())
	if err != nil {
		t.Fatalf("StartSession task-a #2 failed: %v", err)
	}
	if err := store.EndSession(ctx, sessionA2.ID, time.Date(2026, 3, 6, 12, 0, 0, 0, loc).UTC()); err != nil {
		t.Fatalf("EndSession task-a #2 failed: %v", err)
	}

	_, err = store.StartSession(ctx, project.ID, taskA.ID, time.Date(2026, 3, 6, 13, 0, 0, 0, loc).UTC())
	if err != nil {
		t.Fatalf("StartSession task-a active failed: %v", err)
	}

	sessionB1, err := store.StartSession(ctx, project.ID, taskB.ID, time.Date(2026, 3, 6, 15, 0, 0, 0, loc).UTC())
	if err != nil {
		t.Fatalf("StartSession task-b #1 failed: %v", err)
	}
	if err := store.EndSession(ctx, sessionB1.ID, time.Date(2026, 3, 6, 15, 20, 0, 0, loc).UTC()); err != nil {
		t.Fatalf("EndSession task-b #1 failed: %v", err)
	}

	sessionOutside, err := store.StartSession(ctx, project.ID, taskB.ID, time.Date(2026, 3, 7, 9, 0, 0, 0, loc).UTC())
	if err != nil {
		t.Fatalf("StartSession outside range failed: %v", err)
	}
	if err := store.EndSession(ctx, sessionOutside.ID, time.Date(2026, 3, 7, 10, 0, 0, 0, loc).UTC()); err != nil {
		t.Fatalf("EndSession outside range failed: %v", err)
	}

	items, err := store.ListDailySummaryItemsByStartedRange(ctx, from, to)
	if err != nil {
		t.Fatalf("ListDailySummaryItemsByStartedRange failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected summary item count: got %d", len(items))
	}

	if items[0].ProjectName != "alpha" || items[0].TaskName != "task-a" {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if items[0].SessionCount != 2 || items[0].DurationSeconds != 9000 {
		t.Fatalf("unexpected first item aggregate: %+v", items[0])
	}

	if items[1].ProjectName != "alpha" || items[1].TaskName != "task-b" {
		t.Fatalf("unexpected second item: %+v", items[1])
	}
	if items[1].SessionCount != 1 || items[1].DurationSeconds != 1200 {
		t.Fatalf("unexpected second item aggregate: %+v", items[1])
	}

	unfinished, err := store.CountUnfinishedSessionsByStartedRange(ctx, from, to)
	if err != nil {
		t.Fatalf("CountUnfinishedSessionsByStartedRange failed: %v", err)
	}
	if unfinished != 1 {
		t.Fatalf("unexpected unfinished count: got %d", unfinished)
	}
}

func TestListSessionStartedDates(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer func() {
		_ = store.Close()
	}()

	ctx := context.Background()
	project, err := store.GetOrCreateProject(ctx, "alpha")
	if err != nil {
		t.Fatalf("GetOrCreateProject failed: %v", err)
	}
	task, err := store.GetOrCreateTask(ctx, project.ID, "task-a")
	if err != nil {
		t.Fatalf("GetOrCreateTask failed: %v", err)
	}

	loc := time.FixedZone("JST", 9*60*60)
	starts := []time.Time{
		time.Date(2026, 3, 6, 10, 0, 0, 0, loc),
		time.Date(2026, 3, 5, 9, 0, 0, 0, loc),
		time.Date(2026, 3, 6, 14, 0, 0, 0, loc),
		time.Date(2026, 3, 4, 23, 59, 0, 0, loc),
	}
	for _, startedAt := range starts {
		session, err := store.StartSession(ctx, project.ID, task.ID, startedAt.UTC())
		if err != nil {
			t.Fatalf("StartSession failed: %v", err)
		}
		if err := store.EndSession(ctx, session.ID, startedAt.Add(30*time.Minute).UTC()); err != nil {
			t.Fatalf("EndSession failed: %v", err)
		}
	}

	dates, err := store.ListSessionStartedDates(ctx, loc)
	if err != nil {
		t.Fatalf("ListSessionStartedDates failed: %v", err)
	}
	if len(dates) != 3 {
		t.Fatalf("unexpected date count: got %d", len(dates))
	}

	got := []string{dates[0].Format("2006-01-02"), dates[1].Format("2006-01-02"), dates[2].Format("2006-01-02")}
	want := []string{"2026-03-06", "2026-03-05", "2026-03-04"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected date order at %d: got %s, want %s", i, got[i], want[i])
		}
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	return store
}

func assertTableExists(t *testing.T, store *Store, name string) {
	t.Helper()

	ctx := context.Background()
	const q = `SELECT 1 FROM sqlite_master WHERE type='table' AND name = ? LIMIT 1`

	var value int
	if err := store.db.QueryRowContext(ctx, q, name).Scan(&value); err != nil {
		t.Fatalf("table %s not found: %v", name, err)
	}
}
