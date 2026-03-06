package cmd

import (
	"github.com/hanenao/git-handoff/internal/handoff"
	"github.com/spf13/cobra"
)

func newGoCommand(options *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "go <branch>",
		Short: "Print the path that currently owns the branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			repo, err := options.resolveRepo(ctx)
			if err != nil {
				return err
			}
			targetPath, err := handoff.NewService(options.runner).Go(ctx, repo, args[0])
			if err != nil {
				return err
			}
			return printLine(options.stdout, "%s", targetPath)
		},
	}
}
