package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandCompletion(t *testing.T) {
	binary := buildBinary(t)
	repo := newTestRepo(t)

	runGit(t, repo, "checkout", "-b", "feature/alpha")
	create := runBinary(t, binary, repo, "worktree", "create")
	worktreeID := strings.TrimSpace(strings.TrimPrefix(trimLine(create.stdout), "created worktree: "))

	t.Run("go command completes branch owners", func(t *testing.T) {
		out := runBinary(t, binary, repo, "__complete", "go", "")
		if !strings.Contains(out.stdout, "main\t[branch] initial") {
			t.Fatalf("expected main branch completion with branch description, got:\n%s", out.stdout)
		}
		if !strings.Contains(out.stdout, "feature/alpha\t[local] initial") {
			t.Fatalf("expected feature branch completion, got:\n%s", out.stdout)
		}
	})

	t.Run("switch command completes idle worktrees", func(t *testing.T) {
		out := runBinary(t, binary, repo, "__complete", "switch", "")
		expected := worktreeID + "\t[idle] [detached]"
		if !strings.Contains(out.stdout, expected) {
			t.Fatalf("expected switch completion %q, got:\n%s", expected, out.stdout)
		}
	})

	t.Run("worktree remove completes removable worktrees", func(t *testing.T) {
		out := runBinary(t, binary, repo, "__complete", "worktree", "remove", "")
		expected := worktreeID + "\t[idle] [detached]"
		if !strings.Contains(out.stdout, expected) {
			t.Fatalf("expected remove completion %q, got:\n%s", expected, out.stdout)
		}
	})

	worktreePath := resolvedPath(t, filepath.Join(repo, ".ho", worktreeID))
	runBinary(t, binary, repo, "switch")

	t.Run("switch command omits worktree ids inside managed worktree", func(t *testing.T) {
		out := runBinary(t, binary, worktreePath, "__complete", "switch", "")
		if strings.Contains(out.stdout, worktreeID+"\t") {
			t.Fatalf("switch should not suggest worktree ids inside managed worktree, got:\n%s", out.stdout)
		}
	})

	t.Run("worktree remove omits current worktree", func(t *testing.T) {
		out := runBinary(t, binary, worktreePath, "__complete", "worktree", "remove", "")
		if strings.Contains(out.stdout, worktreeID+"\t") {
			t.Fatalf("current worktree should not be removable, got:\n%s", out.stdout)
		}
	})
}
