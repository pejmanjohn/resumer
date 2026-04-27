package cmd

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"resumer/internal/discovery"
	"resumer/internal/picker"
	"resumer/internal/runner"
	"resumer/internal/session"
)

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

func TestParseInvalidFlagReturnsUsageError(t *testing.T) {
	_, err := ParseForTest([]string{"--definitely-invalid"})
	if err == nil {
		t.Fatal("ParseForTest returned nil error for invalid flag")
	}
	if got := ExitCode(err); got != ExitUsage {
		t.Fatalf("ExitCode(ParseForTest error) = %d, want %d", got, ExitUsage)
	}
}

func TestParseBareListReturnsUsageError(t *testing.T) {
	_, err := ParseForTest([]string{"list"})
	if err == nil {
		t.Fatal("ParseForTest returned nil error for bare list")
	}
	if got := ExitCode(err); got != ExitUsage {
		t.Fatalf("ExitCode(ParseForTest error) = %d, want %d", got, ExitUsage)
	}
}

func TestMainInvalidFlagReturnsUsageAndWritesHumanError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	got := Main([]string{"--definitely-invalid"}, &stdout, &stderr)

	if got != ExitUsage {
		t.Fatalf("Main() = %d, want %d", got, ExitUsage)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty string", stdout.String())
	}
	if !strings.Contains(stderr.String(), "resumer:") {
		t.Fatalf("stderr = %q, want human error", stderr.String())
	}
}

func TestMainBareListReturnsUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	got := Main([]string{"list"}, &stdout, &stderr)

	if got != ExitUsage {
		t.Fatalf("Main() = %d, want %d", got, ExitUsage)
	}
}

func TestAppRunListJSONRanksFiltersAndAllowsEmpty(t *testing.T) {
	var wrote []session.SessionCard
	writeCalled := false
	app := testApp(t)
	app.Discover = func(Options) ([]session.SessionCard, []discovery.Diagnostic, error) {
		return []session.SessionCard{
			testCard("old", session.HarnessClaude, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
			testCard("new", session.HarnessClaude, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)),
			testSidechainCard("sidechain", session.HarnessCodex, time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)),
		}, nil, nil
	}
	app.WriteJSON = func(_ io.Writer, cards []session.SessionCard) error {
		writeCalled = true
		wrote = append([]session.SessionCard(nil), cards...)
		return nil
	}

	var stdout bytes.Buffer
	if err := app.Run([]string{"--limit", "1", "list", "--json"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(wrote) != 1 || wrote[0].ID != "new" {
		t.Fatalf("WriteJSON cards = %#v, want newest non-sidechain card only", wrote)
	}

	app.Discover = func(Options) ([]session.SessionCard, []discovery.Diagnostic, error) {
		return nil, nil, nil
	}
	wrote = nil
	writeCalled = false
	if err := app.Run([]string{"list", "--json"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run returned error for empty JSON list: %v", err)
	}
	if !writeCalled || len(wrote) != 0 {
		t.Fatalf("WriteJSON empty cards = %#v, want empty slice", wrote)
	}
}

func TestAppRunInteractivePrintRunsSelectedWithPrintMode(t *testing.T) {
	card := testCard("claude-1", session.HarnessClaude, time.Now())
	app := testAppWithCards(t, card)
	app.Pick = func(cards []session.SessionCard) (PickResult, error) {
		if len(cards) != 1 || cards[0].ID != card.ID {
			t.Fatalf("Pick cards = %#v, want selected card", cards)
		}
		return PickResult{Action: picker.ActionResume, Selected: &cards[0]}, nil
	}
	var gotMode runner.Mode
	var gotCard session.SessionCard
	app.RunSelected = func(card session.SessionCard, mode runner.Mode) error {
		gotCard = card
		gotMode = mode
		return nil
	}

	if err := app.Run([]string{"--print"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if gotCard.ID != card.ID || gotMode != runner.ModePrint {
		t.Fatalf("RunSelected got (%s, %s), want (%s, %s)", gotCard.ID, gotMode, card.ID, runner.ModePrint)
	}
}

func TestAppRunInteractiveTmuxModeAndPrintWins(t *testing.T) {
	card := testCard("codex-1", session.HarnessCodex, time.Now())
	tests := []struct {
		name string
		args []string
		want runner.Mode
	}{
		{name: "tmux", args: []string{"--tmux"}, want: runner.ModeTmux},
		{name: "print wins", args: []string{"--tmux", "--print"}, want: runner.ModePrint},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := testAppWithCards(t, card)
			app.Pick = func(cards []session.SessionCard) (PickResult, error) {
				return PickResult{Action: picker.ActionResume, Selected: &cards[0]}, nil
			}
			var got runner.Mode
			app.RunSelected = func(_ session.SessionCard, mode runner.Mode) error {
				got = mode
				return nil
			}

			if err := app.Run(tt.args, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
				t.Fatalf("Run returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("RunSelected mode = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestAppRunCopyActionCopiesCommandAndDoesNotRun(t *testing.T) {
	card := testCard("claude-1", session.HarnessClaude, time.Now())
	app := testAppWithCards(t, card)
	app.Pick = func(cards []session.SessionCard) (PickResult, error) {
		return PickResult{Action: picker.ActionCopy, Selected: &cards[0]}, nil
	}
	var copied string
	app.CopyCommand = func(command string) error {
		copied = command
		return nil
	}
	app.RunSelected = func(session.SessionCard, runner.Mode) error {
		t.Fatal("RunSelected should not be called for copy action")
		return nil
	}

	if err := app.Run(nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if copied != card.ResumeCommand().Display() {
		t.Fatalf("copied = %q, want %q", copied, card.ResumeCommand().Display())
	}
}

func TestAppRunCancelOrNoSelectionReturnsCanceledError(t *testing.T) {
	card := testCard("claude-1", session.HarnessClaude, time.Now())
	tests := []struct {
		name   string
		result PickResult
	}{
		{name: "cancel", result: PickResult{Action: picker.ActionCancel, Selected: &card}},
		{name: "none", result: PickResult{Action: picker.ActionNone, Selected: &card}},
		{name: "no selected", result: PickResult{Action: picker.ActionResume}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := testAppWithCards(t, card)
			app.Pick = func([]session.SessionCard) (PickResult, error) {
				return tt.result, nil
			}
			err := app.Run(nil, &bytes.Buffer{}, &bytes.Buffer{})
			var canceled CanceledError
			if !errors.As(err, &canceled) {
				t.Fatalf("Run error = %v, want CanceledError", err)
			}
		})
	}
}

func TestAppRunEmptyInteractiveReturnsEmptyError(t *testing.T) {
	app := testApp(t)
	app.Discover = func(Options) ([]session.SessionCard, []discovery.Diagnostic, error) {
		return nil, nil, nil
	}

	err := app.Run(nil, &bytes.Buffer{}, &bytes.Buffer{})
	var empty EmptyError
	if !errors.As(err, &empty) {
		t.Fatalf("Run error = %v, want EmptyError", err)
	}
}

func TestAppRunDebugDiagnosticsOnlyWhenDebugEnabled(t *testing.T) {
	app := testApp(t)
	app.Discover = func(Options) ([]session.SessionCard, []discovery.Diagnostic, error) {
		return []session.SessionCard{testCard("claude-1", session.HarnessClaude, time.Now())},
			[]discovery.Diagnostic{{Source: "index", Message: "bad line"}}, nil
	}
	app.Pick = func(cards []session.SessionCard) (PickResult, error) {
		return PickResult{Action: picker.ActionCancel, Selected: &cards[0]}, nil
	}

	var stderr bytes.Buffer
	_ = app.Run(nil, &bytes.Buffer{}, &stderr)
	if stderr.String() != "" {
		t.Fatalf("stderr without debug = %q, want empty", stderr.String())
	}

	stderr.Reset()
	_ = app.Run([]string{"--debug"}, &bytes.Buffer{}, &stderr)
	if !strings.Contains(stderr.String(), "index: bad line") {
		t.Fatalf("stderr with debug = %q, want diagnostic", stderr.String())
	}
}

func TestMainHelpReturnsOKAndWritesMVPHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	got := Main([]string{"--help"}, &stdout, &stderr)

	if got != ExitOK {
		t.Fatalf("Main(--help) = %d, want %d", got, ExitOK)
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	help := stdout.String()
	for _, want := range []string{"claude", "codex", "list", "--json", "--print", "--tmux"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help missing %q:\n%s", want, help)
		}
	}
}

func testApp(t *testing.T) App {
	t.Helper()
	return App{
		Discover: func(Options) ([]session.SessionCard, []discovery.Diagnostic, error) {
			t.Fatal("Discover was not configured")
			return nil, nil, nil
		},
		WriteJSON: func(_ io.Writer, _ []session.SessionCard) error {
			t.Fatal("WriteJSON should not be called")
			return nil
		},
		Pick: func([]session.SessionCard) (PickResult, error) {
			t.Fatal("Pick should not be called")
			return PickResult{}, nil
		},
		RunSelected: func(session.SessionCard, runner.Mode) error {
			t.Fatal("RunSelected should not be called")
			return nil
		},
		CopyCommand: func(string) error {
			t.Fatal("CopyCommand should not be called")
			return nil
		},
		CWD: func() (string, error) { return "", nil },
	}
}

func testAppWithCards(t *testing.T, cards ...session.SessionCard) App {
	t.Helper()
	app := testApp(t)
	app.Discover = func(Options) ([]session.SessionCard, []discovery.Diagnostic, error) {
		return cards, nil, nil
	}
	return app
}

func testCard(id string, harness session.Harness, updatedAt time.Time) session.SessionCard {
	return session.SessionCard{
		Harness:     harness,
		ID:          id,
		Title:       id,
		ProjectPath: "/tmp/" + id,
		UpdatedAt:   updatedAt,
	}
}

func testSidechainCard(id string, harness session.Harness, updatedAt time.Time) session.SessionCard {
	card := testCard(id, harness, updatedAt)
	card.Sidechain = true
	return card
}
