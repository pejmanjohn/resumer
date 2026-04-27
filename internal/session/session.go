package session

import (
	"strings"
	"time"
)

type Harness string

const (
	HarnessClaude Harness = "claude"
	HarnessCodex  Harness = "codex"
)

type SessionCard struct {
	Harness     Harness
	ID          string
	Title       string
	ProjectPath string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	FirstPrompt string
	Model       string
	SourcePath  string
	Sidechain   bool
	Internal    bool
}

type ResumeCommand struct {
	Argv []string
	Dir  string
}

func (c SessionCard) ResumeCommand() ResumeCommand {
	switch c.Harness {
	case HarnessClaude:
		return ResumeCommand{
			Argv: []string{"claude", "--resume", c.ID},
			Dir:  c.ProjectPath,
		}
	case HarnessCodex:
		argv := []string{"codex", "resume", c.ID}
		if c.ProjectPath != "" {
			argv = append(argv, "--cd", c.ProjectPath)
		}
		return ResumeCommand{Argv: argv}
	default:
		return ResumeCommand{}
	}
}

func (c SessionCard) DisplayTitle() string {
	if title := strings.TrimSpace(c.Title); title != "" {
		return title
	}
	if prompt := strings.TrimSpace(c.FirstPrompt); prompt != "" {
		return prompt
	}
	id := c.ID
	if len(id) > 8 {
		id = id[len(id)-8:]
	}
	return "session " + id
}

func (c SessionCard) SortTime() time.Time {
	if !c.UpdatedAt.IsZero() {
		return c.UpdatedAt
	}
	return c.CreatedAt
}

func (c ResumeCommand) Display() string {
	parts := make([]string, 0, len(c.Argv)+3)
	if c.Dir != "" {
		parts = append(parts, "cd", shellQuote(c.Dir), "&&")
	}
	for _, arg := range c.Argv {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	for _, r := range s {
		if !isShellSafe(r) {
			return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
		}
	}
	return s
}

func isShellSafe(r rune) bool {
	return ('a' <= r && r <= 'z') ||
		('A' <= r && r <= 'Z') ||
		('0' <= r && r <= '9') ||
		strings.ContainsRune("@%_+=:,./-", r)
}
