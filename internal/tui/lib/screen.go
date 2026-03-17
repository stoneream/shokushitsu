package lib

import tea "github.com/charmbracelet/bubbletea"

type Context struct{}

type Screen interface {
	Init(Navigator) tea.Cmd
	Update(tea.Msg, Navigator) tea.Cmd
	View() string
}

type Navigator interface {
	Context() *Context
	Current() Screen
	Replace(Screen) tea.Cmd
	Quit() tea.Cmd
}

type Model struct {
	ctx     *Context
	current Screen
}

func New(root Screen, ctx *Context) *Model {
	if root == nil {
		panic("root screen が nil です")
	}
	if ctx == nil {
		ctx = &Context{}
	}

	return &Model{
		ctx:     ctx,
		current: root,
	}
}

func (m *Model) Init() tea.Cmd {
	current := m.Current()
	if current == nil {
		return tea.Quit
	}
	return current.Init(m)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	current := m.Current()
	if current == nil {
		return m, tea.Quit
	}
	return m, current.Update(msg, m)
}

func (m *Model) View() string {
	current := m.Current()
	if current == nil {
		return ""
	}
	return current.View()
}

func (m *Model) Context() *Context {
	return m.ctx
}

func (m *Model) Current() Screen {
	return m.current
}

func (m *Model) Replace(next Screen) tea.Cmd {
	if next == nil {
		return nil
	}

	m.current = next
	return next.Init(m)
}

func (m *Model) Quit() tea.Cmd {
	return tea.Quit
}
