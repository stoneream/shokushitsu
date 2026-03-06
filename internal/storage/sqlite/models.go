package sqlite

import "time"

type Project struct {
	ID         int64
	Name       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ArchivedAt *time.Time
}

type Task struct {
	ID         int64
	ProjectID  int64
	Name       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ArchivedAt *time.Time
}

type Session struct {
	ID        int64
	ProjectID int64
	TaskID    int64
	StartedAt time.Time
	EndedAt   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RecentTask struct {
	TaskID      int64
	ProjectID   int64
	TaskName    string
	ProjectName string
	LastUsedAt  *time.Time
	UseCount    int64
}

type DailySummaryItem struct {
	ProjectName     string
	TaskName        string
	DurationSeconds int64
	SessionCount    int64
}
