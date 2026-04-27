package runner

import (
	"errors"
	"fmt"

	"resumer/internal/session"
)

func Run(card session.SessionCard, opts Options, exec Executor) error {
	cmd := card.ResumeCommand()
	if len(cmd.Argv) == 0 {
		return errors.New("empty resume command")
	}

	switch opts.Mode {
	case "", ModeExec:
		if exec == nil {
			return errors.New("runner executor is missing")
		}
		return exec.Exec(cmd.Argv, cmd.Dir)
	case ModePrint:
		if opts.Print == nil {
			return errors.New("runner print function is missing")
		}
		opts.Print(cmd.Display())
		return nil
	case ModeTmux:
		if exec == nil {
			return errors.New("runner executor is missing")
		}
		return exec.Exec(TmuxNewSessionArgv(card), "")
	default:
		return fmt.Errorf("unknown runner mode %q", opts.Mode)
	}
}
