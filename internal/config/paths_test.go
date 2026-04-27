package config

import (
	"path/filepath"
	"testing"
)

func TestLoadPathsDefaultsUnderHome(t *testing.T) {
	t.Setenv("HOME", "/home/ada")
	clearResumerEnv(t)

	paths, err := LoadPaths()
	if err != nil {
		t.Fatalf("LoadPaths() returned error: %v", err)
	}

	if paths.ClaudeProjectsPath != filepath.FromSlash("/home/ada/.claude/projects") {
		t.Fatalf("ClaudeProjectsPath = %q", paths.ClaudeProjectsPath)
	}
	if paths.CodexSessionsPath != filepath.FromSlash("/home/ada/.codex/sessions") {
		t.Fatalf("CodexSessionsPath = %q", paths.CodexSessionsPath)
	}
	if paths.CodexIndexPath != filepath.FromSlash("/home/ada/.codex/session_index.jsonl") {
		t.Fatalf("CodexIndexPath = %q", paths.CodexIndexPath)
	}
	if paths.DefaultTmux {
		t.Fatal("DefaultTmux = true, want false")
	}
	if paths.TmuxHostHint != "" {
		t.Fatalf("TmuxHostHint = %q, want empty string", paths.TmuxHostHint)
	}
}

func TestLoadPathsUsesOverrides(t *testing.T) {
	t.Setenv("HOME", "/home/ada")
	t.Setenv("RESUMER_CLAUDE_PROJECTS_PATH", "/tmp/claude")
	t.Setenv("RESUMER_CODEX_SESSIONS_PATH", "/tmp/codex/sessions")
	t.Setenv("RESUMER_CODEX_INDEX_PATH", "/tmp/codex/index.jsonl")
	t.Setenv("RESUMER_DEFAULT_TMUX", "true")
	t.Setenv("RESUMER_TMUX_HOST_HINT", "work")

	paths, err := LoadPaths()
	if err != nil {
		t.Fatalf("LoadPaths() returned error: %v", err)
	}

	if paths.ClaudeProjectsPath != "/tmp/claude" {
		t.Fatalf("ClaudeProjectsPath = %q", paths.ClaudeProjectsPath)
	}
	if paths.CodexSessionsPath != "/tmp/codex/sessions" {
		t.Fatalf("CodexSessionsPath = %q", paths.CodexSessionsPath)
	}
	if paths.CodexIndexPath != "/tmp/codex/index.jsonl" {
		t.Fatalf("CodexIndexPath = %q", paths.CodexIndexPath)
	}
	if !paths.DefaultTmux {
		t.Fatal("DefaultTmux = false, want true")
	}
	if paths.TmuxHostHint != "work" {
		t.Fatalf("TmuxHostHint = %q, want work", paths.TmuxHostHint)
	}
}

func TestLoadPathsReturnsErrorForBadDefaultTmux(t *testing.T) {
	t.Setenv("HOME", "/home/ada")
	clearResumerEnv(t)
	t.Setenv("RESUMER_DEFAULT_TMUX", "definitely")

	if _, err := LoadPaths(); err == nil {
		t.Fatal("LoadPaths() returned nil error for bad RESUMER_DEFAULT_TMUX")
	}
}

func TestLoadPathsDefaultsAfterClearingResumerEnv(t *testing.T) {
	t.Setenv("HOME", "/home/ada")
	t.Setenv("RESUMER_CLAUDE_PROJECTS_PATH", "/tmp/claude")
	t.Setenv("RESUMER_CODEX_SESSIONS_PATH", "/tmp/codex/sessions")
	t.Setenv("RESUMER_CODEX_INDEX_PATH", "/tmp/codex/index.jsonl")
	t.Setenv("RESUMER_DEFAULT_TMUX", "true")
	t.Setenv("RESUMER_TMUX_HOST_HINT", "work")
	clearResumerEnv(t)

	paths, err := LoadPaths()
	if err != nil {
		t.Fatalf("LoadPaths() returned error: %v", err)
	}

	if paths.ClaudeProjectsPath != filepath.FromSlash("/home/ada/.claude/projects") {
		t.Fatalf("ClaudeProjectsPath = %q", paths.ClaudeProjectsPath)
	}
	if paths.CodexSessionsPath != filepath.FromSlash("/home/ada/.codex/sessions") {
		t.Fatalf("CodexSessionsPath = %q", paths.CodexSessionsPath)
	}
	if paths.CodexIndexPath != filepath.FromSlash("/home/ada/.codex/session_index.jsonl") {
		t.Fatalf("CodexIndexPath = %q", paths.CodexIndexPath)
	}
	if paths.DefaultTmux {
		t.Fatal("DefaultTmux = true, want false")
	}
	if paths.TmuxHostHint != "" {
		t.Fatalf("TmuxHostHint = %q, want empty string", paths.TmuxHostHint)
	}
}

func TestLoadPathsIgnoresEmptyOverrides(t *testing.T) {
	t.Setenv("HOME", "/home/ada")
	t.Setenv("RESUMER_CLAUDE_PROJECTS_PATH", "")
	t.Setenv("RESUMER_CODEX_SESSIONS_PATH", "")
	t.Setenv("RESUMER_CODEX_INDEX_PATH", "")
	t.Setenv("RESUMER_DEFAULT_TMUX", "")
	t.Setenv("RESUMER_TMUX_HOST_HINT", "")

	paths, err := LoadPaths()
	if err != nil {
		t.Fatalf("LoadPaths() returned error: %v", err)
	}

	if paths.ClaudeProjectsPath != filepath.FromSlash("/home/ada/.claude/projects") {
		t.Fatalf("ClaudeProjectsPath = %q", paths.ClaudeProjectsPath)
	}
	if paths.CodexSessionsPath != filepath.FromSlash("/home/ada/.codex/sessions") {
		t.Fatalf("CodexSessionsPath = %q", paths.CodexSessionsPath)
	}
	if paths.CodexIndexPath != filepath.FromSlash("/home/ada/.codex/session_index.jsonl") {
		t.Fatalf("CodexIndexPath = %q", paths.CodexIndexPath)
	}
	if paths.DefaultTmux {
		t.Fatal("DefaultTmux = true, want false")
	}
	if paths.TmuxHostHint != "" {
		t.Fatalf("TmuxHostHint = %q, want empty string", paths.TmuxHostHint)
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
