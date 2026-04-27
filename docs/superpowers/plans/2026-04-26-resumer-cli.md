# Resumer CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Resumer Go CLI MVP that discovers Claude Code and Codex sessions, shows a human-first picker, and resumes or prints the correct command without mutating session state.

**Architecture:** Use a small Go module with `cmd/resumer/main.go` delegating to `internal/cmd`. Discovery packages normalize Claude and Codex records into `internal/session.SessionCard`; ranking, JSON output, picker rendering, clipboard, and runner packages consume that normalized model. Each task below is scoped for a fresh subagent and includes explicit red/green TDD checkpoints.

**Tech Stack:** Go 1.26+, Kong for CLI parsing, Bubble Tea v1 and Lip Gloss for TUI, Go standard library JSON/time/path/process APIs, tmux via argv-backed process runner.

---

## Execution Notes for Subagents

- Use `superpowers:test-driven-development` for every task that writes production code.
- Do not write production code before the failing test for that behavior has been created and run.
- The current workspace was observed without `.git` metadata. Before implementation, the coordinator must either initialize Git or confirm a dedicated worktree/repo exists. The commit steps below assume Git exists.
- Do not read real Claude/Codex transcript content into committed fixtures. Use sanitized fixture strings only.
- Keep normal resume execution argv-based. Only tmux command payloads need shell quoting because tmux starts a shell command.

## Source Documents

- Design source: `docs/plans/2026-04-26-001-feat-resumer-cli-plan.md`
- Original sketch: `session-resume-cli-design.md`

## File Responsibility Map

- `go.mod`: module declaration and dependency roots.
- `cmd/resumer/main.go`: tiny executable entrypoint that calls `internal/cmd.Main`.
- `internal/cmd/root.go`: Kong CLI structs, parse helpers for tests, orchestration shell.
- `internal/cmd/exit.go`: stable exit code constants and error classification.
- `internal/errfmt/errfmt.go`: human error formatting that keeps JSON stdout clean.
- `internal/config/paths.go`: default paths and `RESUMER_*` env override resolution.
- `internal/session/session.go`: normalized session card, harness enum, resume command model.
- `internal/discovery/claude.go`: Claude index and transcript discovery.
- `internal/discovery/codex.go`: Codex index and transcript enrichment.
- `internal/rank/rank.go`: harness filtering, sidechain filtering, cwd boost, deterministic sort, limit.
- `internal/outfmt/json.go`: stable JSON list encoder.
- `internal/picker/picker.go`: Bubble Tea update model and selected action state.
- `internal/picker/render.go`: Lip Gloss rendering, truncation, details, empty state.
- `internal/clipboard/clipboard.go`: clipboard adapter with injectable command runner.
- `internal/runner/command.go`: command display/quoting helpers.
- `internal/runner/runner.go`: default print/exec/spawn runner with injectable process executor.
- `internal/runner/tmux.go`: tmux session name, attach/new behavior, shell command payload.
- `README.md`: MVP usage, env vars, safety constraints, tmux notes.

## Task Order

1. Scaffold CLI contract and exit handling.
2. Add normalized session model and config path resolution.
3. Add Claude discovery.
4. Add Codex discovery.
5. Add ranking and JSON output.
6. Add picker model and rendering.
7. Add runner, clipboard, and tmux behavior.
8. Wire command orchestration and README/help polish.

---

### Task 1: Scaffold CLI Contract and Exit Handling

**Files:**
- Create: `go.mod`
- Create: `cmd/resumer/main.go`
- Create: `internal/cmd/root.go`
- Create: `internal/cmd/root_test.go`
- Create: `internal/cmd/exit.go`
- Create: `internal/cmd/exit_test.go`
- Create: `internal/errfmt/errfmt.go`
- Create: `internal/errfmt/errfmt_test.go`

- [ ] **Step 1: Create module skeleton only**

Create these files with package declarations only, plus `go.mod`.

```go
// go.mod
module resumer

go 1.26

require github.com/alecthomas/kong v1.15.0
```

```go
// cmd/resumer/main.go
package main

import "os"

func main() {
	os.Exit(0)
}
```

```go
// internal/cmd/root.go
package cmd
```

```go
// internal/cmd/exit.go
package cmd
```

```go
// internal/errfmt/errfmt.go
package errfmt
```

- [ ] **Step 2: Write failing CLI parse tests**

```go
// internal/cmd/root_test.go
package cmd

import "testing"

func TestParseNoArgsDefaultsToInteractiveAllHarnesses(t *testing.T) {
	opts, err := ParseForTest([]string{})
	if err != nil {
		t.Fatalf("ParseForTest returned error: %v", err)
	}
	if opts.Mode != ModeInteractive {
		t.Fatalf("Mode = %v, want %v", opts.Mode, ModeInteractive)
	}
	if opts.Harness != HarnessAll {
		t.Fatalf("Harness = %v, want %v", opts.Harness, HarnessAll)
	}
	if opts.Limit != 50 {
		t.Fatalf("Limit = %d, want 50", opts.Limit)
	}
}

func TestParseHarnessFilters(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want HarnessFilter
	}{
		{name: "claude", args: []string{"claude"}, want: HarnessClaude},
		{name: "codex", args: []string{"codex"}, want: HarnessCodex},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseForTest(tt.args)
			if err != nil {
				t.Fatalf("ParseForTest returned error: %v", err)
			}
			if opts.Harness != tt.want {
				t.Fatalf("Harness = %v, want %v", opts.Harness, tt.want)
			}
			if opts.Mode != ModeInteractive {
				t.Fatalf("Mode = %v, want %v", opts.Mode, ModeInteractive)
			}
		})
	}
}

func TestParseListJSON(t *testing.T) {
	opts, err := ParseForTest([]string{"list", "--json"})
	if err != nil {
		t.Fatalf("ParseForTest returned error: %v", err)
	}
	if opts.Mode != ModeListJSON {
		t.Fatalf("Mode = %v, want %v", opts.Mode, ModeListJSON)
	}
}

func TestParsePrintAndTmuxFlags(t *testing.T) {
	opts, err := ParseForTest([]string{"--print", "--tmux", "--limit", "25", "--all", "--cwd", "--debug"})
	if err != nil {
		t.Fatalf("ParseForTest returned error: %v", err)
	}
	if !opts.Print || !opts.Tmux || !opts.All || !opts.CWDBias || !opts.Debug {
		t.Fatalf("flags not preserved: %#v", opts)
	}
	if opts.Limit != 25 {
		t.Fatalf("Limit = %d, want 25", opts.Limit)
	}
}

func TestParseRejectsInvalidLimit(t *testing.T) {
	_, err := ParseForTest([]string{"--limit", "0"})
	if err == nil {
		t.Fatal("ParseForTest returned nil error for invalid limit")
	}
}
```

- [ ] **Step 3: Run tests to verify red**

Run:

```bash
go mod tidy
go test ./internal/cmd
```

Expected: FAIL with compiler errors like `undefined: ParseForTest`, `undefined: ModeInteractive`, and `undefined: HarnessAll`.

- [ ] **Step 4: Implement minimal CLI contract**

```go
// internal/cmd/root.go
package cmd

import "github.com/alecthomas/kong"

type Mode string

const (
	ModeInteractive Mode = "interactive"
	ModeListJSON    Mode = "list-json"
)

type HarnessFilter string

const (
	HarnessAll    HarnessFilter = "all"
	HarnessClaude HarnessFilter = "claude"
	HarnessCodex  HarnessFilter = "codex"
)

type Options struct {
	Mode    Mode
	Harness HarnessFilter
	Limit   int
	All     bool
	CWDBias bool
	Debug   bool
	Print   bool
	Tmux    bool
}

type Root struct {
	Limit int  `default:"50" help:"Maximum sessions to show."`
	All   bool `help:"Include old, sidechain, and noisy sessions."`
	CWD   bool `name:"cwd" help:"Bias current working directory sessions higher."`
	Debug bool `help:"Print discovery diagnostics to stderr."`
	Print bool `help:"Print selected resume command instead of executing it."`
	Tmux  bool `help:"Launch selected resume command inside tmux."`

	Claude HarnessCmd `cmd:"" help:"Show Claude Code sessions only."`
	Codex  HarnessCmd `cmd:"" help:"Show Codex sessions only."`
	List   ListCmd    `cmd:"" help:"List sessions for scripts."`
}

type HarnessCmd struct{}

type ListCmd struct {
	JSON bool `name:"json" help:"Emit stable JSON output."`
}

func ParseForTest(args []string) (Options, error) {
	root := Root{}
	parser, err := kong.New(&root, kong.Name("resumer"), kong.Exit(func(int) {}))
	if err != nil {
		return Options{}, err
	}

	ctx, err := parser.Parse(args)
	if err != nil {
		return Options{}, err
	}
	if root.Limit < 1 {
		return Options{}, UsageError{Message: "--limit must be greater than zero"}
	}

	opts := Options{
		Mode:    ModeInteractive,
		Harness: HarnessAll,
		Limit:   root.Limit,
		All:     root.All,
		CWDBias: root.CWD,
		Debug:   root.Debug,
		Print:   root.Print,
		Tmux:    root.Tmux,
	}

	switch ctx.Command() {
	case "claude":
		opts.Harness = HarnessClaude
	case "codex":
		opts.Harness = HarnessCodex
	case "list":
		if root.List.JSON {
			opts.Mode = ModeListJSON
		}
	}

	return opts, nil
}
```

```go
// internal/cmd/exit.go
package cmd

import "errors"

const (
	ExitOK        = 0
	ExitUsage     = 2
	ExitDiscovery = 20
	ExitEmpty     = 21
	ExitCanceled  = 30
	ExitLaunch    = 40
)

type UsageError struct {
	Message string
}

func (e UsageError) Error() string {
	return e.Message
}

type CanceledError struct{}

func (e CanceledError) Error() string {
	return "selection canceled"
}

type LaunchError struct {
	Message string
}

func (e LaunchError) Error() string {
	return e.Message
}

func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var usage UsageError
	if errors.As(err, &usage) {
		return ExitUsage
	}
	var canceled CanceledError
	if errors.As(err, &canceled) {
		return ExitCanceled
	}
	var launch LaunchError
	if errors.As(err, &launch) {
		return ExitLaunch
	}
	return ExitDiscovery
}
```

```go
// cmd/resumer/main.go
package main

import (
	"os"

	"resumer/internal/cmd"
)

func main() {
	os.Exit(cmd.ExitOK)
}
```

- [ ] **Step 5: Verify green**

Run:

```bash
go test ./internal/cmd
```

Expected: PASS.

- [ ] **Step 6: Write failing exit and error formatting tests**

```go
// internal/cmd/exit_test.go
package cmd

import "testing"

func TestExitCodeMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: ExitOK},
		{name: "usage", err: UsageError{Message: "bad flag"}, want: ExitUsage},
		{name: "canceled", err: CanceledError{}, want: ExitCanceled},
		{name: "launch", err: LaunchError{Message: "missing claude"}, want: ExitLaunch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCode(tt.err); got != tt.want {
				t.Fatalf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}
```

```go
// internal/errfmt/errfmt_test.go
package errfmt

import (
	"errors"
	"testing"
)

func TestHumanFormatsSingleLineError(t *testing.T) {
	got := Human(errors.New("missing claude binary"))
	want := "resumer: missing claude binary"
	if got != want {
		t.Fatalf("Human() = %q, want %q", got, want)
	}
}

func TestHumanNilIsEmpty(t *testing.T) {
	if got := Human(nil); got != "" {
		t.Fatalf("Human(nil) = %q, want empty string", got)
	}
}
```

- [ ] **Step 7: Run tests to verify red**

Run:

```bash
go test ./internal/cmd ./internal/errfmt
```

Expected: FAIL in `internal/errfmt` with `undefined: Human`.

- [ ] **Step 8: Implement minimal errfmt**

```go
// internal/errfmt/errfmt.go
package errfmt

func Human(err error) string {
	if err == nil {
		return ""
	}
	return "resumer: " + err.Error()
}
```

- [ ] **Step 9: Verify green and commit**

Run:

```bash
go test ./...
```

Expected: PASS.

Commit:

```bash
git add go.mod go.sum cmd/resumer/main.go internal/cmd internal/errfmt
git commit -m "feat: scaffold resumer cli contract"
```

---

### Task 2: Session Model and Config Path Resolution

**Files:**
- Create: `internal/session/session.go`
- Create: `internal/session/session_test.go`
- Create: `internal/config/paths.go`
- Create: `internal/config/paths_test.go`
- Modify: `internal/cmd/root.go`
- Test: `internal/session/session_test.go`
- Test: `internal/config/paths_test.go`

- [ ] **Step 1: Write failing session command tests**

```go
// internal/session/session_test.go
package session

import (
	"reflect"
	"testing"
	"time"
)

func TestResumeCommandClaudeWithProjectPath(t *testing.T) {
	card := SessionCard{Harness: HarnessClaude, ID: "claude-123", ProjectPath: "/repo/app"}
	cmd := card.ResumeCommand()

	if cmd.Dir != "/repo/app" {
		t.Fatalf("Dir = %q, want /repo/app", cmd.Dir)
	}
	if !reflect.DeepEqual(cmd.Argv, []string{"claude", "--resume", "claude-123"}) {
		t.Fatalf("Argv = %#v", cmd.Argv)
	}
	if got := cmd.Display(); got != "cd /repo/app && claude --resume claude-123" {
		t.Fatalf("Display() = %q", got)
	}
}

func TestResumeCommandCodexWithProjectPathUsesCD(t *testing.T) {
	card := SessionCard{Harness: HarnessCodex, ID: "codex-123", ProjectPath: "/repo/app"}
	cmd := card.ResumeCommand()

	if cmd.Dir != "" {
		t.Fatalf("Dir = %q, want empty string because codex receives --cd", cmd.Dir)
	}
	if !reflect.DeepEqual(cmd.Argv, []string{"codex", "resume", "codex-123", "--cd", "/repo/app"}) {
		t.Fatalf("Argv = %#v", cmd.Argv)
	}
}

func TestTitleFallback(t *testing.T) {
	card := SessionCard{ID: "0123456789abcdef", FirstPrompt: "Investigate websocket reconnect"}
	if got := card.DisplayTitle(); got != "Investigate websocket reconnect" {
		t.Fatalf("DisplayTitle() = %q", got)
	}

	card.FirstPrompt = ""
	if got := card.DisplayTitle(); got != "session 89abcdef" {
		t.Fatalf("DisplayTitle() = %q", got)
	}
}

func TestSortTimeFallsBackToCreatedAt(t *testing.T) {
	created := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	card := SessionCard{CreatedAt: created}
	if !card.SortTime().Equal(created) {
		t.Fatalf("SortTime() = %v, want %v", card.SortTime(), created)
	}
}
```

- [ ] **Step 2: Run session tests to verify red**

Run:

```bash
go test ./internal/session
```

Expected: FAIL with `undefined: SessionCard`.

- [ ] **Step 3: Implement minimal session model**

```go
// internal/session/session.go
package session

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Harness string

const (
	HarnessClaude Harness = "Claude"
	HarnessCodex  Harness = "Codex"
)

type SessionCard struct {
	Harness     Harness   `json:"harness"`
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	ProjectPath string    `json:"project_path,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	FirstPrompt string    `json:"first_prompt,omitempty"`
	Model       string    `json:"model,omitempty"`
	SourcePath  string    `json:"source_path,omitempty"`
	Sidechain   bool      `json:"-"`
	Internal    bool      `json:"-"`
}

type ResumeCommand struct {
	Argv []string
	Dir  string
}

func (c SessionCard) ResumeCommand() ResumeCommand {
	switch c.Harness {
	case HarnessClaude:
		return ResumeCommand{Argv: []string{"claude", "--resume", c.ID}, Dir: c.ProjectPath}
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
	if strings.TrimSpace(c.Title) != "" {
		return strings.TrimSpace(c.Title)
	}
	if strings.TrimSpace(c.FirstPrompt) != "" {
		return strings.TrimSpace(c.FirstPrompt)
	}
	if len(c.ID) > 8 {
		return "session " + c.ID[len(c.ID)-8:]
	}
	return "session " + c.ID
}

func (c SessionCard) SortTime() time.Time {
	if !c.UpdatedAt.IsZero() {
		return c.UpdatedAt
	}
	return c.CreatedAt
}

func (c ResumeCommand) Display() string {
	if len(c.Argv) == 0 {
		return ""
	}
	cmd := joinShell(c.Argv)
	if c.Dir != "" {
		return "cd " + shellQuote(c.Dir) + " && " + cmd
	}
	return cmd
}

func joinShell(parts []string) string {
	quoted := make([]string, len(parts))
	for i, part := range parts {
		quoted[i] = shellQuote(part)
	}
	return strings.Join(quoted, " ")
}

var safeShell = regexp.MustCompile(`^[A-Za-z0-9_./:=+-]+$`)

func shellQuote(s string) string {
	if safeShell.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func (c SessionCard) String() string {
	return fmt.Sprintf("%s %s", c.Harness, c.DisplayTitle())
}
```

- [ ] **Step 4: Verify session green**

Run:

```bash
go test ./internal/session
```

Expected: PASS.

- [ ] **Step 5: Write failing config tests**

```go
// internal/config/paths_test.go
package config

import (
	"path/filepath"
	"testing"
)

func TestLoadPathsDefaults(t *testing.T) {
	t.Setenv("HOME", "/home/ada")
	clearResumerEnv(t)

	paths, err := LoadPaths()
	if err != nil {
		t.Fatalf("LoadPaths returned error: %v", err)
	}
	if paths.ClaudeProjects != filepath.FromSlash("/home/ada/.claude/projects") {
		t.Fatalf("ClaudeProjects = %q", paths.ClaudeProjects)
	}
	if paths.CodexSessions != filepath.FromSlash("/home/ada/.codex/sessions") {
		t.Fatalf("CodexSessions = %q", paths.CodexSessions)
	}
	if paths.CodexIndex != filepath.FromSlash("/home/ada/.codex/session_index.jsonl") {
		t.Fatalf("CodexIndex = %q", paths.CodexIndex)
	}
}

func TestLoadPathsEnvOverrides(t *testing.T) {
	t.Setenv("HOME", "/home/ada")
	t.Setenv("RESUMER_CLAUDE_PROJECTS_PATH", "/tmp/claude")
	t.Setenv("RESUMER_CODEX_SESSIONS_PATH", "/tmp/codex/sessions")
	t.Setenv("RESUMER_CODEX_INDEX_PATH", "/tmp/codex/index.jsonl")
	t.Setenv("RESUMER_DEFAULT_TMUX", "1")
	t.Setenv("RESUMER_TMUX_HOST_HINT", "Ada-Mac-mini")

	paths, err := LoadPaths()
	if err != nil {
		t.Fatalf("LoadPaths returned error: %v", err)
	}
	if paths.ClaudeProjects != "/tmp/claude" || paths.CodexSessions != "/tmp/codex/sessions" || paths.CodexIndex != "/tmp/codex/index.jsonl" {
		t.Fatalf("paths did not honor overrides: %#v", paths)
	}
	if !paths.DefaultTmux {
		t.Fatal("DefaultTmux = false, want true")
	}
	if paths.TmuxHostHint != "Ada-Mac-mini" {
		t.Fatalf("TmuxHostHint = %q", paths.TmuxHostHint)
	}
}

func TestLoadPathsRejectsBadTmuxBoolean(t *testing.T) {
	t.Setenv("HOME", "/home/ada")
	clearResumerEnv(t)
	t.Setenv("RESUMER_DEFAULT_TMUX", "sometimes")

	_, err := LoadPaths()
	if err == nil {
		t.Fatal("LoadPaths returned nil error for bad boolean")
	}
}

func clearResumerEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"RESUMER_CLAUDE_PROJECTS_PATH",
		"RESUMER_CODEX_SESSIONS_PATH",
		"RESUMER_CODEX_INDEX_PATH",
		"RESUMER_DEFAULT_TMUX",
		"RESUMER_TMUX_HOST_HINT",
	} {
		t.Setenv(key, "")
	}
}
```

- [ ] **Step 6: Run config tests to verify red**

Run:

```bash
go test ./internal/config
```

Expected: FAIL with `undefined: LoadPaths`.

- [ ] **Step 7: Implement config path resolution**

```go
// internal/config/paths.go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Paths struct {
	ClaudeProjects string
	CodexSessions  string
	CodexIndex     string
	DefaultTmux    bool
	TmuxHostHint   string
}

func LoadPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	if envHome := os.Getenv("HOME"); envHome != "" {
		home = envHome
	}

	paths := Paths{
		ClaudeProjects: filepath.Join(home, ".claude", "projects"),
		CodexSessions:  filepath.Join(home, ".codex", "sessions"),
		CodexIndex:     filepath.Join(home, ".codex", "session_index.jsonl"),
		TmuxHostHint:   os.Getenv("RESUMER_TMUX_HOST_HINT"),
	}

	if v := os.Getenv("RESUMER_CLAUDE_PROJECTS_PATH"); v != "" {
		paths.ClaudeProjects = v
	}
	if v := os.Getenv("RESUMER_CODEX_SESSIONS_PATH"); v != "" {
		paths.CodexSessions = v
	}
	if v := os.Getenv("RESUMER_CODEX_INDEX_PATH"); v != "" {
		paths.CodexIndex = v
	}
	if v := os.Getenv("RESUMER_DEFAULT_TMUX"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Paths{}, fmt.Errorf("RESUMER_DEFAULT_TMUX: %w", err)
		}
		paths.DefaultTmux = b
	}

	return paths, nil
}
```

- [ ] **Step 8: Verify green and commit**

Run:

```bash
go test ./...
```

Expected: PASS.

Commit:

```bash
git add internal/session internal/config internal/cmd go.mod go.sum
git commit -m "feat: add session model and config paths"
```

---

### Task 3: Claude Discovery

**Files:**
- Create: `internal/discovery/claude.go`
- Create: `internal/discovery/claude_test.go`
- Create: `internal/discovery/testdata/claude/projects/project-a/sessions-index.json`
- Create: `internal/discovery/testdata/claude/projects/project-a/claude-session-1.jsonl`
- Test: `internal/discovery/claude_test.go`

- [ ] **Step 1: Add sanitized Claude fixtures**

```json
// internal/discovery/testdata/claude/projects/project-a/sessions-index.json
{
  "version": 1,
  "originalPath": "/repo/project-a",
  "entries": [
    {
      "sessionId": "claude-session-1",
      "summary": "Resumer CLI design",
      "firstPrompt": "Plan the resume picker",
      "messageCount": 12,
      "created": "2026-04-26T10:00:00Z",
      "modified": "2026-04-26T12:00:00Z",
      "gitBranch": "main",
      "projectPath": "/repo/project-a",
      "fullPath": "/home/ada/.claude/projects/project-a/claude-session-1.jsonl",
      "isSidechain": false
    },
    {
      "sessionId": "claude-sidechain-1",
      "summary": "Internal sidechain",
      "firstPrompt": "Hidden helper",
      "created": "2026-04-26T09:00:00Z",
      "modified": "2026-04-26T09:30:00Z",
      "projectPath": "/repo/project-a",
      "fullPath": "/home/ada/.claude/projects/project-a/sidechain.jsonl",
      "isSidechain": true
    }
  ]
}
```

```json
// internal/discovery/testdata/claude/projects/project-a/claude-session-1.jsonl
{"type":"summary","sessionId":"claude-session-1","timestamp":"2026-04-26T10:00:00Z"}
{"type":"user","sessionId":"claude-session-1","timestamp":"2026-04-26T10:01:00Z","cwd":"/repo/project-a","message":{"role":"user","content":"Plan the resume picker"}}
{"type":"assistant","sessionId":"claude-session-1","timestamp":"2026-04-26T10:02:00Z","cwd":"/repo/project-a","message":{"role":"assistant","content":"Sanitized response"}}
```

- [ ] **Step 2: Write failing Claude discovery tests**

```go
// internal/discovery/claude_test.go
package discovery

import (
	"path/filepath"
	"testing"

	"resumer/internal/session"
)

func TestDiscoverClaudeFromIndex(t *testing.T) {
	root := filepath.Join("testdata", "claude", "projects")
	cards, diagnostics := DiscoverClaude(ClaudeOptions{ProjectsPath: root})

	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
	if len(cards) != 1 {
		t.Fatalf("len(cards) = %d, want 1", len(cards))
	}
	card := cards[0]
	if card.Harness != session.HarnessClaude {
		t.Fatalf("Harness = %q", card.Harness)
	}
	if card.ID != "claude-session-1" || card.Title != "Resumer CLI design" || card.FirstPrompt != "Plan the resume picker" {
		t.Fatalf("card fields wrong: %#v", card)
	}
	if card.ProjectPath != "/repo/project-a" {
		t.Fatalf("ProjectPath = %q", card.ProjectPath)
	}
	if card.UpdatedAt.IsZero() || card.CreatedAt.IsZero() {
		t.Fatalf("timestamps not parsed: %#v", card)
	}
}

func TestDiscoverClaudeIncludesSidechainWhenRequested(t *testing.T) {
	root := filepath.Join("testdata", "claude", "projects")
	cards, _ := DiscoverClaude(ClaudeOptions{ProjectsPath: root, IncludeAll: true})

	if len(cards) != 2 {
		t.Fatalf("len(cards) = %d, want 2", len(cards))
	}
}

func TestDiscoverClaudeFallbackJSONL(t *testing.T) {
	path := filepath.Join("testdata", "claude", "projects", "project-a", "claude-session-1.jsonl")
	card, ok := parseClaudeJSONL(path)
	if !ok {
		t.Fatal("parseClaudeJSONL returned ok=false")
	}
	if card.ID != "claude-session-1" || card.FirstPrompt != "Plan the resume picker" {
		t.Fatalf("card = %#v", card)
	}
}
```

- [ ] **Step 3: Run tests to verify red**

Run:

```bash
go test ./internal/discovery
```

Expected: FAIL with `undefined: DiscoverClaude`.

- [ ] **Step 4: Implement Claude index and JSONL parser**

```go
// internal/discovery/claude.go
package discovery

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"resumer/internal/session"
)

type Diagnostic struct {
	Source  string
	Message string
}

type ClaudeOptions struct {
	ProjectsPath string
	IncludeAll   bool
}

type claudeIndex struct {
	Entries []claudeEntry `json:"entries"`
}

type claudeEntry struct {
	SessionID   string `json:"sessionId"`
	Summary     string `json:"summary"`
	FirstPrompt string `json:"firstPrompt"`
	Created     string `json:"created"`
	Modified    string `json:"modified"`
	ProjectPath string `json:"projectPath"`
	FullPath    string `json:"fullPath"`
	IsSidechain bool   `json:"isSidechain"`
}

func DiscoverClaude(opts ClaudeOptions) ([]session.SessionCard, []Diagnostic) {
	var cards []session.SessionCard
	var diagnostics []Diagnostic

	matches, err := filepath.Glob(filepath.Join(opts.ProjectsPath, "*", "sessions-index.json"))
	if err != nil {
		return nil, []Diagnostic{{Source: opts.ProjectsPath, Message: err.Error()}}
	}

	seen := map[string]bool{}
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{Source: path, Message: err.Error()})
			continue
		}
		var idx claudeIndex
		if err := json.Unmarshal(data, &idx); err != nil {
			diagnostics = append(diagnostics, Diagnostic{Source: path, Message: err.Error()})
			continue
		}
		for _, entry := range idx.Entries {
			if entry.SessionID == "" || seen[entry.SessionID] {
				continue
			}
			if entry.IsSidechain && !opts.IncludeAll {
				continue
			}
			seen[entry.SessionID] = true
			cards = append(cards, claudeEntryCard(entry))
		}
	}

	return cards, diagnostics
}

func claudeEntryCard(entry claudeEntry) session.SessionCard {
	return session.SessionCard{
		Harness:     session.HarnessClaude,
		ID:          entry.SessionID,
		Title:       strings.TrimSpace(entry.Summary),
		ProjectPath: entry.ProjectPath,
		CreatedAt:   parseTime(entry.Created),
		UpdatedAt:   parseTime(entry.Modified),
		FirstPrompt: strings.TrimSpace(entry.FirstPrompt),
		SourcePath:  entry.FullPath,
		Sidechain:   entry.IsSidechain,
	}
}

func parseClaudeJSONL(path string) (session.SessionCard, bool) {
	file, err := os.Open(path)
	if err != nil {
		return session.SessionCard{}, false
	}
	defer file.Close()

	var card session.SessionCard
	card.Harness = session.HarnessClaude
	card.SourcePath = path

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var line struct {
			Type      string `json:"type"`
			SessionID string `json:"sessionId"`
			Timestamp string `json:"timestamp"`
			CWD       string `json:"cwd"`
			Message   struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(scanner.Bytes(), &line) != nil {
			continue
		}
		if card.ID == "" {
			card.ID = line.SessionID
		}
		if card.CreatedAt.IsZero() {
			card.CreatedAt = parseTime(line.Timestamp)
		}
		card.UpdatedAt = parseTime(line.Timestamp)
		if card.ProjectPath == "" {
			card.ProjectPath = line.CWD
		}
		if card.FirstPrompt == "" && line.Type == "user" && line.Message.Role == "user" {
			card.FirstPrompt = strings.TrimSpace(line.Message.Content)
		}
	}

	return card, card.ID != ""
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339Nano, value)
	return t
}
```

- [ ] **Step 5: Verify green and commit**

Run:

```bash
go test ./...
```

Expected: PASS.

Commit:

```bash
git add internal/discovery
git commit -m "feat: discover claude sessions"
```

---

### Task 4: Codex Discovery and Enrichment

**Files:**
- Create: `internal/discovery/codex.go`
- Create: `internal/discovery/codex_test.go`
- Create: `internal/discovery/testdata/codex/session_index.jsonl`
- Create: `internal/discovery/testdata/codex/sessions/2026/04/26/rollout-2026-04-26T10-00-00-codex-session-1.jsonl`
- Test: `internal/discovery/codex_test.go`

- [ ] **Step 1: Add sanitized Codex fixtures**

```json
// internal/discovery/testdata/codex/session_index.jsonl
{"id":"codex-session-1","thread_name":"Explore calculator app","updated_at":"2026-04-26T13:00:00Z"}
{"id":"codex-session-2","thread_name":"","updated_at":"2026-04-26T12:00:00Z"}
{"id":"codex-session-1","thread_name":"Explore calculator app updated","updated_at":"2026-04-26T14:00:00Z"}
```

```json
// internal/discovery/testdata/codex/sessions/2026/04/26/rollout-2026-04-26T10-00-00-codex-session-1.jsonl
{"timestamp":"2026-04-26T10:00:00Z","type":"session_meta","payload":{"id":"codex-session-1","cwd":"/repo/calc","model_provider":"openai"}}
{"timestamp":"2026-04-26T10:01:00Z","type":"event_msg","payload":{"type":"message","message":"sanitized"}}
```

- [ ] **Step 2: Write failing Codex discovery tests**

```go
// internal/discovery/codex_test.go
package discovery

import (
	"path/filepath"
	"testing"

	"resumer/internal/session"
)

func TestDiscoverCodexFromIndex(t *testing.T) {
	cards, diagnostics := DiscoverCodex(CodexOptions{
		IndexPath:    filepath.Join("testdata", "codex", "session_index.jsonl"),
		SessionsPath: filepath.Join("testdata", "codex", "sessions"),
	})

	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
	if len(cards) != 2 {
		t.Fatalf("len(cards) = %d, want 2", len(cards))
	}
	first := cards[0]
	if first.Harness != session.HarnessCodex || first.ID != "codex-session-1" {
		t.Fatalf("first card = %#v", first)
	}
	if first.Title != "Explore calculator app updated" {
		t.Fatalf("Title = %q", first.Title)
	}
	if first.ProjectPath != "/repo/calc" {
		t.Fatalf("ProjectPath = %q", first.ProjectPath)
	}
}

func TestDiscoverCodexTitleFallbackForBlankThreadName(t *testing.T) {
	cards, _ := DiscoverCodex(CodexOptions{
		IndexPath:    filepath.Join("testdata", "codex", "session_index.jsonl"),
		SessionsPath: filepath.Join("testdata", "codex", "sessions"),
	})

	var blank session.SessionCard
	for _, card := range cards {
		if card.ID == "codex-session-2" {
			blank = card
		}
	}
	if blank.DisplayTitle() != "session session-2" {
		t.Fatalf("DisplayTitle() = %q", blank.DisplayTitle())
	}
}

func TestDiscoverCodexMissingIndexIsDiagnosticNotPanic(t *testing.T) {
	cards, diagnostics := DiscoverCodex(CodexOptions{IndexPath: "missing.jsonl"})
	if len(cards) != 0 {
		t.Fatalf("cards = %#v, want empty", cards)
	}
	if len(diagnostics) == 0 {
		t.Fatal("diagnostics empty, want missing index diagnostic")
	}
}
```

- [ ] **Step 3: Run tests to verify red**

Run:

```bash
go test ./internal/discovery
```

Expected: FAIL with `undefined: DiscoverCodex`.

- [ ] **Step 4: Implement Codex discovery**

```go
// internal/discovery/codex.go
package discovery

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"resumer/internal/session"
)

type CodexOptions struct {
	IndexPath    string
	SessionsPath string
}

type codexIndexLine struct {
	ID         string `json:"id"`
	ThreadName string `json:"thread_name"`
	UpdatedAt  string `json:"updated_at"`
}

func DiscoverCodex(opts CodexOptions) ([]session.SessionCard, []Diagnostic) {
	file, err := os.Open(opts.IndexPath)
	if err != nil {
		return nil, []Diagnostic{{Source: opts.IndexPath, Message: err.Error()}}
	}
	defer file.Close()

	byID := map[string]session.SessionCard{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var line codexIndexLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.ID == "" {
			continue
		}
		card := session.SessionCard{
			Harness:   session.HarnessCodex,
			ID:        line.ID,
			Title:     strings.TrimSpace(line.ThreadName),
			UpdatedAt: parseTime(line.UpdatedAt),
		}
		if existing, ok := byID[line.ID]; ok && !existing.UpdatedAt.Before(card.UpdatedAt) {
			continue
		}
		enrichCodexFromTranscript(opts.SessionsPath, &card)
		byID[line.ID] = card
	}

	cards := make([]session.SessionCard, 0, len(byID))
	for _, card := range byID {
		cards = append(cards, card)
	}
	return cards, nil
}

func enrichCodexFromTranscript(root string, card *session.SessionCard) {
	if root == "" || card.ID == "" {
		return
	}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		if !strings.Contains(path, card.ID) {
			return nil
		}
		if meta, ok := parseCodexSessionMeta(path, card.ID); ok {
			card.ProjectPath = meta.ProjectPath
			card.Model = meta.Model
			card.SourcePath = path
			if !meta.CreatedAt.IsZero() {
				card.CreatedAt = meta.CreatedAt
			}
		}
		return nil
	})
}

type codexMeta struct {
	ProjectPath string
	Model       string
	CreatedAt   time.Time
}

func parseCodexSessionMeta(path string, wantID string) (codexMeta, bool) {
	file, err := os.Open(path)
	if err != nil {
		return codexMeta{}, false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var line struct {
			Timestamp string `json:"timestamp"`
			Type      string `json:"type"`
			Payload   struct {
				ID            string `json:"id"`
				CWD           string `json:"cwd"`
				ModelProvider string `json:"model_provider"`
			} `json:"payload"`
		}
		if json.Unmarshal(scanner.Bytes(), &line) != nil || line.Type != "session_meta" {
			continue
		}
		if line.Payload.ID != "" && line.Payload.ID != wantID {
			continue
		}
		return codexMeta{
			ProjectPath: line.Payload.CWD,
			Model:       line.Payload.ModelProvider,
			CreatedAt:   parseTime(line.Timestamp),
		}, true
	}
	return codexMeta{}, false
}
```

- [ ] **Step 5: Verify green**

Run:

```bash
gofmt -w internal/discovery/codex.go internal/discovery/codex_test.go
go test ./...
```

Expected: PASS.

Commit:

```bash
git add internal/discovery
git commit -m "feat: discover codex sessions"
```

---

### Task 5: Ranking and JSON Output

**Files:**
- Create: `internal/rank/rank.go`
- Create: `internal/rank/rank_test.go`
- Create: `internal/outfmt/json.go`
- Create: `internal/outfmt/json_test.go`
- Test: `internal/rank/rank_test.go`
- Test: `internal/outfmt/json_test.go`

- [ ] **Step 1: Write failing ranking tests**

```go
// internal/rank/rank_test.go
package rank

import (
	"testing"
	"time"

	"resumer/internal/session"
)

func TestApplySortsByUpdatedAtDescendingAndLimits(t *testing.T) {
	cards := []session.SessionCard{
		card("old", session.HarnessClaude, "2026-04-26T10:00:00Z", ""),
		card("new", session.HarnessCodex, "2026-04-26T12:00:00Z", ""),
		card("mid", session.HarnessClaude, "2026-04-26T11:00:00Z", ""),
	}
	got := Apply(cards, Options{Limit: 2})

	if len(got) != 2 || got[0].ID != "new" || got[1].ID != "mid" {
		t.Fatalf("ranked = %#v", got)
	}
}

func TestApplyFiltersHarnessAndSidechain(t *testing.T) {
	cards := []session.SessionCard{
		{ID: "claude", Harness: session.HarnessClaude},
		{ID: "codex", Harness: session.HarnessCodex},
		{ID: "side", Harness: session.HarnessClaude, Sidechain: true},
	}

	got := Apply(cards, Options{Harness: session.HarnessClaude, Limit: 50})
	if len(got) != 1 || got[0].ID != "claude" {
		t.Fatalf("filtered = %#v", got)
	}

	got = Apply(cards, Options{Harness: session.HarnessClaude, IncludeAll: true, Limit: 50})
	if len(got) != 2 {
		t.Fatalf("filtered include all = %#v", got)
	}
}

func TestApplyCWDBias(t *testing.T) {
	cards := []session.SessionCard{
		card("outside", session.HarnessClaude, "2026-04-26T12:00:00Z", "/repo/other"),
		card("inside", session.HarnessCodex, "2026-04-26T11:00:00Z", "/repo/app"),
	}
	got := Apply(cards, Options{Limit: 50, CWDBiasPath: "/repo/app"})

	if got[0].ID != "inside" {
		t.Fatalf("first = %s, want inside", got[0].ID)
	}
}

func card(id string, harness session.Harness, ts string, path string) session.SessionCard {
	t, _ := time.Parse(time.RFC3339, ts)
	return session.SessionCard{ID: id, Harness: harness, UpdatedAt: t, ProjectPath: path}
}
```

- [ ] **Step 2: Run ranking tests to verify red**

Run:

```bash
go test ./internal/rank
```

Expected: FAIL with `undefined: Apply`.

- [ ] **Step 3: Implement ranking**

```go
// internal/rank/rank.go
package rank

import (
	"sort"
	"strings"

	"resumer/internal/session"
)

type Options struct {
	Harness     session.Harness
	IncludeAll  bool
	CWDBiasPath string
	Limit       int
}

func Apply(cards []session.SessionCard, opts Options) []session.SessionCard {
	filtered := make([]session.SessionCard, 0, len(cards))
	for _, card := range cards {
		if opts.Harness != "" && card.Harness != opts.Harness {
			continue
		}
		if !opts.IncludeAll && (card.Sidechain || card.Internal) {
			continue
		}
		filtered = append(filtered, card)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		leftBias := cwdMatch(filtered[i].ProjectPath, opts.CWDBiasPath)
		rightBias := cwdMatch(filtered[j].ProjectPath, opts.CWDBiasPath)
		if leftBias != rightBias {
			return leftBias
		}
		if !filtered[i].SortTime().Equal(filtered[j].SortTime()) {
			return filtered[i].SortTime().After(filtered[j].SortTime())
		}
		if filtered[i].Harness != filtered[j].Harness {
			return filtered[i].Harness < filtered[j].Harness
		}
		return filtered[i].DisplayTitle() < filtered[j].DisplayTitle()
	})

	if opts.Limit > 0 && len(filtered) > opts.Limit {
		return filtered[:opts.Limit]
	}
	return filtered
}

func cwdMatch(projectPath string, cwd string) bool {
	if projectPath == "" || cwd == "" {
		return false
	}
	return projectPath == cwd || strings.HasPrefix(projectPath, strings.TrimRight(cwd, "/")+"/")
}
```

- [ ] **Step 4: Verify ranking green**

Run:

```bash
go test ./internal/rank
```

Expected: PASS.

- [ ] **Step 5: Write failing JSON output tests**

```go
// internal/outfmt/json_test.go
package outfmt

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"resumer/internal/session"
)

func TestWriteJSONIncludesStableFields(t *testing.T) {
	ts := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	cards := []session.SessionCard{{
		Harness:     session.HarnessCodex,
		ID:          "codex-1",
		Title:       "Explore calculator",
		ProjectPath: "/repo/calc",
		UpdatedAt:   ts,
		FirstPrompt: "Open calculator",
		SourcePath:  "/tmp/codex.jsonl",
	}}

	var buf bytes.Buffer
	if err := WriteJSON(&buf, cards); err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	var decoded struct {
		Sessions []struct {
			Harness string `json:"harness"`
			ID      string `json:"id"`
			Command string `json:"command"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(decoded.Sessions) != 1 {
		t.Fatalf("sessions len = %d", len(decoded.Sessions))
	}
	if decoded.Sessions[0].Command != "codex resume codex-1 --cd /repo/calc" {
		t.Fatalf("command = %q", decoded.Sessions[0].Command)
	}
}
```

- [ ] **Step 6: Run JSON tests to verify red**

Run:

```bash
go test ./internal/outfmt
```

Expected: FAIL with `undefined: WriteJSON`.

- [ ] **Step 7: Implement JSON writer**

```go
// internal/outfmt/json.go
package outfmt

import (
	"encoding/json"
	"io"
	"time"

	"resumer/internal/session"
)

type JSONDocument struct {
	Sessions []JSONSession `json:"sessions"`
}

type JSONSession struct {
	Harness     session.Harness `json:"harness"`
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	ProjectPath string          `json:"project_path,omitempty"`
	UpdatedAt   *time.Time      `json:"updated_at,omitempty"`
	CreatedAt   *time.Time      `json:"created_at,omitempty"`
	FirstPrompt string          `json:"first_prompt,omitempty"`
	Model       string          `json:"model,omitempty"`
	SourcePath  string          `json:"source_path,omitempty"`
	Command     string          `json:"command"`
}

func WriteJSON(w io.Writer, cards []session.SessionCard) error {
	doc := JSONDocument{Sessions: make([]JSONSession, 0, len(cards))}
	for _, card := range cards {
		row := JSONSession{
			Harness:     card.Harness,
			ID:          card.ID,
			Title:       card.DisplayTitle(),
			ProjectPath: card.ProjectPath,
			FirstPrompt: card.FirstPrompt,
			Model:       card.Model,
			SourcePath:  card.SourcePath,
			Command:     card.ResumeCommand().Display(),
		}
		if !card.UpdatedAt.IsZero() {
			row.UpdatedAt = &card.UpdatedAt
		}
		if !card.CreatedAt.IsZero() {
			row.CreatedAt = &card.CreatedAt
		}
		doc.Sessions = append(doc.Sessions, row)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}
```

- [ ] **Step 8: Verify green and commit**

Run:

```bash
go test ./...
```

Expected: PASS.

Commit:

```bash
git add internal/rank internal/outfmt
git commit -m "feat: rank sessions and write json output"
```

---

### Task 6: Interactive Picker Model and Rendering

**Files:**
- Create: `internal/picker/picker.go`
- Create: `internal/picker/render.go`
- Create: `internal/picker/picker_test.go`
- Modify: `go.mod`
- Test: `internal/picker/picker_test.go`

- [ ] **Step 1: Add TUI dependencies**

Run:

```bash
go get github.com/charmbracelet/bubbletea@v1.3.10 github.com/charmbracelet/lipgloss@v1.1.0
```

Expected: `go.mod` and `go.sum` include Bubble Tea v1 and Lip Gloss.

- [ ] **Step 2: Write failing picker tests**

```go
// internal/picker/picker_test.go
package picker

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"resumer/internal/session"
)

func TestInitialViewShowsSelectedSessionAndFooter(t *testing.T) {
	model := New([]session.SessionCard{sampleCard("one")})
	view := model.View()

	for _, want := range []string{"Resume a session", "Codex", "Project one", "enter resume", "d details", "c copy command"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
}

func TestNavigationDoesNotMovePastBounds(t *testing.T) {
	model := New([]session.SessionCard{sampleCard("one"), sampleCard("two")})
	model, _ = updateForTest(model, tea.KeyMsg{Type: tea.KeyDown})
	model, _ = updateForTest(model, tea.KeyMsg{Type: tea.KeyDown})
	if model.Cursor != 1 {
		t.Fatalf("Cursor = %d, want 1", model.Cursor)
	}
	model, _ = updateForTest(model, tea.KeyMsg{Type: tea.KeyUp})
	model, _ = updateForTest(model, tea.KeyMsg{Type: tea.KeyUp})
	if model.Cursor != 0 {
		t.Fatalf("Cursor = %d, want 0", model.Cursor)
	}
}

func TestDetailsToggleAndSelection(t *testing.T) {
	model := New([]session.SessionCard{sampleCard("one")})
	model, _ = updateForTest(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !model.ShowDetails {
		t.Fatal("ShowDetails = false, want true")
	}
	if !strings.Contains(model.View(), "ID:") {
		t.Fatalf("details view missing ID:\n%s", model.View())
	}

	model, _ = updateForTest(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.Action != ActionResume || model.Selected == nil || model.Selected.ID != "one" {
		t.Fatalf("selection not recorded: %#v", model)
	}
}

func TestCopyActionAndCancel(t *testing.T) {
	model := New([]session.SessionCard{sampleCard("one")})
	model, _ = updateForTest(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if model.Action != ActionCopy || model.Selected == nil {
		t.Fatalf("copy action not recorded: %#v", model)
	}

	model = New([]session.SessionCard{sampleCard("one")})
	model, _ = updateForTest(model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.Action != ActionCancel {
		t.Fatalf("Action = %v, want cancel", model.Action)
	}
}

func sampleCard(id string) session.SessionCard {
	return session.SessionCard{
		Harness:     session.HarnessCodex,
		ID:          id,
		Title:       "Project " + id,
		ProjectPath: "/repo/" + id,
		UpdatedAt:   time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
		FirstPrompt: "Open calculator",
	}
}
```

- [ ] **Step 3: Run picker tests to verify red**

Run:

```bash
go test ./internal/picker
```

Expected: FAIL with `undefined: New`.

- [ ] **Step 4: Implement picker model**

```go
// internal/picker/picker.go
package picker

import (
	tea "github.com/charmbracelet/bubbletea"

	"resumer/internal/session"
)

type Action string

const (
	ActionNone   Action = ""
	ActionResume Action = "resume"
	ActionCopy   Action = "copy"
	ActionCancel Action = "cancel"
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
	return Model{Sessions: sessions, Width: 100, Height: 30}
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
		switch msg.String() {
		case "down", "j":
			if m.Cursor < len(m.Sessions)-1 {
				m.Cursor++
			}
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "enter":
			m.selectAction(ActionResume)
			return m, tea.Quit
		case "c":
			m.selectAction(ActionCopy)
		case "d":
			m.ShowDetails = !m.ShowDetails
		case "q", "esc", "ctrl+c":
			m.Action = ActionCancel
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *Model) selectAction(action Action) {
	m.Action = action
	if len(m.Sessions) == 0 {
		return
	}
	selected := m.Sessions[m.Cursor]
	m.Selected = &selected
}
```

- [ ] **Step 5: Implement rendering**

```go
// internal/picker/render.go
package picker

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Bold(true)
	dimStyle      = lipgloss.NewStyle().Faint(true)
)

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Resume a session"))
	b.WriteString("\n\n")

	if len(m.Sessions) == 0 {
		b.WriteString("No sessions found. Try --all or check RESUMER_* path overrides.\n")
		b.WriteString("\nq quit\n")
		return b.String()
	}

	for i, card := range m.Sessions {
		prefix := "  "
		style := lipgloss.NewStyle()
		if i == m.Cursor {
			prefix = "> "
			style = selectedStyle
		}
		b.WriteString(style.Render(fmt.Sprintf("%s%s   %s   %s", prefix, card.Harness, relative(card.SortTime()), truncate(card.ProjectPath, 48))))
		b.WriteString("\n")
		b.WriteString("   " + truncate(card.DisplayTitle(), 72) + "\n")
		if card.FirstPrompt != "" {
			b.WriteString("   " + dimStyle.Render(truncate(card.FirstPrompt, 72)) + "\n")
		}
		if m.ShowDetails && i == m.Cursor {
			b.WriteString(fmt.Sprintf("   ID: %s\n", card.ID))
			b.WriteString(fmt.Sprintf("   Source: %s\n", card.SourcePath))
			b.WriteString(fmt.Sprintf("   Resume command: %s\n", card.ResumeCommand().Display()))
		}
		b.WriteString("\n")
	}

	b.WriteString("up/down move   enter resume   d details   c copy command   q quit\n")
	return b.String()
}

func truncate(s string, max int) string {
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	if max <= 3 {
		return string(rs[:max])
	}
	return string(rs[:max-3]) + "..."
}

func relative(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return t.UTC().Format("Jan 2 15:04")
}
```

- [ ] **Step 6: Verify green and commit**

Run:

```bash
gofmt -w internal/picker
go test ./...
```

Expected: PASS.

Commit:

```bash
git add go.mod go.sum internal/picker
git commit -m "feat: add interactive picker model"
```

---

### Task 7: Runner, Clipboard, and tmux

**Files:**
- Create: `internal/clipboard/clipboard.go`
- Create: `internal/clipboard/clipboard_test.go`
- Create: `internal/runner/command.go`
- Create: `internal/runner/runner.go`
- Create: `internal/runner/tmux.go`
- Create: `internal/runner/runner_test.go`
- Test: `internal/clipboard/clipboard_test.go`
- Test: `internal/runner/runner_test.go`

- [ ] **Step 1: Write failing runner tests**

```go
// internal/runner/runner_test.go
package runner

import (
	"reflect"
	"testing"

	"resumer/internal/session"
)

func TestPlanDefaultRunUsesSessionCommand(t *testing.T) {
	exec := &FakeExecutor{}
	card := session.SessionCard{Harness: session.HarnessClaude, ID: "abc", ProjectPath: "/repo/app"}

	err := Run(card, Options{Mode: ModeExec}, exec)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if exec.Dir != "/repo/app" {
		t.Fatalf("Dir = %q", exec.Dir)
	}
	if !reflect.DeepEqual(exec.Argv, []string{"claude", "--resume", "abc"}) {
		t.Fatalf("Argv = %#v", exec.Argv)
	}
}

func TestPlanPrintDoesNotExecute(t *testing.T) {
	exec := &FakeExecutor{}
	card := session.SessionCard{Harness: session.HarnessCodex, ID: "abc", ProjectPath: "/repo/app"}
	var printed string

	err := Run(card, Options{Mode: ModePrint, Print: func(s string) { printed = s }}, exec)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if exec.Called {
		t.Fatal("executor called in print mode")
	}
	if printed != "codex resume abc --cd /repo/app" {
		t.Fatalf("printed = %q", printed)
	}
}

func TestTmuxSessionNameIsStable(t *testing.T) {
	card := session.SessionCard{Harness: session.HarnessClaude, Title: "Resumer CLI design!", ProjectPath: "/repo/openclaw fork"}
	if got := TmuxSessionName(card); got != "resumer-claude-openclaw-fork" {
		t.Fatalf("TmuxSessionName() = %q", got)
	}
}

type FakeExecutor struct {
	Called bool
	Argv   []string
	Dir    string
}

func (f *FakeExecutor) Exec(argv []string, dir string) error {
	f.Called = true
	f.Argv = append([]string(nil), argv...)
	f.Dir = dir
	return nil
}
```

- [ ] **Step 2: Run runner tests to verify red**

Run:

```bash
go test ./internal/runner
```

Expected: FAIL with `undefined: Run`.

- [ ] **Step 3: Implement runner command paths**

```go
// internal/runner/runner.go
package runner

import (
	"fmt"

	"resumer/internal/session"
)

type Mode string

const (
	ModeExec  Mode = "exec"
	ModePrint Mode = "print"
	ModeTmux  Mode = "tmux"
)

type Options struct {
	Mode  Mode
	Print func(string)
}

type Executor interface {
	Exec(argv []string, dir string) error
}

func Run(card session.SessionCard, opts Options, exec Executor) error {
	cmd := card.ResumeCommand()
	switch opts.Mode {
	case ModePrint:
		if opts.Print != nil {
			opts.Print(cmd.Display())
		}
		return nil
	case ModeExec, "":
		return exec.Exec(cmd.Argv, cmd.Dir)
	default:
		return fmt.Errorf("unknown runner mode %q", opts.Mode)
	}
}
```

```go
// internal/runner/tmux.go
package runner

import (
	"path/filepath"
	"regexp"
	"strings"

	"resumer/internal/session"
)

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func TmuxSessionName(card session.SessionCard) string {
	base := filepath.Base(card.ProjectPath)
	if base == "." || base == "/" || base == "" {
		base = card.DisplayTitle()
	}
	slug := strings.ToLower(base)
	slug = nonSlug.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "session"
	}
	return "resumer-" + strings.ToLower(string(card.Harness)) + "-" + slug
}
```

```go
// internal/runner/command.go
package runner
```

- [ ] **Step 4: Verify runner green**

Run:

```bash
gofmt -w internal/runner
go test ./internal/runner
```

Expected: PASS.

- [ ] **Step 5: Write failing clipboard tests**

```go
// internal/clipboard/clipboard_test.go
package clipboard

import "testing"

func TestCopyUsesConfiguredCommandRunner(t *testing.T) {
	var gotName string
	var gotInput string
	cb := Clipboard{
		Run: func(name string, input string) error {
			gotName = name
			gotInput = input
			return nil
		},
		CommandName: "pbcopy",
	}

	if err := cb.Copy("codex resume abc"); err != nil {
		t.Fatalf("Copy returned error: %v", err)
	}
	if gotName != "pbcopy" || gotInput != "codex resume abc" {
		t.Fatalf("runner got name=%q input=%q", gotName, gotInput)
	}
}

func TestCopyUnsupportedWhenNoCommand(t *testing.T) {
	cb := Clipboard{}
	if err := cb.Copy("anything"); err == nil {
		t.Fatal("Copy returned nil error without command")
	}
}
```

- [ ] **Step 6: Run clipboard tests to verify red**

Run:

```bash
go test ./internal/clipboard
```

Expected: FAIL with `undefined: Clipboard`.

- [ ] **Step 7: Implement clipboard adapter**

```go
// internal/clipboard/clipboard.go
package clipboard

import (
	"bytes"
	"errors"
	"os/exec"
)

type Clipboard struct {
	CommandName string
	Run         func(name string, input string) error
}

func Default() Clipboard {
	return Clipboard{CommandName: "pbcopy", Run: runCommand}
}

func (c Clipboard) Copy(text string) error {
	if c.CommandName == "" || c.Run == nil {
		return errors.New("clipboard unsupported")
	}
	return c.Run(c.CommandName, text)
}

func runCommand(name string, input string) error {
	cmd := exec.Command(name)
	cmd.Stdin = bytes.NewBufferString(input)
	return cmd.Run()
}
```

- [ ] **Step 8: Add tmux command construction tests**

```go
// Append to internal/runner/runner_test.go
func TestTmuxNewSessionCommandQuotesResumePayload(t *testing.T) {
	card := session.SessionCard{Harness: session.HarnessCodex, ID: "abc", ProjectPath: "/repo/space dir"}
	got := TmuxNewSessionArgv(card)
	want := []string{"tmux", "new-session", "-A", "-s", "resumer-codex-space-dir", "codex resume abc --cd '/repo/space dir'"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("TmuxNewSessionArgv() = %#v", got)
	}
}
```

Run:

```bash
go test ./internal/runner
```

Expected: FAIL with `undefined: TmuxNewSessionArgv`.

- [ ] **Step 9: Implement tmux argv construction**

```go
// Append to internal/runner/tmux.go
func TmuxNewSessionArgv(card session.SessionCard) []string {
	return []string{
		"tmux",
		"new-session",
		"-A",
		"-s",
		TmuxSessionName(card),
		card.ResumeCommand().Display(),
	}
}
```

- [ ] **Step 10: Verify green and commit**

Run:

```bash
gofmt -w internal/runner internal/clipboard
go test ./...
```

Expected: PASS.

Commit:

```bash
git add internal/runner internal/clipboard
git commit -m "feat: add resume runner and tmux support"
```

---

### Task 8: Command Orchestration, README, and Help Polish

**Files:**
- Modify: `internal/cmd/root.go`
- Modify: `internal/cmd/root_test.go`
- Modify: `cmd/resumer/main.go`
- Create: `README.md`
- Test: `internal/cmd/root_test.go`

- [ ] **Step 1: Write failing command orchestration tests**

```go
// Append to internal/cmd/root_test.go
func TestRunListJSONUsesDiscoverRankAndWriter(t *testing.T) {
	var wrote bool
	app := App{
		Discover: func(opts Options) ([]session.SessionCard, error) {
			return []session.SessionCard{{Harness: session.HarnessCodex, ID: "codex-1", Title: "Calc"}}, nil
		},
		WriteJSON: func(cards []session.SessionCard) error {
			wrote = true
			if len(cards) != 1 || cards[0].ID != "codex-1" {
				t.Fatalf("cards = %#v", cards)
			}
			return nil
		},
	}

	err := app.Run(Options{Mode: ModeListJSON, Limit: 50})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !wrote {
		t.Fatal("WriteJSON was not called")
	}
}

func TestRunInteractivePrintUsesPickerThenRunner(t *testing.T) {
	var ran bool
	app := App{
		Discover: func(opts Options) ([]session.SessionCard, error) {
			return []session.SessionCard{{Harness: session.HarnessClaude, ID: "claude-1"}}, nil
		},
		Pick: func(cards []session.SessionCard) (session.SessionCard, error) {
			return cards[0], nil
		},
		RunSelected: func(card session.SessionCard, opts Options) error {
			ran = true
			if card.ID != "claude-1" || !opts.Print {
				t.Fatalf("runner got card=%#v opts=%#v", card, opts)
			}
			return nil
		},
	}

	err := app.Run(Options{Mode: ModeInteractive, Print: true, Limit: 50})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !ran {
		t.Fatal("RunSelected was not called")
	}
}
```

Also add imports to `internal/cmd/root_test.go`:

```go
import (
	"testing"

	"resumer/internal/session"
)
```

- [ ] **Step 2: Run command tests to verify red**

Run:

```bash
go test ./internal/cmd
```

Expected: FAIL with `undefined: App`.

- [ ] **Step 3: Implement orchestration seam**

```go
// Replace the existing import block in internal/cmd/root.go with this block.
import (
	"github.com/alecthomas/kong"

	"resumer/internal/session"
)

// Append the App orchestration seam below ParseForTest in internal/cmd/root.go.

type App struct {
	Discover    func(Options) ([]session.SessionCard, error)
	WriteJSON   func([]session.SessionCard) error
	Pick        func([]session.SessionCard) (session.SessionCard, error)
	RunSelected func(session.SessionCard, Options) error
}

func (a App) Run(opts Options) error {
	if a.Discover == nil {
		return UsageError{Message: "discovery not configured"}
	}
	cards, err := a.Discover(opts)
	if err != nil {
		return err
	}
	if opts.Mode == ModeListJSON {
		if a.WriteJSON == nil {
			return UsageError{Message: "json writer not configured"}
		}
		return a.WriteJSON(cards)
	}
	if len(cards) == 0 {
		return UsageError{Message: "no sessions found"}
	}
	if a.Pick == nil || a.RunSelected == nil {
		return UsageError{Message: "interactive runner not configured"}
	}
	card, err := a.Pick(cards)
	if err != nil {
		return err
	}
	return a.RunSelected(card, opts)
}
```

- [ ] **Step 4: Verify command green**

Run:

```bash
gofmt -w internal/cmd
go test ./internal/cmd
```

Expected: PASS.

- [ ] **Step 5: Write failing `Main` entrypoint test**

```go
// Append to internal/cmd/root_test.go
func TestMainReturnsUsageCodeForBadArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Main([]string{"--limit", "0"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("Main code = %d, want %d", code, ExitUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "--limit must be greater than zero") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
```

Add these imports to `internal/cmd/root_test.go`:

```go
import (
	"bytes"
	"strings"
	"testing"

	"resumer/internal/session"
)
```

Run:

```bash
go test ./internal/cmd
```

Expected: FAIL with `undefined: Main`.

- [ ] **Step 6: Wire `Main` and executable entrypoint**

Update the import block in `internal/cmd/root.go` again, then add `Main` below `App.Run`:

```go
import (
	"fmt"
	"io"

	"github.com/alecthomas/kong"

	"resumer/internal/errfmt"
	"resumer/internal/session"
)

func Main(args []string, stdout io.Writer, stderr io.Writer) int {
	_, err := ParseForTest(args)
	if err != nil {
		fmt.Fprintln(stderr, errfmt.Human(err))
		return ExitCode(err)
	}
	return ExitOK
}
```

Update `cmd/resumer/main.go`:

```go
package main

import (
	"os"

	"resumer/internal/cmd"
)

func main() {
	os.Exit(cmd.Main(os.Args[1:], os.Stdout, os.Stderr))
}
```

Run:

```bash
go test ./...
```

Expected: tests PASS.

- [ ] **Step 7: Write README**

```markdown
<!-- README.md -->
# Resumer

Resumer is a small Go CLI for resuming Claude Code and Codex sessions without memorizing session IDs.

```bash
resumer
```

The default command opens an interactive terminal picker with recent Claude and Codex sessions. Rows emphasize the fields that help a person recognize the right session: harness, recency, project path, title, and first prompt preview.

## MVP Commands

```bash
resumer                 # interactive picker for Claude and Codex
resumer claude          # Claude sessions only
resumer codex           # Codex sessions only
resumer list --json     # machine-readable normalized session list
resumer --print         # print selected resume command instead of executing
resumer --tmux          # start or attach a Resumer tmux session
```

## Environment Overrides

```bash
RESUMER_CLAUDE_PROJECTS_PATH=~/.claude/projects
RESUMER_CODEX_SESSIONS_PATH=~/.codex/sessions
RESUMER_CODEX_INDEX_PATH=~/.codex/session_index.jsonl
RESUMER_DEFAULT_TMUX=1
RESUMER_TMUX_HOST_HINT=Ada-Mac-mini
```

## Safety

Resumer reads Claude and Codex session files. It does not mutate harness session state and does not maintain its own session database.

## tmux

`resumer --tmux` starts the selected resume command inside a stable session name like `resumer-claude-openclaw-fork`, then attaches to it. Resumer does not move already-running non-tmux processes into tmux.

## Deferred

Search, config commands, extra harnesses, OpenClaw integration, Homebrew packaging, and `resumer tmux` dashboard actions are outside the MVP.
```

- [ ] **Step 8: Add help-text assertions**

```go
// Append to internal/cmd/root_test.go
func TestHelpIncludesMVPFlags(t *testing.T) {
	_, err := ParseForTest([]string{"--help"})
	if err == nil {
		t.Fatal("ParseForTest --help returned nil error; kong should stop after help")
	}
}
```

Run:

```bash
go test ./internal/cmd
```

Expected: PASS or a Kong help-interruption error that is accepted by the test after adjusting the assertion to Kong's actual help behavior.

- [ ] **Step 9: Final verification and commit**

Run:

```bash
gofmt -w cmd internal
go test ./...
go run ./cmd/resumer --help
```

Expected: tests PASS; `go run ./cmd/resumer --help` exits cleanly and shows the MVP commands/flags.

Commit:

```bash
git add cmd internal README.md go.mod go.sum
git commit -m "feat: wire resumer command and docs"
```

---

## Self-Review Checklist

- [ ] R1 default interactive picker: Task 6 and Task 8.
- [ ] R2 harness filters: Task 1 and Task 8.
- [ ] R3 Claude index and JSONL fallback: Task 3.
- [ ] R4 Codex index and transcript enrichment: Task 4.
- [ ] R5 normalized session model: Task 2.
- [ ] R6 human-first card rendering: Task 6.
- [ ] R7 ranking, limit, sidechain filter, cwd bias: Task 5.
- [ ] R8 resume command generation: Task 2 and Task 7.
- [ ] R9 print/copy command: Task 6 and Task 7.
- [ ] R10 JSON output: Task 5 and Task 8.
- [ ] R11 tmux mode: Task 7.
- [ ] R12 read-only session state: Task 3, Task 4, Task 7.
- [ ] R13 stable exit/error behavior: Task 1 and Task 8.
- [ ] R14 MVP non-goals documented: Task 8.

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-26-resumer-cli.md`. Two execution options:

1. **Subagent-Driven (recommended)** - Dispatch a fresh subagent per task, review between tasks, fast iteration.
2. **Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints.
