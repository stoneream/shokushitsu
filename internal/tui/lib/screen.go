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

func (model *Model) Init() tea.Cmd {
	current := model.Current()
	if current == nil {
		return tea.Quit
	}
	return current.Init(model)
}

func (model *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	current := model.Current()
	if current == nil {
		return model, tea.Quit
	}
	return model, current.Update(msg, model)
}

func (model *Model) View() string {
	current := model.Current()
	if current == nil {
		return ""
	}
	return current.View()
}

func (model *Model) Context() *Context {
	return model.ctx
}

func (model *Model) Current() Screen {
	return model.current
}

func (model *Model) Replace(next Screen) tea.Cmd {
	if next == nil {
		return nil
	}

	model.current = next
	return next.Init(model)
}

func (model *Model) Quit() tea.Cmd {
	return tea.Quit
}
