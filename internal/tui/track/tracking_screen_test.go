package track

import (
	"testing"
	"time"

	"github.com/stoneream/shokushitsu/internal/storage/sqlite"
)

func TestTrackingScreenTickShowsContinuePromptAtTwentyFiveMinutes(t *testing.T) {
	t.Parallel()

	startedAt := time.Unix(1_700_000_000, 0).UTC()
	app := &appState{
		currentSession: &sqlite.Session{StartedAt: startedAt},
		tracking: trackingState{
			now:     startedAt,
			project: "alpha",
			task:    "task-a",
		},
	}

	screen := newTrackingScreen(app)
	screen.Update(tickMsg{at: startedAt.Add(25 * time.Minute)}, nil)

	if app.notice != "25分経過しました。cで継続、eで終了します。" {
		t.Fatalf("unexpected notice: %q", app.notice)
	}
	if !app.tracking.continueCheckActive {
		t.Fatal("expected continue check to become active")
	}
}

func TestTrackingScreenTickShowsBreakPromptAtFiftyMinutes(t *testing.T) {
	t.Parallel()

	startedAt := time.Unix(1_700_000_000, 0).UTC()
	app := &appState{
		currentSession: &sqlite.Session{StartedAt: startedAt},
		tracking: trackingState{
			now:     startedAt,
			project: "alpha",
			task:    "task-a",
		},
	}

	screen := newTrackingScreen(app)
	screen.Update(tickMsg{at: startedAt.Add(50 * time.Minute)}, nil)

	if app.notice != "休憩しませんか？ cで継続、eで終了します。" {
		t.Fatalf("unexpected notice: %q", app.notice)
	}
	if !app.tracking.continueCheckActive {
		t.Fatal("expected continue check to become active")
	}
}
