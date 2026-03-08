package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	ghgit "github.com/hanenao/git-handoff/internal/git"
	ghwt "github.com/hanenao/git-handoff/internal/worktree"
	"github.com/spf13/cobra"
)

type completionItem struct {
	value       string
	description string
}

func completeBranchOwners(options *rootOptions) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		ctx := cmd.Context()
		repo, err := options.resolveRepo(ctx)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		branches, err := listLocalBranches(ctx, options.runner, repo.CurrentWorktreePath)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		worktrees, err := ghgit.ListWorktrees(ctx, options.runner, repo.CurrentWorktreePath)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		environments, _, err := ghwt.NewManager(options.runner).List(ctx, repo)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		worktreeByPath := make(map[string]ghwt.Environment, len(environments))
		for _, environment := range environments {
			resolvedPath, err := ghgit.EvalPath(environment.Path)
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			worktreeByPath[resolvedPath] = environment
		}

		items := make([]completionItem, 0, len(branches))
		for _, branch := range branches {
			description := "[branch]"
			if owner := ghgit.FindBranchOwner(worktrees, branch.name); owner != nil {
				ownerPath, err := ghgit.EvalPath(owner.Path)
				if err != nil {
					return nil, cobra.ShellCompDirectiveError
				}
				switch {
				case filepath.Clean(ownerPath) == filepath.Clean(repo.MainWorktreePath):
					description = "[local]"
				case worktreeByPath[ownerPath].ID != "":
					description = fmt.Sprintf("[worktree:%s]", worktreeByPath[ownerPath].ID)
				default:
					description = "[worktree]"
				}
			}
			if branch.subject != "" {
				description += " " + branch.subject
			}
			items = append(items, completionItem{
				value:       branch.name,
				description: description,
			})
		}
		return formatCompletionItems(items), cobra.ShellCompDirectiveNoFileComp
	}
}

func completeSwitchWorktrees(options *rootOptions) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		ctx := cmd.Context()
		repo, err := options.resolveRepo(ctx)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		current, _, err := ghwt.NewManager(options.runner).ResolveCurrentBackground(ctx, repo)
		if err == nil && filepath.Clean(current.Path) == filepath.Clean(repo.CurrentWorktreePath) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		return completeWorktreeIDs(cmd.Context(), options, func(environment ghwt.Environment) bool {
			return !environment.IsCurrent && environment.State == ghwt.StateIdle
		})
	}
}

func completeRemovableWorktrees(options *rootOptions) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completeWorktreeIDs(cmd.Context(), options, func(environment ghwt.Environment) bool {
			return !environment.IsCurrent && environment.State == ghwt.StateIdle
		})
	}
}

func completeWorktreeIDs(ctx context.Context, options *rootOptions, include func(ghwt.Environment) bool) ([]string, cobra.ShellCompDirective) {
	repo, err := options.resolveRepo(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	environments, _, err := ghwt.NewManager(options.runner).List(ctx, repo)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	items := make([]completionItem, 0, len(environments))
	for _, environment := range environments {
		if !include(environment) {
			continue
		}
		description := fmt.Sprintf("[%s] %s", environment.State, environment.Branch)
		items = append(items, completionItem{
			value:       environment.ID,
			description: description,
		})
	}
	return formatCompletionItems(items), cobra.ShellCompDirectiveNoFileComp
}

type localBranch struct {
	name    string
	subject string
}

func listLocalBranches(ctx context.Context, runner ghgit.Runner, dir string) ([]localBranch, error) {
	result, err := runner.Run(ctx, dir, "for-each-ref", "--sort=refname", "--format=%(refname:short)|%(contents:subject)", "refs/heads")
	if err != nil {
		return nil, err
	}
	if result.Stdout == "" {
		return nil, nil
	}

	lines := strings.Split(result.Stdout, "\n")
	branches := make([]localBranch, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		branch := localBranch{name: parts[0]}
		if len(parts) == 2 {
			branch.subject = strings.TrimSpace(parts[1])
		}
		branches = append(branches, branch)
	}
	return branches, nil
}

func formatCompletionItems(items []completionItem) []string {
	completions := make([]string, 0, len(items))
	for _, item := range items {
		if item.description == "" {
			completions = append(completions, item.value)
			continue
		}
		completions = append(completions, fmt.Sprintf("%s\t%s", item.value, item.description))
	}
	return completions
}
