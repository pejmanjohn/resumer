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
