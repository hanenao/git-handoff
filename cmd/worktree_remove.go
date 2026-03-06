package cmd

import (
	"fmt"
	"path/filepath"

	ghgit "github.com/hanenao/git-handoff/internal/git"
	ghwt "github.com/hanenao/git-handoff/internal/worktree"
	"github.com/spf13/cobra"
)

func newWorktreeRemoveCommand(options *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <worktree-id>",
		Short: "Remove an idle worktree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			repo, err := options.resolveRepo(ctx)
			if err != nil {
				return err
			}
			manager := ghwt.NewManager(options.runner)
			target, _, err := manager.Resolve(ctx, repo, args[0])
			if err != nil {
				return err
			}
			if filepath.Clean(target.Path) == filepath.Clean(repo.CurrentWorktreePath) {
				return fmt.Errorf("current worktree cannot be removed")
			}
			if target.State != ghwt.StateIdle {
				return fmt.Errorf("worktree %s is attached; switch it back before removing", target.ID)
			}
			clean, err := ghgit.IsWorktreeClean(ctx, options.runner, target.Path)
			if err != nil {
				return err
			}
			if !clean {
				return fmt.Errorf("worktree %s has uncommitted changes; clean it before removing", target.ID)
			}
			if _, err := options.runner.Run(ctx, repo.CurrentWorktreePath, "worktree", "remove", target.Path); err != nil {
				return err
			}
			if err := ghwt.DeleteMetadata(repo.CommonDir, target.ID); err != nil {
				return err
			}
			return printLine(options.stdout, "removed worktree: %s", target.ID)
		},
	}
}
