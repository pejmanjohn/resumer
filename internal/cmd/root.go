package cmd

import "github.com/alecthomas/kong"

type Mode string

const (
	ModeInteractive Mode = "interactive"
	ModeListJSON    Mode = "list-json"
)

type HarnessFilter string

const (
	HarnessAll    HarnessFilter = "all"
	HarnessClaude HarnessFilter = "claude"
	HarnessCodex  HarnessFilter = "codex"
)

type Options struct {
	Mode    Mode
	Harness HarnessFilter
	Limit   int
	All     bool
	CWDBias bool
	Debug   bool
	Print   bool
	Tmux    bool
}

type Root struct {
	Limit int  `default:"50" help:"Maximum sessions to show."`
	All   bool `help:"Include old, sidechain, and noisy sessions."`
	CWD   bool `name:"cwd" help:"Bias current working directory sessions higher."`
	Debug bool `help:"Print discovery diagnostics to stderr."`
	Print bool `help:"Print selected resume command instead of executing it."`
	Tmux  bool `help:"Launch selected resume command inside tmux."`

	Default HarnessCmd `cmd:"" default:"1" hidden:""`
	Claude  HarnessCmd `cmd:"" help:"Show Claude Code sessions only."`
	Codex   HarnessCmd `cmd:"" help:"Show Codex sessions only."`
	List    ListCmd    `cmd:"" help:"List sessions for scripts."`
}

type HarnessCmd struct{}

type ListCmd struct {
	JSON bool `name:"json" help:"Emit stable JSON output."`
}

func ParseForTest(args []string) (Options, error) {
	root := Root{}
	parser, err := kong.New(&root, kong.Name("resumer"), kong.Exit(func(int) {}))
	if err != nil {
		return Options{}, err
	}

	ctx, err := parser.Parse(args)
	if err != nil {
		return Options{}, err
	}
	if root.Limit < 1 {
		return Options{}, UsageError{Message: "--limit must be greater than zero"}
	}

	opts := Options{
		Mode:    ModeInteractive,
		Harness: HarnessAll,
		Limit:   root.Limit,
		All:     root.All,
		CWDBias: root.CWD,
		Debug:   root.Debug,
		Print:   root.Print,
		Tmux:    root.Tmux,
	}

	switch ctx.Command() {
	case "claude":
		opts.Harness = HarnessClaude
	case "codex":
		opts.Harness = HarnessCodex
	case "list":
		if root.List.JSON {
			opts.Mode = ModeListJSON
		}
	}

	return opts, nil
}
