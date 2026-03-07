package main

import "github.com/spf13/cobra"

const kCmdRestart = "restart"

func NewRestartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   kCmdRestart,
		Short: "Restart sessions",
		Run:   func(cmd *cobra.Command, args []string) { initCmd(cmd, args).restart() },
	}
}
