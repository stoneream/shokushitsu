package lib

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type testMsg string

type testScreen struct {
	name      string
	view      string
	initCount int
	updateFn  func(tea.Msg, Navigator) tea.Cmd
}

func (s *testScreen) Init(Navigator) tea.Cmd {
	s.initCount++
	return nil
}

func (s *testScreen) Update(msg tea.Msg, nav Navigator) tea.Cmd {
	if s.updateFn != nil {
		return s.updateFn(msg, nav)
	}
	return nil
}

func (s *testScreen) View() string {
	if s.view != "" {
		return s.view
	}
	return s.name
}

func TestModelInitUsesRootScreen(t *testing.T) {
	t.Parallel()

	root := &testScreen{name: "root", view: "root view"}
	m := New(root, nil)

	if cmd := m.Init(); cmd != nil {
		t.Fatalf("expected nil init cmd, got %v", cmd)
	}
	if root.initCount != 1 {
		t.Fatalf("expected root init once, got %d", root.initCount)
	}
	if got := m.View(); got != "root view" {
		t.Fatalf("expected delegated view, got %q", got)
	}
}

func TestModelReplaceSwapsCurrentScreen(t *testing.T) {
	t.Parallel()

	replacement := &testScreen{name: "replacement"}
	root := &testScreen{
		name: "root",
		updateFn: func(msg tea.Msg, nav Navigator) tea.Cmd {
			if msg == testMsg("replace") {
				return nav.Replace(replacement)
			}
			return nil
		},
	}

	m := New(root, nil)
	m.Init()

	updated, cmd := m.Update(testMsg("replace"))
	next := updated.(*Model)
	if next.Current() != replacement {
		t.Fatal("expected replacement screen to become current")
	}
	if replacement.initCount != 1 {
		t.Fatalf("expected replacement init once, got %d", replacement.initCount)
	}
	if cmd != nil {
		t.Fatalf("expected nil replace init cmd, got %v", cmd)
	}
}

func TestModelQuitReturnsCmd(t *testing.T) {
	t.Parallel()

	root := &testScreen{name: "root"}
	m := New(root, nil)

	cmd := m.Quit()
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
}

func TestModelReplaceNilKeepsCurrentScreen(t *testing.T) {
	t.Parallel()

	root := &testScreen{name: "root"}
	m := New(root, nil)
	m.Init()

	if cmd := m.Replace(nil); cmd != nil {
		t.Fatalf("expected nil cmd on nil replace, got %v", cmd)
	}
	if m.Current() != root {
		t.Fatal("expected current screen to remain unchanged")
	}
}
