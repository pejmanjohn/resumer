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
