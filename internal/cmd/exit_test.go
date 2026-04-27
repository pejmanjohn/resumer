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
		{name: "empty", err: EmptyError{}, want: ExitEmpty},
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
