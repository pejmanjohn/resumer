# Resumer

Resumer is a small Go CLI for finding recent Claude Code and Codex sessions and resuming one without digging through hidden state directories or copying opaque IDs by hand.

## MVP Commands

```bash
resumer                         # interactive picker across Claude Code and Codex
resumer claude                  # interactive picker for Claude Code sessions
resumer codex                   # interactive picker for Codex sessions
resumer list --json             # machine-readable session list
resumer --print                 # pick a session, then print the resume command
resumer --tmux                  # pick a session, then run it through tmux
```

Useful flags:

```bash
--limit N    Maximum sessions to show, default 50
--all        Include old, sidechain, and noisy sessions
--cwd        Bias sessions under the current working directory
--debug      Print discovery diagnostics to stderr
--print      Print the selected command instead of executing it
--tmux       Launch the selected command in a Resumer-named tmux session
```

In the picker, use `up`/`down` or `j`/`k` to move, Enter to resume, `d` for details, `c` to copy the resume command, and `q`, Escape, or Ctrl-C to cancel.

## Output And Safety

The default mode is interactive and human-focused. `resumer list --json` is the scriptable mode and writes only JSON to stdout:

```json
{
  "sessions": []
}
```

Diagnostics and human errors go to stderr so JSON stdout stays clean.

Resumer is read-only with respect to Claude Code and Codex session storage. It discovers sessions from existing harness files, ranks normalized session cards in memory, and then either runs, prints, or copies the harness resume command. It does not mutate transcript files, create a session database, or rewrite harness state.

Normal resume execution uses direct argv execution through `os/exec`, not a shell. The displayed and copied command is shell-formatted for humans.

## Discovery Paths

By default, Resumer looks in the standard harness locations:

```text
~/.claude/projects
~/.codex/session_index.jsonl
~/.codex/sessions
```

Environment overrides:

```bash
RESUMER_CLAUDE_PROJECTS_PATH=/path/to/claude/projects
RESUMER_CODEX_INDEX_PATH=/path/to/session_index.jsonl
RESUMER_CODEX_SESSIONS_PATH=/path/to/codex/sessions
```

The config package also recognizes `RESUMER_DEFAULT_TMUX` and `RESUMER_TMUX_HOST_HINT` for the planned tmux defaults/remote-host workflow, but the MVP command path is explicit: use `--tmux` when you want tmux launch behavior.

## Tmux

`resumer --tmux` runs the selected resume command through:

```bash
tmux new-session -A -s <resumer-session-name> <resume-command>
```

The session name is derived from the harness and project/title, so repeated launches attach/reuse the same Resumer-named tmux session instead of creating a new one. `--print` takes precedence over `--tmux` when both flags are present.

Resumer does not move an already-running non-tmux Claude or Codex process into tmux.

## Deferred

The MVP intentionally does not include:

- fuzzy search or `resumer search`
- a `resumer config` command system or config file
- plain/TSV output or `resumer command <session-id>`
- support for Gemini, Cursor, OpenCode, OpenClaw, or other harnesses
- Codex metadata sources beyond the current index and transcript JSONL enrichment
- a `resumer tmux` dashboard, kill, or rename actions
- Homebrew packaging or release installers

## Development

```bash
go test ./...
go run ./cmd/resumer --help
```
