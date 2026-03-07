package main

import "github.com/spf13/cobra"

const kCmdDown = "down"

func NewDownCommand() *cobra.Command {
	return &cobra.Command{
		Use:   kCmdDown,
		Short: "Kill sessions and servers",
		Run:   func(cmd *cobra.Command, args []string) { initCmd(cmd, args).down() },
	}
}
