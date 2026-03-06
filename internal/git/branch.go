package git

import (
	"context"
	"fmt"
)

func CurrentBranch(ctx context.Context, runner Runner, dir string) (string, bool, error) {
	result, err := runner.Run(ctx, dir, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err == nil {
		return result.Stdout, false, nil
	}
	if IsExitCode(err, 1) {
		return "", true, nil
	}
	return "", false, err
}

func CheckoutBranch(ctx context.Context, runner Runner, dir, branch string) error {
	if branch == "" {
		return fmt.Errorf("branch is required")
	}
	_, err := runner.Run(ctx, dir, "checkout", branch)
	return err
}

func DetachHead(ctx context.Context, runner Runner, dir string) error {
	_, err := runner.Run(ctx, dir, "checkout", "--detach")
	return err
}
