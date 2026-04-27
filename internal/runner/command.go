package runner

type Mode string

const (
	ModeExec  Mode = "exec"
	ModePrint Mode = "print"
	ModeTmux  Mode = "tmux"
)

type Options struct {
	Mode  Mode
	Print func(string)
}

type Executor interface {
	Exec(argv []string, dir string) error
}
