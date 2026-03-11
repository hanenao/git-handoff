package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	ghgit "github.com/hanenao/git-handoff/internal/git"
	ghwt "github.com/hanenao/git-handoff/internal/worktree"
	"github.com/spf13/cobra"
)

func newWorktreeCreateCommand(options *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create a detached AI worktree",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			repo, err := options.resolveRepo(ctx)
			if err != nil {
				return err
			}
			cfg, err := options.resolveConfig(ctx, cmd, repo)
			if err != nil {
				return err
			}
			if err := ensureBasedirExcluded(repo, cfg.Basedir); err != nil {
				return err
			}
			if err := os.MkdirAll(cfg.Basedir, 0o755); err != nil {
				return err
			}

			id, err := ghwt.GenerateID(ctx, worktreeIDExists(repo.CommonDir))
			if err != nil {
				return err
			}
			path := filepath.Join(cfg.Basedir, id)

			if _, err := options.runner.Run(ctx, repo.CurrentWorktreePath, "worktree", "add", "--detach", path, "HEAD"); err != nil {
				return err
			}

			createdAt := time.Now().UTC()
			metadata := ghwt.Metadata{
				ID:        id,
				Path:      path,
				CreatedAt: createdAt,
				UpdatedAt: createdAt,
			}

			if cfg.CopyIgnored {
				if err := copyIgnoredFiles(ctx, options.runner, repo.CurrentWorktreePath, path); err != nil {
					return err
				}
			}
			if err := runHooks(ctx, path, cfg.Hooks, options.stdout, options.stderr); err != nil {
				return err
			}
			if err := ghwt.WriteMetadata(repo.CommonDir, metadata); err != nil {
				return err
			}
			return printLine(options.stdout, "created worktree: %s", id)
		},
	}
}

func copyIgnoredFiles(ctx context.Context, runner ghgit.Runner, sourceRoot, targetRoot string) error {
	result, err := runner.Run(ctx, sourceRoot, "ls-files", "--others", "-i", "--exclude-standard")
	if err != nil {
		return err
	}
	for relative := range strings.SplitSeq(strings.TrimSpace(result.Stdout), "\n") {
		if relative == "" {
			continue
		}
		sourcePath := filepath.Join(sourceRoot, relative)
		targetPath := filepath.Join(targetRoot, relative)
		if err := copyPath(sourcePath, targetPath); err != nil {
			return err
		}
	}
	return nil
}

func copyPath(source, target string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	switch mode := info.Mode(); {
	case mode.IsDir():
		return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			relative, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			targetPath := filepath.Join(target, relative)
			if entry.IsDir() {
				return os.MkdirAll(targetPath, 0o755)
			}
			return copyFile(path, targetPath, entry.Type())
		})
	case mode&os.ModeSymlink != 0:
		link, err := os.Readlink(source)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.Symlink(link, target)
	default:
		return copyFile(source, target, mode)
	}
}

func copyFile(source, target string, mode fs.FileMode) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, data, mode.Perm())
}

func runHooks(ctx context.Context, dir string, hooks []string, stdout, stderr io.Writer) error {
	for _, hook := range hooks {
		command := exec.CommandContext(ctx, "sh", "-c", hook)
		command.Dir = dir

		var stdoutBuffer bytes.Buffer
		var stderrBuffer bytes.Buffer
		command.Stdout = io.MultiWriter(stdout, &stdoutBuffer)
		command.Stderr = io.MultiWriter(stderr, &stderrBuffer)

		err := command.Run()
		if err != nil {
			output := strings.TrimSpace(strings.Join([]string{
				strings.TrimSpace(stdoutBuffer.String()),
				strings.TrimSpace(stderrBuffer.String()),
			}, "\n"))
			if output == "" {
				output = hook
			}
			return fmt.Errorf("hook failed: %s: %w", output, err)
		}
	}
	return nil
}

func ensureBasedirExcluded(repo *ghgit.RepoContext, basedir string) error {
	relative, err := filepath.Rel(repo.Root, basedir)
	if err != nil {
		return err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil
	}
	pattern := filepath.ToSlash(relative)
	if pattern == "." {
		return nil
	}
	if !strings.HasSuffix(pattern, "/") {
		pattern += "/"
	}

	excludePath := filepath.Join(repo.CommonDir, "info", "exclude")
	contents, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for line := range strings.SplitSeq(string(contents), "\n") {
		if strings.TrimSpace(line) == pattern {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(excludePath), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(excludePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "%s\n", pattern); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
