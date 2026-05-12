package home

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stoneream/shokushitsu/internal/tui/lib"
)

func TestSummaryDateSelectionEnter(t *testing.T) {
	t.Parallel()

	app := &appState{
		choices: []choice{
			{label: "track", action: ActionTrack},
			{label: "summary", action: ActionSummary},
		},
		menuCursor:   1,
		summaryDates: []string{"2026-03-06", "2026-03-05"},
	}
	model := lib.New(newMenuScreen(app), nil)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(*lib.Model)
	summary, ok := next.Current().(*summaryDateScreen)
	if !ok {
		t.Fatalf("expected summaryDateScreen, got %T", next.Current())
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyEnter})
	final := updated.(*lib.Model)
	if final.Current() != summary {
		t.Fatal("expected to stay on summary screen until program quits")
	}
	if app.result.Action != ActionSummary {
		t.Fatalf("expected selected action summary, got %s", app.result.Action)
	}
	if app.result.SummaryDate != "2026-03-06" {
		t.Fatalf("expected first date selected, got %s", app.result.SummaryDate)
	}
}

func TestSummaryDateSelectionNoDates(t *testing.T) {
	t.Parallel()

	app := &appState{
		choices: []choice{
			{label: "track", action: ActionTrack},
			{label: "summary", action: ActionSummary},
		},
		menuCursor:   1,
		summaryDates: nil,
	}
	model := lib.New(newMenuScreen(app), nil)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(*lib.Model)
	summary, ok := next.Current().(*summaryDateScreen)
	if !ok {
		t.Fatalf("expected summaryDateScreen, got %T", next.Current())
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyEnter})
	final := updated.(*lib.Model)
	current, ok := final.Current().(*summaryDateScreen)
	if !ok {
		t.Fatalf("expected summaryDateScreen, got %T", final.Current())
	}
	if current != summary {
		t.Fatal("expected to remain on summary screen when no dates exist")
	}
	if app.result.Action != "" {
		t.Fatalf("expected no action selected, got %s", app.result.Action)
	}
	if current.notice == "" {
		t.Fatal("expected notice for no selectable dates")
	}
}

func TestSummaryDateViewPaginatesToWindowHeight(t *testing.T) {
	t.Parallel()

	app := &appState{
		windowHeight: 7,
		summaryDates: []string{
			"2026-03-06",
			"2026-03-05",
			"2026-03-04",
			"2026-03-03",
		},
	}
	screen := newSummaryDateScreen(app)

	firstPage := screen.View()
	if !strings.Contains(firstPage, "2026-03-06") || !strings.Contains(firstPage, "2026-03-05") {
		t.Fatalf("expected first page dates to be visible: %s", firstPage)
	}
	if strings.Contains(firstPage, "2026-03-04") || strings.Contains(firstPage, "2026-03-03") {
		t.Fatalf("expected later page dates to be hidden: %s", firstPage)
	}

	screen.cursor = 3
	secondPage := screen.View()
	if !strings.Contains(secondPage, "2026-03-04") || !strings.Contains(secondPage, "2026-03-03") {
		t.Fatalf("expected second page dates to be visible: %s", secondPage)
	}
	if strings.Contains(secondPage, "2026-03-06") || strings.Contains(secondPage, "2026-03-05") {
		t.Fatalf("expected first page dates to be hidden: %s", secondPage)
	}
	if !strings.Contains(secondPage, "(3-4/4)") {
		t.Fatalf("expected paging status to be shown: %s", secondPage)
	}
}
