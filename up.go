package main

import "github.com/spf13/cobra"

const kCmdUp = "up"

func NewUpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   kCmdUp,
		Short: "Create and start the server and sessions",
		Run:   func(cmd *cobra.Command, args []string) { initCmd(cmd, args).up() },
	}
}
