package runner

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"resumer/internal/session"
)

type recordingExecutor struct {
	calls []execCall
	err   error
}

type execCall struct {
	argv []string
	dir  string
}

func (e *recordingExecutor) Exec(argv []string, dir string) error {
	e.calls = append(e.calls, execCall{
		argv: append([]string(nil), argv...),
		dir:  dir,
	})
	return e.err
}

func TestRunDefaultExecUsesClaudeResumeCommandArgvAndDir(t *testing.T) {
	card := session.SessionCard{
		Harness:     session.HarnessClaude,
		ID:          "claude-session",
		ProjectPath: "/repo/app",
	}
	exec := &recordingExecutor{}

	if err := Run(card, Options{}, exec); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	want := []execCall{{
		argv: []string{"claude", "--resume", "claude-session"},
		dir:  "/repo/app",
	}}
	if !reflect.DeepEqual(exec.calls, want) {
		t.Fatalf("exec calls = %#v, want %#v", exec.calls, want)
	}
}

func TestRunPrintModeDoesNotExecuteAndPrintsCodexDisplayCommand(t *testing.T) {
	card := session.SessionCard{
		Harness:     session.HarnessCodex,
		ID:          "codex-session",
		ProjectPath: "/repo/app with spaces",
	}
	exec := &recordingExecutor{}
	var printed []string

	err := Run(card, Options{
		Mode:  ModePrint,
		Print: func(s string) { printed = append(printed, s) },
	}, exec)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if len(exec.calls) != 0 {
		t.Fatalf("exec calls = %#v, want none", exec.calls)
	}
	wantPrinted := []string{"codex resume codex-session --cd '/repo/app with spaces'"}
	if !reflect.DeepEqual(printed, wantPrinted) {
		t.Fatalf("printed = %#v, want %#v", printed, wantPrinted)
	}
}

func TestRunTmuxModeExecutesTmuxArgvWithEmptyDir(t *testing.T) {
	card := session.SessionCard{
		Harness:     session.HarnessCodex,
		ID:          "codex-session",
		ProjectPath: "/repo/app",
	}
	exec := &recordingExecutor{}

	if err := Run(card, Options{Mode: ModeTmux}, exec); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	want := []execCall{{
		argv: []string{"tmux", "new-session", "-A", "-s", "resumer-codex-app", "codex resume codex-session --cd /repo/app"},
		dir:  "",
	}}
	if !reflect.DeepEqual(exec.calls, want) {
		t.Fatalf("exec calls = %#v, want %#v", exec.calls, want)
	}
}

func TestTmuxSessionNameStableForProjectPathWithSpacesAndPunctuation(t *testing.T) {
	card := session.SessionCard{
		Harness:     session.HarnessCodex,
		ProjectPath: "/Users/ada/My App!!!",
		Title:       "Ignored Title",
	}

	if got := TmuxSessionName(card); got != "resumer-codex-my-app" {
		t.Fatalf("TmuxSessionName() = %q, want resumer-codex-my-app", got)
	}
}

func TestTmuxSessionNameFallsBackWhenDisplayTitleHasNoSlugCharacters(t *testing.T) {
	card := session.SessionCard{
		Harness: session.HarnessCodex,
		Title:   "!!!",
	}

	if got := TmuxSessionName(card); got != "resumer-codex-session" {
		t.Fatalf("TmuxSessionName() = %q, want resumer-codex-session", got)
	}
}

func TestTmuxNewSessionArgvQuotesResumePayloadForCodexProjectPathWithSpaces(t *testing.T) {
	card := session.SessionCard{
		Harness:     session.HarnessCodex,
		ID:          "codex-session",
		ProjectPath: "/repo/app with spaces",
	}

	got := TmuxNewSessionArgv(card)
	want := []string{"tmux", "new-session", "-A", "-s", "resumer-codex-app-with-spaces", "codex resume codex-session --cd '/repo/app with spaces'"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("TmuxNewSessionArgv() = %#v, want %#v", got, want)
	}
}

func TestRunUnknownModeReturnsError(t *testing.T) {
	card := session.SessionCard{
		Harness: session.HarnessClaude,
		ID:      "claude-session",
	}

	err := Run(card, Options{Mode: Mode("bogus")}, &recordingExecutor{})
	if err == nil || !strings.Contains(err.Error(), "unknown runner mode") {
		t.Fatalf("Run() error = %v, want unknown runner mode error", err)
	}
}

func TestRunInvalidEmptyResumeCommandReturnsErrorWithoutExecuting(t *testing.T) {
	exec := &recordingExecutor{}

	err := Run(session.SessionCard{Harness: session.Harness("unknown")}, Options{}, exec)
	if err == nil || !strings.Contains(err.Error(), "empty resume command") {
		t.Fatalf("Run() error = %v, want empty resume command error", err)
	}
	if len(exec.calls) != 0 {
		t.Fatalf("exec calls = %#v, want none", exec.calls)
	}
}

func TestRunReturnsExecutorError(t *testing.T) {
	wantErr := errors.New("exec failed")
	card := session.SessionCard{
		Harness: session.HarnessClaude,
		ID:      "claude-session",
	}

	err := Run(card, Options{}, &recordingExecutor{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
}
