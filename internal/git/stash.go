package git

import (
	"context"
)

func CreateStash(ctx context.Context, runner Runner, dir, message string) (string, bool, error) {
	hasChanges, err := HasChanges(ctx, runner, dir)
	if err != nil {
		return "", false, err
	}
	if !hasChanges {
		return "", false, nil
	}
	if _, err := runner.Run(ctx, dir, "stash", "push", "--include-untracked", "-m", message); err != nil {
		return "", false, err
	}
	return "stash@{0}", true, nil
}

func ApplyStash(ctx context.Context, runner Runner, dir, ref string) error {
	if ref == "" {
		return nil
	}
	_, err := runner.Run(ctx, dir, "stash", "apply", "--index", ref)
	return err
}

func DropStash(ctx context.Context, runner Runner, dir, ref string) error {
	if ref == "" {
		return nil
	}
	_, err := runner.Run(ctx, dir, "stash", "drop", ref)
	return err
}
