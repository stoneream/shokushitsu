package home

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSummaryDateSelectionEnter(t *testing.T) {
	t.Parallel()

	m := &model{
		choices: []choice{
			{label: "track", action: ActionTrack},
			{label: "summary", action: ActionSummary},
		},
		cursor:       1,
		state:        stateMenu,
		summaryDates: []string{"2026-03-06", "2026-03-05"},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(*model)
	if next.state != stateSummaryDateSelect {
		t.Fatalf("expected stateSummaryDateSelect, got %v", next.state)
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyEnter})
	final := updated.(*model)
	if final.selected != ActionSummary {
		t.Fatalf("expected selected action summary, got %s", final.selected)
	}
	if final.summaryDate != "2026-03-06" {
		t.Fatalf("expected first date selected, got %s", final.summaryDate)
	}
}

func TestSummaryDateSelectionNoDates(t *testing.T) {
	t.Parallel()

	m := &model{
		choices: []choice{
			{label: "track", action: ActionTrack},
			{label: "summary", action: ActionSummary},
		},
		cursor:       1,
		state:        stateMenu,
		summaryDates: nil,
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(*model)
	if next.state != stateSummaryDateSelect {
		t.Fatalf("expected stateSummaryDateSelect, got %v", next.state)
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyEnter})
	final := updated.(*model)
	if final.selected != "" {
		t.Fatalf("expected no action selected, got %s", final.selected)
	}
	if final.notice == "" {
		t.Fatal("expected notice for no selectable dates")
	}
}
