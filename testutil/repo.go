package testutil

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	ghgit "github.com/hanenao/git-handoff/internal/git"
)

type Repo struct {
	T      *testing.T
	Root   string
	Runner ghgit.Runner
}

func NewRepo(t *testing.T) *Repo {
	t.Helper()

	root := t.TempDir()
	runGit(t, root, "init", "-b", "main")
	runGit(t, root, "config", "user.name", "git-handoff test")
	runGit(t, root, "config", "user.email", "git-handoff@example.com")

	repo := &Repo{
		T:      t,
		Root:   root,
		Runner: ghgit.CLI{},
	}

	repo.WriteFile("README.md", "initial\n")
	repo.Stage("README.md")
	repo.Commit("initial commit")
	return repo
}

func (r *Repo) WriteFile(relativePath, content string) {
	r.T.Helper()
	path := filepath.Join(r.Root, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		r.T.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		r.T.Fatalf("write file failed: %v", err)
	}
}

func (r *Repo) ReadFile(relativePath string) string {
	r.T.Helper()
	data, err := os.ReadFile(filepath.Join(r.Root, relativePath))
	if err != nil {
		r.T.Fatalf("read file failed: %v", err)
	}
	return string(data)
}

func (r *Repo) Stage(relativePath string) {
	r.T.Helper()
	runGit(r.T, r.Root, "add", relativePath)
}

func (r *Repo) Commit(message string) {
	r.T.Helper()
	runGit(r.T, r.Root, "commit", "-m", message)
}

func (r *Repo) Checkout(branch string) {
	r.T.Helper()
	runGit(r.T, r.Root, "checkout", branch)
}

func (r *Repo) CreateBranch(branch string) {
	r.T.Helper()
	runGit(r.T, r.Root, "checkout", "-b", branch)
}

func (r *Repo) CreateDetachedWorktree(name string) string {
	r.T.Helper()
	path := filepath.Join(r.T.TempDir(), name)
	runGit(r.T, r.Root, "worktree", "add", "--detach", path, "HEAD")
	return path
}

func (r *Repo) Git(dir string, args ...string) string {
	r.T.Helper()
	return runGit(r.T, dir, args...)
}

func (r *Repo) GitStatus(dir string) string {
	r.T.Helper()
	return strings.TrimSpace(runGit(r.T, dir, "status", "--porcelain=v1", "--untracked-files=all"))
}

func (r *Repo) CurrentBranch(dir string) (string, bool) {
	r.T.Helper()
	branch, detached, err := ghgit.CurrentBranch(context.Background(), r.Runner, dir)
	if err != nil {
		r.T.Fatalf("CurrentBranch failed: %v", err)
	}
	return branch, detached
}

func (r *Repo) RepoContext(dir string) *ghgit.RepoContext {
	r.T.Helper()
	ctx, err := ghgit.ResolveRepoContext(context.Background(), r.Runner, dir)
	if err != nil {
		r.T.Fatalf("ResolveRepoContext failed: %v", err)
	}
	return ctx
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed in %s: %v\n%s", strings.Join(args, " "), dir, err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output))
}
