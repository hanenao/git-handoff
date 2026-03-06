package git

import (
	"context"
	"strings"
)

func IsWorktreeClean(ctx context.Context, runner Runner, dir string) (bool, error) {
	result, err := runner.Run(ctx, dir, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(result.Stdout) == "", nil
}

func HasChanges(ctx context.Context, runner Runner, dir string) (bool, error) {
	clean, err := IsWorktreeClean(ctx, runner, dir)
	if err != nil {
		return false, err
	}
	return !clean, nil
}
