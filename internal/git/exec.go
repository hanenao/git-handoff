package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type CommandError struct {
	Name     string
	Args     []string
	Dir      string
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

func (e *CommandError) Error() string {
	msg := fmt.Sprintf("%s %s failed", e.Name, strings.Join(e.Args, " "))
	if e.Dir != "" {
		msg += fmt.Sprintf(" (dir=%s)", e.Dir)
	}
	if e.Stderr != "" {
		msg += fmt.Sprintf(": %s", strings.TrimSpace(e.Stderr))
	}
	if e.Err != nil && e.Stderr == "" {
		msg += fmt.Sprintf(": %v", e.Err)
	}
	return msg
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

type Result struct {
	Stdout string
	Stderr string
}

type Runner interface {
	Run(ctx context.Context, dir string, args ...string) (Result, error)
}

type CLI struct{}

func (CLI) Run(ctx context.Context, dir string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{
		Stdout: strings.TrimSpace(stdout.String()),
		Stderr: strings.TrimSpace(stderr.String()),
	}
	if err == nil {
		return result, nil
	}

	var exitErr *exec.ExitError
	code := -1
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	}
	return result, &CommandError{
		Name:     "git",
		Args:     append([]string(nil), args...),
		Dir:      dir,
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: code,
		Err:      err,
	}
}

func IsExitCode(err error, code int) bool {
	var cmdErr *CommandError
	return errors.As(err, &cmdErr) && cmdErr.ExitCode == code
}
