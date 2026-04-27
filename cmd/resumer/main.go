package main

import (
	"os"

	"resumer/internal/cmd"
)

func main() {
	os.Exit(cmd.Main(os.Args[1:], os.Stdout, os.Stderr))
}
