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
		width = 120
	}
	height := m.Height
	if height <= 0 {
		height = 24
	}

	lines := []string{titleStyle.Render(truncate("Resume a session", width))}

	if len(m.Sessions) == 0 {
		lines = append(lines, truncate("No sessions found", width))
		lines = append(lines, mutedStyle.Render(truncate("q quit", width)))
		return joinBounded(lines, height)
	}

	cursor := clampCursor(m.Cursor, len(m.Sessions))
	details := []string(nil)
	if m.ShowDetails {
		details = renderDetails(m.Sessions[cursor], width)
	}
	start, end := visibleRange(cursor, len(m.Sessions), listBudget(height, len(details)))

	for i := start; i < end; i++ {
		lines = append(lines, renderRow(m.Sessions[i], i == cursor, width))
		if i == cursor {
			lines = append(lines, details...)
		}
	}

	lines = append(lines, mutedStyle.Render(truncate("enter resume  d details  c copy command  q quit", width)))
	return joinBounded(lines, height)
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

func renderDetails(card session.SessionCard, width int) []string {
	lines := []string{
		"    Prompt: " + emptyDash(card.FirstPrompt),
		"    ID: " + emptyDash(card.ID),
		"    Source: " + emptyDash(card.SourcePath),
		"    Resume command: " + emptyDash(card.ResumeCommand().Display()),
	}

	for i, line := range lines {
		lines[i] = truncate(line, width)
	}
	return lines
}

func listBudget(height int, detailLines int) int {
	budget := height - 2 - detailLines
	if budget < 1 {
		return 1
	}
	return budget
}

func visibleRange(cursor int, total int, budget int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if budget >= total {
		return 0, total
	}
	start := cursor - budget/2
	if start < 0 {
		start = 0
	}
	if start+budget > total {
		start = total - budget
	}
	return start, start + budget
}

func clampCursor(cursor int, total int) int {
	if cursor < 0 {
		return 0
	}
	if cursor >= total {
		return total - 1
	}
	return cursor
}

func joinBounded(lines []string, height int) string {
	if height > 0 && len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
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
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 3 {
		return truncateCells(s, width)
	}
	return truncateCells(s, width-3) + "..."
}

func truncateCells(s string, width int) string {
	var b strings.Builder
	used := 0
	for _, r := range s {
		next := string(r)
		nextWidth := lipgloss.Width(next)
		if used+nextWidth > width {
			break
		}
		b.WriteString(next)
		used += nextWidth
	}
	return b.String()
}
