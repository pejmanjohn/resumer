# Resumer — Design Sketch

## Goal

Build a small Go CLI that helps users resume existing Claude Code and Codex sessions without remembering session IDs or digging through hidden state directories.

The core experience:

```bash
resumer
```

opens a polished interactive picker showing recent Claude + Codex sessions with enough metadata to identify the right one, then launches the appropriate harness resume command.

This is human-first software. Machine-readable output is useful and should exist, but the default path should be a delightful terminal UI for a person trying to get back into flow.

## Inspiration

Use the `gogcli` style rather than forking it:

- Go CLI
- Kong-style command definitions
- clean `cmd/` + `internal/` package layout
- stable errors and exit codes
- human-friendly defaults
- optional JSON/plain output for scripts and agents

This should feel like a polished small tool, not a giant framework.

## Non-goals for MVP

- No daemon
- No mutation of Claude/Codex session files
- No custom session database
- No OpenClaw integration yet
- No support for every harness on day one
- No full config command system unless standard paths/env vars are not enough

## Supported Harnesses — MVP

### Claude Code

Primary discovery:

```text
~/.claude/projects/*/sessions-index.json
```

Useful fields:

- `sessionId`
- `summary`
- `firstPrompt`
- `messageCount`
- `created`
- `modified`
- `gitBranch`
- `projectPath`
- `fullPath`

Fallback discovery:

```text
~/.claude/projects/**/*.jsonl
```

Parse first and last messages if no index exists.

Resume command:

```bash
cd <projectPath> && claude --resume <sessionId>
```

If `projectPath` is missing:

```bash
claude --resume <sessionId>
```

### Codex

Primary discovery:

```text
~/.codex/session_index.jsonl
```

Observed fields:

- `id`
- `thread_name`
- `updated_at`

Resume command:

```bash
codex resume <sessionId>
```

Potential later enrichment:

- Codex SQLite database
- transcript files if discoverable
- cwd/project metadata when available

## Default Cross-Harness Metadata

Use the best fields that both Claude Code and Codex support by default. Keep the default card focused on human recognition, not exhaustive metadata.

```go
type SessionCard struct {
    Harness     string    // Claude | Codex
    ID          string    // hidden by default; shown in details/debug
    Title       string    // Claude summary/title, Codex thread_name
    ProjectPath string    // Claude projectPath/cwd, Codex session cwd
    UpdatedAt   time.Time // relative display: 14m ago
    CreatedAt   time.Time
    FirstPrompt string    // first real user prompt, filtered
    Model       string    // details/fallback display if useful
    SourcePath  string    // details/debug only
}
```

### Default menu priority

1. `Harness`
2. `UpdatedAt`
3. `ProjectPath`
4. `Title`
5. `FirstPrompt`

### Default row format

```text
❯  Claude   14m ago   ~/repos/openclaw-fork
   Resumer CLI design + Claude/Codex session discovery
   “Let’s plan and design this before building”

   Codex    2h ago    ~/Documents/Codex/2026-04-22-computer-use...
   Explore calculator app
   “Play around with calculator app”
```

### Intentionally excluded from the default card

- `GitBranch` — excellent for Claude, not reliable enough for Codex by default
- `Summary` as a separate field — uneven; title + prompt preview is sturdier
- `MessageCount` — mildly useful, not recognition-critical
- `LastUserPrompt` — useful later, but noisier and more expensive to extract cleanly
- `Source` — usually clutter; can live in details/debug

### Details pane fields

A details view can show the full normalized metadata:

```text
Claude session

ID:        446c1642-df4a-43d4-b544-4afa15816f66
Project:   ~/repos/openclaw-fork
Created:   Sun Apr 26, 10:42 AM
Modified:  Sun Apr 26, 12:21 PM
Model:     claude-opus-4-6
Source:    ~/.claude/projects/-Users-ada-repos-openclaw-fork/...

Resume command:
cd ~/repos/openclaw-fork && claude --resume 446c1642-df4a-43d4-b544-4afa15816f66
```

## Default UX

Run with no args:

```bash
resumer
```

Interactive picker, sorted by most recently modified:

```text
resumer

Resume a session

❯  Claude   14m ago   ~/repos/openclaw-fork
   Resumer CLI design + Claude/Codex session discovery
   “Let’s plan and design this before building”

   Codex    2h ago    ~/Documents/Codex/2026-04-22-computer-use...
   Explore calculator app
   “Play around with calculator app”

   Claude   yesterday ~/repos/pinchbot-ios
   Pinch relay reconnect race investigation
   “Background → foreground reconnect sometimes misses state…”

────────────────────────────────────────────────────────────
↑/↓ move   enter resume   d details   c copy command   q quit
```

Expected controls for MVP:

- up/down or `j`/`k` to navigate
- Enter to resume
- `d` to toggle details for the selected session
- `c` to copy the resume command
- `q`, Escape, or Ctrl-C to quit

Search can come later. It should not be central to MVP; good recency sorting plus strong card metadata should make the last ~20 sessions easy to scan.

## CLI Shape

```bash
resumer                         # interactive picker, all harnesses
resumer claude                  # interactive picker, Claude only
resumer codex                   # interactive picker, Codex only
resumer --tmux                  # launch selected resume command inside tmux
resumer tmux                    # dashboard of existing Resumer tmux sessions; enter attaches
resumer --limit 50
resumer --all                   # show everything, including old/noisy sessions
resumer --cwd                   # bias current working directory sessions higher
resumer --debug
```

Machine/script support:

```bash
resumer list --json             # machine-readable session list
resumer list --plain            # stable plain/TSV-ish output
resumer command <session-id>     # print the resume command for one session
resumer --print                 # after interactive selection, print instead of executing
resumer --copy                  # after interactive selection, copy command
```

Direct convenience paths:

```bash
resumer last
resumer claude last
resumer codex last
resumer search "pinch websocket" # later, not MVP
```

## TUI Implementation

Use Bubble Tea for the interactive terminal UI and Lip Gloss for styling.

Responsibilities:

- Bubble Tea handles arrow-key navigation, Enter, details toggle, copy action, resize events, and redraws
- Lip Gloss handles selected-row highlighting, dim/bright text, separators, truncation, padding, and color
- Write a custom picker model instead of starting with `bubbles/list`, because session cards are multi-line and need tight control

MVP key actions:

- `↑/↓` or `j/k` — move selection
- `enter` — resume selected session
- `d` — toggle details pane
- `c` — copy resume command
- `q`, `esc`, `ctrl+c` — quit

Later key actions:

- `/` — search/filter title, path, and first prompt
- `o` — open/reveal backing transcript file for debugging
- `t` — launch/attach selected session in tmux
- `f` — fork mode, if/when both harnesses support this cleanly

Minimal model:

```go
type pickerModel struct {
    sessions    []SessionCard
    cursor      int
    offset      int
    width       int
    height      int
    showDetails bool
    selected    *SessionCard
    copied      bool
    err         error
}
```

Key handling sketch:

```go
switch msg.String() {
case "up", "k":
    m.moveUp()
case "down", "j":
    m.moveDown()
case "enter":
    m.selected = &m.sessions[m.cursor]
    return m, tea.Quit
case "d":
    m.showDetails = !m.showDetails
case "c":
    m.copyCommand(m.sessions[m.cursor])
case "q", "esc", "ctrl+c":
    return m, tea.Quit
}
```

The footer is rendered every frame:

```text
↑/↓ move   enter resume   d details   c copy command   q quit
```

Later search can add:

```text
/ search   esc clear search
```

## Configuration

MVP should be auto-detect first, with environment variable overrides. Avoid a full `resumer config` subsystem until there is real pain.

Default locations:

```text
Claude projects: ~/.claude/projects
Codex sessions:  ~/.codex/sessions
Codex index:     ~/.codex/session_index.jsonl
```

Environment overrides:

```bash
RESUMER_CLAUDE_PROJECTS_PATH=~/.claude/projects
RESUMER_CODEX_SESSIONS_PATH=~/.codex/sessions
RESUMER_CODEX_INDEX_PATH=~/.codex/session_index.jsonl
RESUMER_DEFAULT_TMUX=1
RESUMER_TMUX_HOST_HINT=Ada-Mac-mini
```

Possible later config file:

```toml
[claude]
projects_path = "~/.claude/projects"

[codex]
sessions_path = "~/.codex/sessions"
index_path = "~/.codex/session_index.jsonl"

[ui]
limit = 50
cwd_bias = true

[tmux]
default = true
host_hint = "Ada-Mac-mini"
```

Possible later commands:

```bash
resumer config set claude.projects_path ~/.claude/projects
resumer config set codex.sessions_path ~/.codex/sessions
resumer config show
```

## Filtering and Ranking

Default behavior:

- show the most recent 50 sessions
- sort by `UpdatedAt` descending
- hide clearly internal/noisy sessions if reliably identifiable
- hide Claude sidechain sessions when `isSidechain` is true

`--all` behavior:

- include old sessions beyond the default limit
- include sidechain/internal sessions where available

`--cwd` behavior:

- do not strictly filter by current directory
- rank sessions from the current project/repo higher
- then fall back to recency

## Execution Model

Default execution after selection should replace the current process with the harness command on Unix via `syscall.Exec` where possible.

This makes the tool feel native: once a session is selected, the terminal belongs to Claude/Codex exactly as if the user had typed the resume command manually.

If exec replacement is not available, fall back to spawning the command with inherited stdin/stdout/stderr.

## tmux / mosh Mode

This is a major workflow feature for handing off coding-agent sessions from computer to phone over mosh.

Important constraint: Resumer should not promise to move an already-running non-tmux Claude/Codex process into tmux. A process must generally start inside tmux for tmux to own its PTY. Retrofitting an existing live process is brittle/hacky, especially on macOS.

What Resumer can do cleanly:

```bash
resumer --tmux
```

Flow:

1. User picks a session in the normal interactive picker
2. Resumer creates a sensible, stable tmux session name
3. Resumer starts the appropriate resume command inside tmux
4. Resumer attaches to that tmux session immediately
5. Later, the user can reconnect from phone via mosh and `tmux attach`

Example commands Resumer might run internally:

```bash
tmux new-session -s resumer-openclaw-fork \
  'cd ~/repos/openclaw-fork && claude --resume <session-id>'
```

```bash
tmux new-session -s resumer-codex-calculator \
  'codex resume <session-id> --cd ~/Documents/Codex/2026-04-22-computer-use-plugin-computer-use-openai'
```

If the selected session appears to already be running inside tmux, Resumer should prefer attach:

```text
This session appears to already be running in tmux.

enter  attach tmux session
r      resume separately anyway
esc    cancel
```

Detection can be best-effort:

- inspect existing tmux panes for `claude --resume <id>` or `codex resume <id>`
- inspect pane current paths where available
- match tmux session/window names created by Resumer

Naming strategy:

```text
resumer-<harness>-<project-or-title-slug>
```

Examples:

```text
resumer-claude-openclaw-fork
resumer-codex-emdashium-blog
```

Names should be stable and guessable from a phone, not random.

Collision strategy:

- if the tmux session exists, attach to it
- otherwise append a short suffix, or ask if the existing session should be reused

Suggested interactive footer when tmux support ships:

```text
↑/↓ move   enter resume   t tmux   d details   c copy command   q quit
```

### Phone reconnect instructions

After launching a tmux session, print a concise handoff note before/while attaching:

```text
Started tmux session: resumer-claude-openclaw-fork

Detach: ctrl-b then d
Reconnect from phone:
  mosh Ada-Mac-mini
  tmux attach -t resumer-claude-openclaw-fork
```

The exact host alias should be configurable later; for MVP the tmux attach command alone is enough.

### Default tmux mode

If phone handoff becomes the primary workflow, support default tmux mode via env var first:

```bash
RESUMER_DEFAULT_TMUX=1
```

Possible later config:

```toml
[tmux]
default = true
host_hint = "Ada-Mac-mini"
```

### Existing tmux dashboard

Add a dashboard command for Resumer-created tmux sessions:

```bash
resumer tmux
```

Example UI:

```text
resumer tmux

Attach to a Resumer tmux session

❯  resumer-claude-openclaw-fork    detached   updated 12m ago
   ~/repos/openclaw-fork
   Claude — Resumer CLI design + discovery

   resumer-codex-emdashium-blog    attached   updated 48m ago
   ~/repos/emdashium-blog
   Codex — Fix blog publishing MCP validation

────────────────────────────────────────────────────────────
↑/↓ move   enter attach   k kill   r rename   q quit
```

MVP version can be attach-only; kill/rename should be later because they are more destructive.

### Active session warning

The active/recent warning matters more in the phone handoff workflow:

```text
This session was updated 38 seconds ago and may still be running elsewhere.

If that terminal is accessible, attach to its tmux session instead.
Resuming separately could duplicate work or create confusing history.

enter  resume anyway
f      fork instead, if supported
esc    cancel
```

## Package Layout

```text
cmd/resumer/main.go
internal/cmd/root.go
internal/discovery/claude.go
internal/discovery/codex.go
internal/session/session.go
internal/picker/picker.go
internal/runner/runner.go
internal/outfmt/json.go
internal/outfmt/plain.go
internal/errfmt/errfmt.go
```

## MVP Cut

Ship only:

```bash
resumer
resumer claude
resumer codex
resumer list --json
resumer --print
```

MVP requirements:

- discovers Claude sessions from `sessions-index.json`, with JSONL fallback where needed
- discovers Codex sessions from `session_index.jsonl` plus transcript JSONL enrichment
- supports env var overrides for session/index locations
- sorts by recent activity
- optionally biases current working directory sessions higher
- presents a usable picker with details and copy-command actions
- launches correct resume command
- can optionally launch selected resume command inside tmux for mosh-friendly continuation
- can print JSON for scripts/agents without compromising the human-first default
- never mutates session state

## Later Ideas

- fuzzy search command
- cwd-aware ranking/filtering
- richer Codex metadata extraction
- `resumer tmux` kill/rename actions
- shell helper aliases, e.g. `r='resumer --tmux'` and `rt='resumer tmux'`
- shell completions
- Homebrew formula
- config file for non-standard paths
- support Gemini/Cursor/OpenCode/etc.
- OpenClaw ACP/session discovery
- terminal preview pane showing last user prompt / summary / git status

## Name

Chosen name: **Resumer**

Binary:

```bash
resumer
```

Why it works:

- built on the shared `resume` command used by both Claude and Codex
- clear without being cute
- easy to remember
- works as a project name and a binary name
- reads naturally in commands: `resumer claude`, `resumer codex`, `resumer list`
