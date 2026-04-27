package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/alecthomas/kong"
	tea "github.com/charmbracelet/bubbletea"

	"resumer/internal/clipboard"
	"resumer/internal/config"
	"resumer/internal/discovery"
	"resumer/internal/errfmt"
	"resumer/internal/outfmt"
	"resumer/internal/picker"
	"resumer/internal/rank"
	"resumer/internal/runner"
	"resumer/internal/session"
)

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

func ParseOptions(args []string) (Options, error) {
	root := Root{}
	parser, err := kong.New(&root, kong.Name("resumer"), kong.Exit(func(int) {}))
	if err != nil {
		return Options{}, err
	}

	ctx, err := parser.Parse(args)
	if err != nil {
		return Options{}, UsageError{Message: err.Error()}
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
		if !root.List.JSON {
			return Options{}, UsageError{Message: "list requires --json"}
		}
		opts.Mode = ModeListJSON
	}

	return opts, nil
}

func ParseForTest(args []string) (Options, error) {
	return ParseOptions(args)
}

func Main(args []string, stdout io.Writer, stderr io.Writer) int {
	err := DefaultApp().Run(args, stdout, stderr)
	if err != nil {
		fmt.Fprintln(stderr, errfmt.Human(err))
		return ExitCode(err)
	}
	return ExitOK
}

type PickResult struct {
	Action   picker.Action
	Selected *session.SessionCard
}

type App struct {
	Discover    func(Options) ([]session.SessionCard, []discovery.Diagnostic, error)
	WriteJSON   func(io.Writer, []session.SessionCard) error
	Pick        func([]session.SessionCard) (PickResult, error)
	RunSelected func(session.SessionCard, runner.Mode) error
	CopyCommand func(string) error
	CWD         func() (string, error)
}

func DefaultApp() App {
	return App{
		Discover:  defaultDiscover,
		WriteJSON: outfmt.WriteJSON,
		Pick:      defaultPick,
		CWD:       os.Getwd,
	}
}

func (app App) Run(args []string, stdout io.Writer, stderr io.Writer) error {
	if isHelp(args) {
		return writeHelp(args, stdout)
	}

	opts, err := ParseOptions(args)
	if err != nil {
		return err
	}

	discover := app.Discover
	if discover == nil {
		discover = defaultDiscover
	}
	cards, diagnostics, err := discover(opts)
	if err != nil {
		return err
	}
	if opts.Debug {
		writeDiagnostics(stderr, diagnostics)
	}

	ranked := rank.Apply(cards, rankOptions(opts, app.CWD))
	if opts.Mode == ModeListJSON {
		writeJSON := app.WriteJSON
		if writeJSON == nil {
			writeJSON = outfmt.WriteJSON
		}
		return wrapLaunch(writeJSON(stdout, ranked))
	}

	if len(ranked) == 0 {
		return EmptyError{}
	}

	pick := app.Pick
	if pick == nil {
		pick = defaultPick
	}
	result, err := pick(ranked)
	if err != nil {
		return LaunchError{Message: err.Error()}
	}
	if result.Selected == nil {
		return CanceledError{}
	}

	switch result.Action {
	case picker.ActionResume:
		runSelected := app.RunSelected
		if runSelected == nil {
			executor := processExecutor{stdin: os.Stdin, stdout: stdout, stderr: stderr}
			runSelected = func(card session.SessionCard, mode runner.Mode) error {
				return runner.Run(card, runner.Options{
					Mode: mode,
					Print: func(line string) {
						fmt.Fprintln(stdout, line)
					},
				}, executor)
			}
		}
		return wrapLaunch(runSelected(*result.Selected, runnerMode(opts)))
	case picker.ActionCopy:
		copyCommand := app.CopyCommand
		if copyCommand == nil {
			clip := clipboard.Default()
			copyCommand = clip.Copy
		}
		return wrapLaunch(copyCommand(result.Selected.ResumeCommand().Display()))
	case picker.ActionCancel, picker.ActionNone:
		return CanceledError{}
	default:
		return CanceledError{}
	}
}

func defaultDiscover(opts Options) ([]session.SessionCard, []discovery.Diagnostic, error) {
	paths, err := config.LoadPaths()
	if err != nil {
		return nil, nil, err
	}

	var cards []session.SessionCard
	var diagnostics []discovery.Diagnostic
	if opts.Harness == HarnessAll || opts.Harness == HarnessClaude {
		discovered, diags := discovery.DiscoverClaude(discovery.ClaudeOptions{
			ProjectsPath: paths.ClaudeProjectsPath,
			IncludeAll:   opts.All,
		})
		cards = append(cards, discovered...)
		diagnostics = append(diagnostics, diags...)
	}
	if opts.Harness == HarnessAll || opts.Harness == HarnessCodex {
		discovered, diags := discovery.DiscoverCodex(discovery.CodexOptions{
			IndexPath:    paths.CodexIndexPath,
			SessionsPath: paths.CodexSessionsPath,
		})
		cards = append(cards, discovered...)
		diagnostics = append(diagnostics, diags...)
	}

	return cards, diagnostics, nil
}

func defaultPick(cards []session.SessionCard) (PickResult, error) {
	program := tea.NewProgram(picker.New(cards))
	model, err := program.Run()
	if err != nil {
		return PickResult{}, err
	}
	final, ok := model.(picker.Model)
	if !ok {
		return PickResult{Action: picker.ActionCancel}, nil
	}
	return PickResult{Action: final.Action, Selected: final.Selected}, nil
}

func rankOptions(opts Options, cwd func() (string, error)) rank.Options {
	var harness session.Harness
	switch opts.Harness {
	case HarnessClaude:
		harness = session.HarnessClaude
	case HarnessCodex:
		harness = session.HarnessCodex
	}

	var cwdBiasPath string
	if opts.CWDBias {
		if cwd == nil {
			cwd = os.Getwd
		}
		if value, err := cwd(); err == nil {
			cwdBiasPath = value
		}
	}

	return rank.Options{
		Harness:     harness,
		IncludeAll:  opts.All,
		CWDBiasPath: cwdBiasPath,
		Limit:       opts.Limit,
	}
}

func runnerMode(opts Options) runner.Mode {
	if opts.Print {
		return runner.ModePrint
	}
	if opts.Tmux {
		return runner.ModeTmux
	}
	return runner.ModeExec
}

func writeDiagnostics(w io.Writer, diagnostics []discovery.Diagnostic) {
	for _, diagnostic := range diagnostics {
		fmt.Fprintf(w, "%s: %s\n", diagnostic.Source, diagnostic.Message)
	}
}

func wrapLaunch(err error) error {
	if err == nil {
		return nil
	}
	return LaunchError{Message: err.Error()}
}

func isHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

type helpExit int

func writeHelp(args []string, w io.Writer) (err error) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		if code, ok := recovered.(helpExit); ok && code == 0 {
			err = nil
			return
		}
		panic(recovered)
	}()

	root := Root{}
	parser, err := kong.New(&root, kong.Name("resumer"), kong.Writers(w, io.Discard), kong.Exit(func(code int) {
		panic(helpExit(code))
	}))
	if err != nil {
		return err
	}
	if isTopLevelHelp(args) {
		fmt.Fprintln(w, "MVP commands:")
		fmt.Fprintln(w, "  resumer              Resume from Claude Code or Codex sessions.")
		fmt.Fprintln(w, "  resumer claude       Show Claude Code sessions only.")
		fmt.Fprintln(w, "  resumer codex        Show Codex sessions only.")
		fmt.Fprintln(w, "  resumer list --json  Emit stable JSON output for scripts.")
		fmt.Fprintln(w)
	}
	_, err = parser.Parse(args)
	return err
}

func isTopLevelHelp(args []string) bool {
	return len(args) == 1 && (args[0] == "--help" || args[0] == "-h")
}

type processExecutor struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func (e processExecutor) Exec(argv []string, dir string) error {
	if len(argv) == 0 {
		return fmt.Errorf("empty command")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Stdin = e.stdin
	cmd.Stdout = e.stdout
	cmd.Stderr = e.stderr
	return cmd.Run()
}
