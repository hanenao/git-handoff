package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGoCommandFindsBranchOwner(t *testing.T) {
	binary := buildBinary(t)
	repo := newTestRepo(t)
	resolvedRepo := resolvedPath(t, repo)

	mainPath := runBinary(t, binary, repo, "go", "main")
	if trimLine(mainPath.stdout) != resolvedRepo {
		t.Fatalf("expected main branch to live in repo root, got %q", mainPath.stdout)
	}

	runGit(t, repo, "checkout", "-b", "feature/path")
	create := runBinary(t, binary, repo, "worktree", "create")
	worktreeID := strings.TrimSpace(strings.TrimPrefix(trimLine(create.stdout), "created worktree: "))
	worktreePath := resolvedPath(t, filepath.Join(repo, ".ho", worktreeID))

	runBinary(t, binary, repo, "switch")
	owner := runBinary(t, binary, repo, "go", "feature/path")
	if trimLine(owner.stdout) != worktreePath {
		t.Fatalf("expected feature branch to live in %s, got %q", worktreePath, owner.stdout)
	}

	errResult := runBinaryExpectError(t, binary, repo, "go", "missing-branch")
	if !strings.Contains(errResult.stderr, "missing-branch") {
		t.Fatalf("unexpected error output: %q", errResult.stderr)
	}
}
