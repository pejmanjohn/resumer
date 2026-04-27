package picker

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"resumer/internal/session"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Bold(true)
	mutedStyle    = lipgloss.NewStyle().Faint(true)
)

func (m Model) View() string {
	width := m.Width
	if width <= 0 {
		width = 80
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Resume a session"))
	b.WriteString("\n\n")

	if len(m.Sessions) == 0 {
		b.WriteString("No sessions found\n\n")
		b.WriteString(mutedStyle.Render("q quit"))
		return b.String()
	}

	for i, card := range m.Sessions {
		b.WriteString(renderRow(card, i == m.Cursor, width))
		b.WriteByte('\n')
		if i == m.Cursor && m.ShowDetails {
			b.WriteString(renderDetails(card, width))
		}
	}

	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("enter resume  d details  c copy command  q quit"))
	return b.String()
}

func renderRow(card session.SessionCard, selected bool, width int) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}

	line := fmt.Sprintf(
		"%s%s  %s  %s  %s%s",
		prefix,
		titleCaseHarness(card.Harness),
		formatTime(card.SortTime()),
		emptyDash(card.ProjectPath),
		card.DisplayTitle(),
		promptSuffix(card.FirstPrompt),
	)
	line = truncate(line, width)
	if selected {
		return selectedStyle.Render(line)
	}
	return line
}

func renderDetails(card session.SessionCard, width int) string {
	lines := []string{
		"    Prompt: " + emptyDash(card.FirstPrompt),
		"    ID: " + emptyDash(card.ID),
		"    Source: " + emptyDash(card.SourcePath),
		"    Resume command: " + emptyDash(card.ResumeCommand().Display()),
	}

	for i, line := range lines {
		lines[i] = truncate(line, width)
	}
	return strings.Join(lines, "\n") + "\n"
}

func titleCaseHarness(h session.Harness) string {
	switch h {
	case session.HarnessCodex:
		return "Codex"
	case session.HarnessClaude:
		return "Claude"
	default:
		raw := strings.TrimSpace(string(h))
		if raw == "" {
			return "Unknown"
		}
		return strings.ToUpper(raw[:1]) + strings.ToLower(raw[1:])
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "unknown time"
	}
	return t.UTC().Format("2006-01-02 15:04 UTC")
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return strings.TrimSpace(s)
}

func promptSuffix(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ""
	}
	return "  " + prompt
}

func truncate(s string, width int) string {
	if width <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}
