package picker

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"resumer/internal/session"
)

func TestInitialViewShowsSelectedSessionAndFooter(t *testing.T) {
	m := New(testSessions())

	view := m.View()

	for _, want := range []string{
		"Resume a session",
		"Codex",
		"Project one",
		"Help me ship the picker",
		"enter resume",
		"d details",
		"c copy command",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q:\n%s", want, view)
		}
	}
}

func TestNavigationDoesNotMovePastBounds(t *testing.T) {
	m := New(testSessions())

	m, _ = updateForTest(m, tea.KeyMsg{Type: tea.KeyUp, Runes: []rune{'k'}})
	if m.Cursor != 0 {
		t.Fatalf("Cursor after up at top = %d, want 0", m.Cursor)
	}

	m, _ = updateForTest(m, tea.KeyMsg{Type: tea.KeyDown, Runes: []rune{'j'}})
	m, _ = updateForTest(m, tea.KeyMsg{Type: tea.KeyDown, Runes: []rune{'j'}})
	if m.Cursor != 1 {
		t.Fatalf("Cursor after down past bottom = %d, want 1", m.Cursor)
	}
}

func TestDetailsToggleAndEnterSelectionRecordResumeAction(t *testing.T) {
	m := New(testSessions())

	m, _ = updateForTest(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !m.ShowDetails {
		t.Fatal("ShowDetails = false, want true")
	}
	if view := m.View(); !strings.Contains(view, "ID: codex-1") || !strings.Contains(view, "Source: /tmp/codex.jsonl") || !strings.Contains(view, "Resume command:") {
		t.Fatalf("details view missing expected fields:\n%s", view)
	}

	m, _ = updateForTest(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Action != ActionResume {
		t.Fatalf("Action = %v, want ActionResume", m.Action)
	}
	if m.Selected == nil || m.Selected.ID != "codex-1" {
		t.Fatalf("Selected = %#v, want codex-1", m.Selected)
	}
}

func TestCopyActionAndCancelRecordActions(t *testing.T) {
	m := New(testSessions())

	m, _ = updateForTest(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if m.Action != ActionCopy {
		t.Fatalf("Action = %v, want ActionCopy", m.Action)
	}
	if m.Selected == nil || m.Selected.ID != "codex-1" {
		t.Fatalf("Selected = %#v, want codex-1", m.Selected)
	}

	m = New(testSessions())
	m, _ = updateForTest(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Action != ActionCancel {
		t.Fatalf("Action = %v, want ActionCancel", m.Action)
	}
}

func TestWindowSizeMsgUpdatesDimensions(t *testing.T) {
	m := New(testSessions())

	m, _ = updateForTest(m, tea.WindowSizeMsg{Width: 101, Height: 42})

	if m.Width != 101 || m.Height != 42 {
		t.Fatalf("dimensions = %dx%d, want 101x42", m.Width, m.Height)
	}
}

func TestEmptySessionsViewAndActions(t *testing.T) {
	m := New(nil)

	view := m.View()
	if !strings.Contains(view, "No sessions found") || !strings.Contains(view, "q quit") {
		t.Fatalf("empty View() missing expected text:\n%s", view)
	}

	m, _ = updateForTest(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Action != ActionNone || m.Selected != nil {
		t.Fatalf("after enter on empty: Action=%v Selected=%#v, want none and nil", m.Action, m.Selected)
	}

	m, _ = updateForTest(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if m.Action != ActionNone || m.Selected != nil {
		t.Fatalf("after copy on empty: Action=%v Selected=%#v, want none and nil", m.Action, m.Selected)
	}

	m, _ = updateForTest(m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if m.Action != ActionCancel {
		t.Fatalf("Action = %v, want ActionCancel", m.Action)
	}
}

func testSessions() []session.SessionCard {
	return []session.SessionCard{
		{
			Harness:     session.HarnessCodex,
			ID:          "codex-1",
			Title:       "Project one",
			ProjectPath: "/repo/project-one",
			CreatedAt:   time.Date(2026, 4, 25, 9, 30, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 4, 26, 10, 30, 0, 0, time.UTC),
			FirstPrompt: "Help me ship the picker",
			SourcePath:  "/tmp/codex.jsonl",
		},
		{
			Harness:     session.HarnessClaude,
			ID:          "claude-2",
			Title:       "Project two",
			ProjectPath: "/repo/project-two",
			CreatedAt:   time.Date(2026, 4, 24, 9, 30, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 4, 24, 10, 30, 0, 0, time.UTC),
			FirstPrompt: "Review the ranking code",
			SourcePath:  "/tmp/claude.jsonl",
		},
	}
}
