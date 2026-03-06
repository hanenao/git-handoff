package cmd

import (
	"github.com/hanenao/git-handoff/internal/ui"
	ghwt "github.com/hanenao/git-handoff/internal/worktree"
	"github.com/spf13/cobra"
)

func newWorktreeListCommand(options *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List local and worktree states",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			repo, err := options.resolveRepo(ctx)
			if err != nil {
				return err
			}
			rows, err := ghwt.NewManager(options.runner).Rows(ctx, repo)
			if err != nil {
				return err
			}
			table, err := ui.RenderWorktreeTable(rows)
			if err != nil {
				return err
			}
			return ui.WriteString(options.stdout, table)
		},
	}
}
