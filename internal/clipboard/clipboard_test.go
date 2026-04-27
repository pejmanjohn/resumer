package clipboard

import (
	"errors"
	"testing"
)

func TestCopyUsesConfiguredCommandRunnerAndPassesTextUnchanged(t *testing.T) {
	var gotName, gotInput string
	clip := Clipboard{
		CommandName: "copy-tool",
		Run: func(name string, input string) error {
			gotName = name
			gotInput = input
			return nil
		},
	}

	if err := clip.Copy("first line\nsecond line"); err != nil {
		t.Fatalf("Copy() returned error: %v", err)
	}
	if gotName != "copy-tool" {
		t.Fatalf("runner name = %q, want copy-tool", gotName)
	}
	if gotInput != "first line\nsecond line" {
		t.Fatalf("runner input = %q, want text unchanged", gotInput)
	}
}

func TestCopyUnsupportedWhenNoCommandOrRunnerConfigured(t *testing.T) {
	tests := []struct {
		name string
		clip Clipboard
	}{
		{name: "empty", clip: Clipboard{}},
		{name: "missing command", clip: Clipboard{Run: func(string, string) error { return nil }}},
		{name: "missing runner", clip: Clipboard{CommandName: "copy-tool"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.clip.Copy("text"); err == nil {
				t.Fatalf("Copy() error = nil, want unsupported error")
			}
		})
	}
}

func TestCopyReturnsRunnerError(t *testing.T) {
	wantErr := errors.New("copy failed")
	clip := Clipboard{
		CommandName: "copy-tool",
		Run: func(string, string) error {
			return wantErr
		},
	}

	if err := clip.Copy("text"); !errors.Is(err, wantErr) {
		t.Fatalf("Copy() error = %v, want %v", err, wantErr)
	}
}

func TestDefaultHasPbcopyCommandAndRunner(t *testing.T) {
	clip := Default()

	if clip.CommandName != "pbcopy" {
		t.Fatalf("CommandName = %q, want pbcopy", clip.CommandName)
	}
	if clip.Run == nil {
		t.Fatalf("Run = nil, want configured runner")
	}
}
