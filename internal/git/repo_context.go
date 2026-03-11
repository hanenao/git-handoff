package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type RepoContext struct {
	Root                string
	CommonDir           string
	GitDir              string
	MainWorktreePath    string
	CurrentWorktreePath string
	IsMainWorktree      bool
	IsLinkedWorktree    bool
}

func ResolveRepoContext(ctx context.Context, runner Runner, cwd string) (*RepoContext, error) {
	rootResult, err := runner.Run(ctx, cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("git repository root could not be resolved: %w", err)
	}
	root, err := ResolvePath(cwd, rootResult.Stdout)
	if err != nil {
		return nil, err
	}
	root, err = EvalPath(root)
	if err != nil {
		return nil, err
	}

	commonDirResult, err := runner.Run(ctx, cwd, "rev-parse", "--git-common-dir")
	if err != nil {
		return nil, fmt.Errorf("git common dir could not be resolved: %w", err)
	}
	commonDir, err := ResolvePath(root, commonDirResult.Stdout)
	if err != nil {
		return nil, err
	}
	commonDir, err = EvalPath(commonDir)
	if err != nil {
		return nil, err
	}

	gitDirResult, err := runner.Run(ctx, cwd, "rev-parse", "--git-dir")
	if err != nil {
		return nil, fmt.Errorf("git dir could not be resolved: %w", err)
	}
	gitDir, err := ResolvePath(cwd, gitDirResult.Stdout)
	if err != nil {
		return nil, err
	}
	gitDir, err = EvalPath(gitDir)
	if err != nil {
		return nil, err
	}

	worktrees, err := ListWorktrees(ctx, runner, cwd)
	if err != nil {
		return nil, err
	}

	mainWorktreePath, err := findMainWorktreePath(worktrees, commonDir)
	if err != nil {
		return nil, err
	}

	isMain := samePath(root, mainWorktreePath)
	return &RepoContext{
		Root:                root,
		CommonDir:           commonDir,
		GitDir:              gitDir,
		MainWorktreePath:    mainWorktreePath,
		CurrentWorktreePath: root,
		IsMainWorktree:      isMain,
		IsLinkedWorktree:    !isMain,
	}, nil
}

func ResolvePath(base, raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("path is empty")
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw), nil
	}
	return filepath.Clean(filepath.Join(base, raw)), nil
}

func EvalPath(path string) (string, error) {
	evaluated, err := filepath.EvalSymlinks(path)
	if err == nil {
		return evaluated, nil
	}
	if os.IsNotExist(err) {
		return filepath.Clean(path), nil
	}
	return "", err
}

func ExpandHome(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path == "~" || path == "$HOME" || path == "${HOME}" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	switch {
	case strings.HasPrefix(path, "~/"):
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	case strings.HasPrefix(path, "$HOME/"):
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(path, "$HOME/")), nil
	case strings.HasPrefix(path, "${HOME}/"):
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(path, "${HOME}/")), nil
	default:
		return path, nil
	}
}

func samePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func findMainWorktreePath(worktrees []Worktree, commonDir string) (string, error) {
	for _, wt := range worktrees {
		candidate, err := EvalPath(filepath.Join(wt.Path, ".git"))
		if err != nil {
			continue
		}
		if samePath(candidate, commonDir) {
			resolved, err := EvalPath(wt.Path)
			if err != nil {
				return "", err
			}
			return resolved, nil
		}
	}
	if len(worktrees) == 0 {
		return "", fmt.Errorf("no worktrees found")
	}
	return EvalPath(worktrees[0].Path)
}
