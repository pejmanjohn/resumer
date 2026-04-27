package picker

import (
	tea "github.com/charmbracelet/bubbletea"

	"resumer/internal/session"
)

type Action int

const (
	ActionNone Action = iota
	ActionResume
	ActionCopy
	ActionCancel
)

type Model struct {
	Sessions    []session.SessionCard
	Cursor      int
	Width       int
	Height      int
	ShowDetails bool
	Selected    *session.SessionCard
	Action      Action
}

func New(sessions []session.SessionCard) Model {
	return Model{
		Sessions: sessions,
		Width:    120,
		Height:   24,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := updateForTest(m, msg)
	return next, cmd
}

func updateForTest(m Model, msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	case tea.KeyMsg:
		return updateKey(m, msg)
	}

	return m, nil
}

func updateKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "down", "j":
		if m.Cursor < len(m.Sessions)-1 {
			m.Cursor++
		}
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "d":
		m.ShowDetails = !m.ShowDetails
	case "enter":
		if selected, ok := m.current(); ok {
			m.Action = ActionResume
			m.Selected = &selected
			return m, tea.Quit
		}
	case "c":
		if selected, ok := m.current(); ok {
			m.Action = ActionCopy
			m.Selected = &selected
			return m, tea.Quit
		}
	case "q", "esc", "ctrl+c":
		m.Action = ActionCancel
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) current() (session.SessionCard, bool) {
	if m.Cursor < 0 || m.Cursor >= len(m.Sessions) {
		return session.SessionCard{}, false
	}
	return m.Sessions[m.Cursor], true
}
