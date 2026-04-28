package picker

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/pejmanjohn/resumer/internal/session"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Bold(true)
	mutedStyle    = lipgloss.NewStyle().Faint(true)
)

const compactLayoutWidth = 80

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

	compact := isCompactLayout(width)
	cursor := clampCursor(m.Cursor, len(m.Sessions))
	start, end := visibleRange(cursor, len(m.Sessions), rowBudget(height, 0, compact))
	showHarness := mixedHarnesses(m.Sessions[start:end])
	details := []string(nil)
	if m.ShowDetails {
		details = fitDetails(renderDetails(m.Sessions[cursor], width), height)
		start, end = visibleRange(cursor, len(m.Sessions), rowBudget(height, len(details), compact))
		showHarness = mixedHarnesses(m.Sessions[start:end])
	}

	if compact {
		for i := start; i < end; i++ {
			lines = append(lines, renderCompactRow(m.Sessions[i], i == cursor, showHarness, width)...)
			if i == cursor {
				lines = append(lines, details...)
			}
		}

		lines = append(lines, mutedStyle.Render(truncate(compactFooter(width), width)))
		return joinBounded(lines, height)
	}

	cols := rowLayout(width, showHarness, m.Sessions[start:end])

	lines = append(lines, renderHeader(cols, showHarness))
	for i := start; i < end; i++ {
		lines = append(lines, renderRow(m.Sessions[i], i == cursor, cols, showHarness))
		if i == cursor {
			lines = append(lines, details...)
		}
	}

	lines = append(lines, mutedStyle.Render(truncate("enter resume  d details  c copy command  q quit", width)))
	return joinBounded(lines, height)
}

func isCompactLayout(width int) bool {
	return width < compactLayoutWidth
}

func renderHeader(cols rowColumns, showHarness bool) string {
	return mutedStyle.Render(joinRow(rowCells{
		prefix:  "  ",
		title:   padCell("Title", cols.title),
		project: padCell("Project", cols.project),
		age:     padCell("Age", cols.age),
		tool:    padCell("Tool", cols.tool),
	}, showHarness))
}

func renderRow(card session.SessionCard, selected bool, cols rowColumns, showHarness bool) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}

	title := padCell(oneLine(card.DisplayTitle()), cols.title)
	if selected {
		title = selectedStyle.Render(title)
	}
	return joinRow(rowCells{
		prefix:  prefix,
		title:   title,
		project: mutedStyle.Render(padCell(projectLabel(card.ProjectPath), cols.project)),
		age:     mutedStyle.Render(padCell(formatTime(card.SortTime()), cols.age)),
		tool:    mutedStyle.Render(padCell(titleCaseHarness(card.Harness), cols.tool)),
	}, showHarness)
}

func renderCompactRow(card session.SessionCard, selected bool, showHarness bool, width int) []string {
	prefix := "  "
	if selected {
		prefix = "> "
	}

	titleWidth := width - lipgloss.Width(prefix)
	if titleWidth < 1 {
		titleWidth = 1
	}
	title := truncate(oneLine(card.DisplayTitle()), titleWidth)
	if selected {
		title = selectedStyle.Render(title)
	}

	metaWidth := width - 2
	if metaWidth < 1 {
		metaWidth = 1
	}
	meta := mutedStyle.Render(truncate(compactMetadata(card, showHarness), metaWidth))
	return []string{
		truncate(prefix+title, width),
		truncate("  "+meta, width),
	}
}

func compactMetadata(card session.SessionCard, showHarness bool) string {
	parts := []string{
		projectLabel(card.ProjectPath),
		formatTime(card.SortTime()),
	}
	if showHarness {
		parts = append(parts, titleCaseHarness(card.Harness))
	}
	return strings.Join(parts, "  ")
}

func compactFooter(width int) string {
	if width < 44 {
		return "enter resume  d/c/q"
	}
	return "enter resume  d details  c copy  q quit"
}

func renderDetails(card session.SessionCard, width int) []string {
	lines := []string{
		"    Project: " + emptyDash(card.ProjectPath),
		"    ID: " + emptyDash(card.ID),
		"    Source: " + emptyDash(card.SourcePath),
		"    Resume command: " + emptyDash(card.ResumeCommand().Display()),
		"    Prompt: " + emptyDash(card.FirstPrompt),
	}

	for i, line := range lines {
		lines[i] = truncate(line, width)
	}
	return lines
}

type rowCells struct {
	prefix  string
	title   string
	project string
	age     string
	tool    string
}

type rowColumns struct {
	title   int
	project int
	age     int
	tool    int
}

func rowLayout(width int, showHarness bool, cards []session.SessionCard) rowColumns {
	if width <= 0 {
		width = 120
	}

	cols := rowColumns{project: 18, age: 6}
	if showHarness {
		cols.tool = 6
	}

	gaps := 4
	if showHarness {
		gaps = 6
	}
	availableTitle := width - 2 - gaps - cols.project - cols.age - cols.tool
	if availableTitle >= 12 {
		cols.title = minInt(titleColumnWidth(cards), availableTitle)
		return cols
	}

	deficit := 12 - availableTitle
	if cols.project-deficit >= 8 {
		cols.project -= deficit
		cols.title = 12
		return cols
	}

	cols.project = 8
	cols.title = width - 2 - gaps - cols.project - cols.age - cols.tool
	if cols.title < 4 {
		cols.title = 4
	}
	return cols
}

func titleColumnWidth(cards []session.SessionCard) int {
	width := lipgloss.Width("Title")
	for _, card := range cards {
		width = maxInt(width, lipgloss.Width(oneLine(card.DisplayTitle())))
	}
	if width < 18 {
		return 18
	}
	if width > 64 {
		return 64
	}
	return width
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func joinRow(cells rowCells, showHarness bool) string {
	parts := []string{cells.prefix + cells.title, cells.project, cells.age}
	if showHarness {
		parts = append(parts, cells.tool)
	}
	return strings.Join(parts, "  ")
}

func padCell(s string, width int) string {
	s = truncate(s, width)
	padding := width - lipgloss.Width(s)
	if padding <= 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
}

func fitDetails(details []string, height int) []string {
	maxDetails := height - 4
	if maxDetails <= 0 || len(details) <= maxDetails {
		return details
	}
	return details[:maxDetails]
}

func mixedHarnesses(cards []session.SessionCard) bool {
	if len(cards) < 2 {
		return false
	}
	first := cards[0].Harness
	for _, card := range cards[1:] {
		if card.Harness != first {
			return true
		}
	}
	return false
}

func rowBudget(height int, detailLines int, compact bool) int {
	staticLines := 3
	if compact {
		staticLines = 2
	}
	lineBudget := height - staticLines - detailLines
	if lineBudget < 1 {
		return 1
	}
	if !compact {
		return lineBudget
	}

	rows := lineBudget / 2
	if rows < 1 {
		return 1
	}
	return rows
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
	return relativeTime(t, time.Now())
}

func relativeTime(t time.Time, now time.Time) string {
	delta := now.Sub(t)
	suffix := "ago"
	if delta < 0 {
		delta = -delta
		suffix = "from now"
	}

	if delta < time.Minute {
		return "now"
	}
	if delta < time.Hour {
		return relativeUnit(int(delta.Minutes()), "m", suffix)
	}
	if delta < 24*time.Hour {
		return relativeUnit(int(delta.Hours()), "h", suffix)
	}

	days := int(delta.Hours() / 24)
	if days < 14 {
		return relativeUnit(days, "d", suffix)
	}
	if days < 60 {
		return relativeUnit(days/7, "w", suffix)
	}
	if days < 365 {
		return relativeUnit(days/30, "mo", suffix)
	}
	return relativeUnit(days/365, "y", suffix)
}

func relativeUnit(value int, unit string, suffix string) string {
	if suffix == "from now" {
		return fmt.Sprintf("in %d%s", value, unit)
	}
	return fmt.Sprintf("%d%s", value, unit)
}

func emptyDash(s string) string {
	s = oneLine(s)
	if s == "" {
		return "-"
	}
	return s
}

func projectLabel(projectPath string) string {
	projectPath = oneLine(projectPath)
	if projectPath == "" {
		return "-"
	}
	base := filepath.Base(filepath.Clean(projectPath))
	if base == "." || base == string(filepath.Separator) || base == "" {
		return projectPath
	}
	return base
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
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
