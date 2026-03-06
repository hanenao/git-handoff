package cmd

import "github.com/spf13/cobra"

func newWorktreeCommand(options *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worktree",
		Short: "Manage AI worktrees",
	}
	cmd.AddCommand(newWorktreeCreateCommand(options))
	cmd.AddCommand(newWorktreeListCommand(options))
	cmd.AddCommand(newWorktreeRemoveCommand(options))
	return cmd
}
