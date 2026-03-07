package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func getDefaultShell() string {
	sh := os.Getenv("SHELL")

	if sh == "" {
		return "/bin/sh -c"
	}

	return sh + " -c"
}

func main() {
	cmd := cobra.Command{
		Use: filepath.Base(os.Args[0]),
	}

	cmd.PersistentFlags().StringP("file", "f", "tmux-compose.yml", "Specify an alternate compose file")
	cmd.PersistentFlags().String("shell", getDefaultShell(), "Specify an alternate shell path")

	cmd.AddCommand(NewUpCommand())
	cmd.AddCommand(NewDownCommand())
	cmd.AddCommand(NewRestartCommand())
	cmd.AddCommand(NewAttachCommand())
	cmd.Execute()
}
