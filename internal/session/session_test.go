package session

import (
	"reflect"
	"testing"
	"time"
)

func TestClaudeResumeCommandUsesProjectDirectory(t *testing.T) {
	card := SessionCard{
		Harness:     HarnessClaude,
		ID:          "claude-123",
		ProjectPath: "/repo/app",
	}

	cmd := card.ResumeCommand()

	wantArgv := []string{"claude", "--resume", "claude-123"}
	if !reflect.DeepEqual(cmd.Argv, wantArgv) {
		t.Fatalf("Argv = %#v, want %#v", cmd.Argv, wantArgv)
	}
	if cmd.Dir != "/repo/app" {
		t.Fatalf("Dir = %q, want /repo/app", cmd.Dir)
	}
	if got := cmd.Display(); got != "cd /repo/app && claude --resume claude-123" {
		t.Fatalf("Display() = %q", got)
	}
}

func TestCodexResumeCommandWithProjectPathUsesCDFlag(t *testing.T) {
	card := SessionCard{
		Harness:     HarnessCodex,
		ID:          "codex-123",
		ProjectPath: "/repo/app",
	}

	cmd := card.ResumeCommand()

	wantArgv := []string{"codex", "resume", "codex-123", "--cd", "/repo/app"}
	if !reflect.DeepEqual(cmd.Argv, wantArgv) {
		t.Fatalf("Argv = %#v, want %#v", cmd.Argv, wantArgv)
	}
	if cmd.Dir != "" {
		t.Fatalf("Dir = %q, want empty string", cmd.Dir)
	}
}

func TestCodexResumeCommandWithoutProjectPathOmitsCDFlag(t *testing.T) {
	card := SessionCard{
		Harness: HarnessCodex,
		ID:      "codex-123",
	}

	cmd := card.ResumeCommand()

	wantArgv := []string{"codex", "resume", "codex-123"}
	if !reflect.DeepEqual(cmd.Argv, wantArgv) {
		t.Fatalf("Argv = %#v, want %#v", cmd.Argv, wantArgv)
	}
}

func TestDisplayTitleFallsBackToFirstPromptThenSessionID(t *testing.T) {
	tests := []struct {
		name string
		card SessionCard
		want string
	}{
		{
			name: "trimmed title",
			card: SessionCard{Title: "  Fix bug  "},
			want: "Fix bug",
		},
		{
			name: "trimmed first prompt",
			card: SessionCard{FirstPrompt: "  Help me refactor  "},
			want: "Help me refactor",
		},
		{
			name: "last eight id chars",
			card: SessionCard{ID: "1234567890abcdef"},
			want: "session 90abcdef",
		},
		{
			name: "short id",
			card: SessionCard{ID: "abc"},
			want: "session abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.card.DisplayTitle(); got != tt.want {
				t.Fatalf("DisplayTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSortTimeFallsBackToCreatedAt(t *testing.T) {
	created := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)

	if got := (SessionCard{CreatedAt: created}).SortTime(); !got.Equal(created) {
		t.Fatalf("SortTime() = %v, want %v", got, created)
	}
	if got := (SessionCard{CreatedAt: created, UpdatedAt: updated}).SortTime(); !got.Equal(updated) {
		t.Fatalf("SortTime() = %v, want %v", got, updated)
	}
}

func TestResumeCommandDisplayQuotesPathsWithSpaces(t *testing.T) {
	cmd := ResumeCommand{
		Dir:  "/repo/app with spaces",
		Argv: []string{"claude", "--resume", "id with spaces"},
	}

	want := "cd '/repo/app with spaces' && claude --resume 'id with spaces'"
	if got := cmd.Display(); got != want {
		t.Fatalf("Display() = %q, want %q", got, want)
	}
}
