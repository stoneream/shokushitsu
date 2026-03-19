package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrSessionNotFound     = errors.New("session not found")
	ErrSessionAlreadyEnded = errors.New("session already ended")
)

func (s *Store) GetOrCreateProject(ctx context.Context, name string) (Project, error) {
	project, err := s.getProjectByName(ctx, name)
	if err == nil {
		return project, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Project{}, err
	}

	now := time.Now().UTC()
	const insert = `
INSERT INTO projects(name, created_at, updated_at)
VALUES(?, ?, ?)
ON CONFLICT(name) DO UPDATE SET updated_at=excluded.updated_at
RETURNING id, name, created_at, updated_at, archived_at;
`

	var row Project
	var createdAt, updatedAt int64
	var archivedAt sql.NullInt64
	if err := s.db.QueryRowContext(ctx, insert, name, now.Unix(), now.Unix()).Scan(
		&row.ID,
		&row.Name,
		&createdAt,
		&updatedAt,
		&archivedAt,
	); err != nil {
		return Project{}, fmt.Errorf("create project %q: %w", name, err)
	}

	row.CreatedAt = fromUnix(createdAt)
	row.UpdatedAt = fromUnix(updatedAt)
	row.ArchivedAt = nullableTime(archivedAt)
	return row, nil
}

func (s *Store) GetOrCreateTask(ctx context.Context, projectID int64, name string) (Task, error) {
	task, err := s.getTaskByProjectAndName(ctx, projectID, name)
	if err == nil {
		return task, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Task{}, err
	}

	now := time.Now().UTC()
	const insert = `
INSERT INTO tasks(project_id, name, created_at, updated_at)
VALUES(?, ?, ?, ?)
ON CONFLICT(project_id, name) DO UPDATE SET updated_at=excluded.updated_at
RETURNING id, project_id, name, created_at, updated_at, archived_at;
`

	var row Task
	var createdAt, updatedAt int64
	var archivedAt sql.NullInt64
	if err := s.db.QueryRowContext(ctx, insert, projectID, name, now.Unix(), now.Unix()).Scan(
		&row.ID,
		&row.ProjectID,
		&row.Name,
		&createdAt,
		&updatedAt,
		&archivedAt,
	); err != nil {
		return Task{}, fmt.Errorf("create task %q: %w", name, err)
	}

	row.CreatedAt = fromUnix(createdAt)
	row.UpdatedAt = fromUnix(updatedAt)
	row.ArchivedAt = nullableTime(archivedAt)
	return row, nil
}

func (s *Store) StartSession(ctx context.Context, projectID, taskID int64, startedAt time.Time) (Session, error) {
	startedAt = startedAt.UTC()
	now := time.Now().UTC()

	const insertSessionQuery = `
INSERT INTO sessions(project_id, task_id, started_at, created_at, updated_at)
VALUES(?, ?, ?, ?, ?)
RETURNING id, project_id, task_id, started_at, ended_at, created_at, updated_at;
`

	var row Session
	var startedAtUnix, createdAtUnix, updatedAtUnix int64
	var endedAt sql.NullInt64
	if err := s.db.QueryRowContext(ctx, insertSessionQuery, projectID, taskID, startedAt.Unix(), now.Unix(), now.Unix()).Scan(
		&row.ID,
		&row.ProjectID,
		&row.TaskID,
		&startedAtUnix,
		&endedAt,
		&createdAtUnix,
		&updatedAtUnix,
	); err != nil {
		return Session{}, fmt.Errorf("start session: %w", err)
	}

	row.StartedAt = fromUnix(startedAtUnix)
	row.EndedAt = nullableTime(endedAt)
	row.CreatedAt = fromUnix(createdAtUnix)
	row.UpdatedAt = fromUnix(updatedAtUnix)
	return row, nil
}

func (s *Store) EndSession(ctx context.Context, sessionID int64, endedAt time.Time) error {
	endedAt = endedAt.UTC()
	now := time.Now().UTC()

	const endSessionQuery = `
UPDATE sessions
SET ended_at = ?, updated_at = ?
WHERE id = ? AND ended_at IS NULL;
`

	result, err := s.db.ExecContext(ctx, endSessionQuery, endedAt.Unix(), now.Unix(), sessionID)
	if err != nil {
		return fmt.Errorf("end session %d: %w", sessionID, err)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for end session %d: %w", sessionID, err)
	}
	if affectedRows > 0 {
		return nil
	}

	const check = `SELECT ended_at FROM sessions WHERE id = ?`
	var existingEndedAt sql.NullInt64
	if err := s.db.QueryRowContext(ctx, check, sessionID).Scan(&existingEndedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSessionNotFound
		}

		return fmt.Errorf("check session %d: %w", sessionID, err)
	}

	if existingEndedAt.Valid {
		return ErrSessionAlreadyEnded
	}

	return nil
}

func (s *Store) GetActiveSessions(ctx context.Context) ([]Session, error) {
	const activeSessionsQuery = `
SELECT id, project_id, task_id, started_at, ended_at, created_at, updated_at
FROM sessions
WHERE ended_at IS NULL
ORDER BY started_at ASC, id ASC;
`

	rows, err := s.db.QueryContext(ctx, activeSessionsQuery)
	if err != nil {
		return nil, fmt.Errorf("query active sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active sessions: %w", err)
	}

	return sessions, nil
}

func (s *Store) ListSessionsByRange(ctx context.Context, from, to time.Time) ([]Session, error) {
	const sessionsByRangeQuery = `
SELECT id, project_id, task_id, started_at, ended_at, created_at, updated_at
FROM sessions
WHERE started_at >= ? AND started_at < ?
ORDER BY started_at ASC, id ASC;
`

	rows, err := s.db.QueryContext(ctx, sessionsByRangeQuery, from.UTC().Unix(), to.UTC().Unix())
	if err != nil {
		return nil, fmt.Errorf("query sessions by range: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions by range: %w", err)
	}

	return sessions, nil
}

func (s *Store) ListProjects(ctx context.Context) ([]Project, error) {
	const listProjectsQuery = `
SELECT id, name, created_at, updated_at, archived_at
FROM projects
WHERE archived_at IS NULL
ORDER BY name ASC;
`

	rows, err := s.db.QueryContext(ctx, listProjectsQuery)
	if err != nil {
		return nil, fmt.Errorf("query projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var row Project
		var createdAt, updatedAt int64
		var archivedAt sql.NullInt64
		if err := rows.Scan(
			&row.ID,
			&row.Name,
			&createdAt,
			&updatedAt,
			&archivedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project row: %w", err)
		}

		row.CreatedAt = fromUnix(createdAt)
		row.UpdatedAt = fromUnix(updatedAt)
		row.ArchivedAt = nullableTime(archivedAt)
		projects = append(projects, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}

	return projects, nil
}

func (s *Store) ListRecentTasks(ctx context.Context) ([]RecentTask, error) {
	const recentTasksQuery = `
SELECT
	t.id,
	t.project_id,
	t.name,
	p.name,
	MAX(s.ended_at) AS last_used_at,
	COUNT(s.id) AS use_count
FROM tasks AS t
JOIN projects AS p
	ON p.id = t.project_id
LEFT JOIN sessions AS s
	ON s.task_id = t.id
	AND s.ended_at IS NOT NULL
WHERE p.archived_at IS NULL
	AND t.archived_at IS NULL
GROUP BY
	t.id,
	t.project_id,
	t.name,
	p.name
ORDER BY
	(last_used_at IS NULL) ASC,
	last_used_at DESC,
	p.name ASC,
	t.name ASC;
`

	rows, err := s.db.QueryContext(ctx, recentTasksQuery)
	if err != nil {
		return nil, fmt.Errorf("query recent tasks: %w", err)
	}
	defer rows.Close()

	var tasks []RecentTask
	for rows.Next() {
		var row RecentTask
		var lastUsedAt sql.NullInt64
		if err := rows.Scan(
			&row.TaskID,
			&row.ProjectID,
			&row.TaskName,
			&row.ProjectName,
			&lastUsedAt,
			&row.UseCount,
		); err != nil {
			return nil, fmt.Errorf("scan recent task row: %w", err)
		}

		row.LastUsedAt = nullableTime(lastUsedAt)
		tasks = append(tasks, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent tasks: %w", err)
	}

	return tasks, nil
}

func (s *Store) ListDailySummaryItemsByStartedRange(ctx context.Context, from, to time.Time) ([]DailySummaryItem, error) {
	const dailySummaryQuery = `
SELECT
	p.name,
	t.name,
	SUM(s.ended_at - s.started_at) AS duration_seconds,
	COUNT(s.id) AS session_count
FROM sessions AS s
JOIN projects AS p
	ON p.id = s.project_id
JOIN tasks AS t
	ON t.id = s.task_id
WHERE s.started_at >= ?
	AND s.started_at < ?
	AND s.ended_at IS NOT NULL
GROUP BY
	p.name,
	t.name
ORDER BY
	p.name ASC,
	t.name ASC;
`

	rows, err := s.db.QueryContext(ctx, dailySummaryQuery, from.UTC().Unix(), to.UTC().Unix())
	if err != nil {
		return nil, fmt.Errorf("query daily summary items: %w", err)
	}
	defer rows.Close()

	var items []DailySummaryItem
	for rows.Next() {
		var row DailySummaryItem
		if err := rows.Scan(
			&row.ProjectName,
			&row.TaskName,
			&row.DurationSeconds,
			&row.SessionCount,
		); err != nil {
			return nil, fmt.Errorf("scan daily summary item row: %w", err)
		}
		items = append(items, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate daily summary item rows: %w", err)
	}

	return items, nil
}

func (s *Store) ListSessionStartedDates(ctx context.Context, loc *time.Location) ([]time.Time, error) {
	if loc == nil {
		loc = time.Local
	}

	const sessionStartedDatesQuery = `
SELECT started_at
FROM sessions
ORDER BY started_at DESC, id DESC;
`

	rows, err := s.db.QueryContext(ctx, sessionStartedDatesQuery)
	if err != nil {
		return nil, fmt.Errorf("query session started dates: %w", err)
	}
	defer rows.Close()

	seen := make(map[string]struct{})
	dates := make([]time.Time, 0)

	for rows.Next() {
		var startedAtUnix int64
		if err := rows.Scan(&startedAtUnix); err != nil {
			return nil, fmt.Errorf("scan session started_at row: %w", err)
		}

		startedAt := time.Unix(startedAtUnix, 0).In(loc)
		date := time.Date(startedAt.Year(), startedAt.Month(), startedAt.Day(), 0, 0, 0, 0, loc)
		key := date.Format("2006-01-02")
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
		dates = append(dates, date)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session started_at rows: %w", err)
	}

	return dates, nil
}

func (s *Store) CountUnfinishedSessionsByStartedRange(ctx context.Context, from, to time.Time) (int64, error) {
	const unfinishedSessionsQuery = `
SELECT COUNT(1)
FROM sessions
WHERE started_at >= ?
	AND started_at < ?
	AND ended_at IS NULL;
`

	var count int64
	if err := s.db.QueryRowContext(ctx, unfinishedSessionsQuery, from.UTC().Unix(), to.UTC().Unix()).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unfinished sessions by range: %w", err)
	}

	return count, nil
}

func scanSession(rows *sql.Rows) (Session, error) {
	var row Session
	var startedAtUnix, createdAtUnix, updatedAtUnix int64
	var endedAt sql.NullInt64

	if err := rows.Scan(
		&row.ID,
		&row.ProjectID,
		&row.TaskID,
		&startedAtUnix,
		&endedAt,
		&createdAtUnix,
		&updatedAtUnix,
	); err != nil {
		return Session{}, fmt.Errorf("scan session row: %w", err)
	}

	row.StartedAt = fromUnix(startedAtUnix)
	row.EndedAt = nullableTime(endedAt)
	row.CreatedAt = fromUnix(createdAtUnix)
	row.UpdatedAt = fromUnix(updatedAtUnix)
	return row, nil
}

func (s *Store) getProjectByName(ctx context.Context, name string) (Project, error) {
	const projectByNameQuery = `
SELECT id, name, created_at, updated_at, archived_at
FROM projects
WHERE name = ?;
`

	var row Project
	var createdAt, updatedAt int64
	var archivedAt sql.NullInt64
	if err := s.db.QueryRowContext(ctx, projectByNameQuery, name).Scan(
		&row.ID,
		&row.Name,
		&createdAt,
		&updatedAt,
		&archivedAt,
	); err != nil {
		return Project{}, err
	}

	row.CreatedAt = fromUnix(createdAt)
	row.UpdatedAt = fromUnix(updatedAt)
	row.ArchivedAt = nullableTime(archivedAt)
	return row, nil
}

func (s *Store) getTaskByProjectAndName(ctx context.Context, projectID int64, name string) (Task, error) {
	const taskByProjectAndNameQuery = `
SELECT id, project_id, name, created_at, updated_at, archived_at
FROM tasks
WHERE project_id = ? AND name = ?;
`

	var row Task
	var createdAt, updatedAt int64
	var archivedAt sql.NullInt64
	if err := s.db.QueryRowContext(ctx, taskByProjectAndNameQuery, projectID, name).Scan(
		&row.ID,
		&row.ProjectID,
		&row.Name,
		&createdAt,
		&updatedAt,
		&archivedAt,
	); err != nil {
		return Task{}, err
	}

	row.CreatedAt = fromUnix(createdAt)
	row.UpdatedAt = fromUnix(updatedAt)
	row.ArchivedAt = nullableTime(archivedAt)
	return row, nil
}

func fromUnix(unixSeconds int64) time.Time {
	return time.Unix(unixSeconds, 0).UTC()
}

func nullableTime(nullableUnixSeconds sql.NullInt64) *time.Time {
	if !nullableUnixSeconds.Valid {
		return nil
	}

	timestamp := fromUnix(nullableUnixSeconds.Int64)
	return &timestamp
}
