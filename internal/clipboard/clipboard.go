package clipboard

import (
	"errors"
	"os/exec"
	"strings"
)

type Clipboard struct {
	CommandName string
	Run         func(name string, input string) error
}

func Default() Clipboard {
	return Clipboard{
		CommandName: "pbcopy",
		Run:         runCommand,
	}
}

func (c Clipboard) Copy(text string) error {
	if c.CommandName == "" || c.Run == nil {
		return errors.New("clipboard is unsupported")
	}
	return c.Run(c.CommandName, text)
}

func runCommand(name string, input string) error {
	cmd := exec.Command(name)
	cmd.Stdin = strings.NewReader(input)
	return cmd.Run()
}
