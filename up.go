package main

import "github.com/spf13/cobra"

const kCmdUp = "up"

func NewUpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   kCmdUp,
		Short: "Create and start the server and sessions",
		Run: func(cmd *cobra.Command, args []string) {
			project := initCmd(cmd, args)
			project.up()
			if !cmd.Flag("detach").Changed {
				project.attach()
			}
		},
	}

	cmd.PersistentFlags().BoolP("detach", "d", false, "Detach from tmux session after starting it.")

	return cmd
}
