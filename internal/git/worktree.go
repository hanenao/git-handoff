package git

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type Worktree struct {
	Path       string
	Head       string
	BranchRef  string
	BranchName string
	Detached   bool
}

func ListWorktrees(ctx context.Context, runner Runner, dir string) ([]Worktree, error) {
	result, err := runner.Run(ctx, dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return ParseWorktreeList(result.Stdout)
}

func ParseWorktreeList(raw string) ([]Worktree, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	blocks := strings.Split(strings.TrimSpace(raw), "\n\n")
	worktrees := make([]Worktree, 0, len(blocks))
	for _, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) == 0 {
			continue
		}

		var wt Worktree
		for _, line := range lines {
			switch {
			case strings.HasPrefix(line, "worktree "):
				wt.Path = strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
			case strings.HasPrefix(line, "HEAD "):
				wt.Head = strings.TrimSpace(strings.TrimPrefix(line, "HEAD "))
			case strings.HasPrefix(line, "branch "):
				wt.BranchRef = strings.TrimSpace(strings.TrimPrefix(line, "branch "))
				wt.BranchName = strings.TrimPrefix(wt.BranchRef, "refs/heads/")
			case line == "detached":
				wt.Detached = true
			}
		}

		if wt.Path == "" {
			return nil, fmt.Errorf("failed to parse worktree block: %q", block)
		}
		worktrees = append(worktrees, wt)
	}
	return worktrees, nil
}

func FindBranchOwner(worktrees []Worktree, branch string) *Worktree {
	for i := range worktrees {
		if worktrees[i].BranchName == branch {
			return &worktrees[i]
		}
	}
	return nil
}

func ResolveWorktreePath(base, raw string) (string, error) {
	abs, err := ResolvePath(base, raw)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}
