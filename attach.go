package main

import "github.com/spf13/cobra"

const kCmdAttach = "attach"

func NewAttachCommand() *cobra.Command {
	return &cobra.Command{
		Use:   kCmdAttach,
		Short: "Attach to the tmux server",
		Run:   func(cmd *cobra.Command, args []string) { initCmd(cmd, args).attach() },
	}
}
