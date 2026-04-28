package main

import (
	"os"

	"github.com/pejmanjohn/resumer/internal/cmd"
)

func main() {
	os.Exit(cmd.Main(os.Args[1:], os.Stdout, os.Stderr))
}
