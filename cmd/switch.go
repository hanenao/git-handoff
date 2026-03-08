package cmd

import (
	"github.com/hanenao/git-handoff/internal/handoff"
	"github.com/spf13/cobra"
)

func newSwitchCommand(options *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:               "switch [worktree-id]",
		Short:             "Hand off the current branch between local and worktree",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeSwitchWorktrees(options),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			repo, err := options.resolveRepo(ctx)
			if err != nil {
				return err
			}
			cfg, err := options.resolveConfig(ctx, cmd, repo)
			if err != nil {
				return err
			}
			worktreeID := ""
			if len(args) == 1 {
				worktreeID = args[0]
			}
			targetPath, err := handoff.NewService(options.runner).Switch(ctx, repo, worktreeID, cfg)
			if err != nil {
				return err
			}
			return printLine(options.stdout, "%s", targetPath)
		},
	}
}
