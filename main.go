package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

// version is stamped at release time via -ldflags "-X main.version=…".
var version = "dev"

func main() {
	// Scriptable usage: any arguments are delegated verbatim to the bash engine,
	// so `swoop use …`, `swoop update --all`, `-g`, etc. keep working.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Println("skillswoop " + version)
			return
		}
		os.Exit(passthrough(os.Args[1:]))
	}

	p := tea.NewProgram(newModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "swoop:", err)
		os.Exit(1)
	}
}

func passthrough(args []string) int {
	cmd := exec.Command(corePath(), args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "swoop: cannot run engine ("+corePath()+"):", err)
		return 127
	}
	return 0
}
