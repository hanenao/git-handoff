package git_test

import (
	"context"
	"path/filepath"
	"testing"

	ghgit "github.com/hanenao/git-handoff/internal/git"
	"github.com/hanenao/git-handoff/testutil"
)

func TestResolveRepoContextMainAndLinkedWorktree(t *testing.T) {
	t.Parallel()

	repo := testutil.NewRepo(t)
	worktreePath := repo.CreateDetachedWorktree("wt1")
	resolvedRoot, err := ghgit.EvalPath(repo.Root)
	if err != nil {
		t.Fatalf("EvalPath(root) failed: %v", err)
	}

	mainCtx, err := ghgit.ResolveRepoContext(context.Background(), ghgit.CLI{}, repo.Root)
	if err != nil {
		t.Fatalf("ResolveRepoContext(main) failed: %v", err)
	}
	if !mainCtx.IsMainWorktree || mainCtx.IsLinkedWorktree {
		t.Fatalf("unexpected main context: %+v", mainCtx)
	}
	if filepath.Clean(mainCtx.MainWorktreePath) != filepath.Clean(resolvedRoot) {
		t.Fatalf("unexpected main worktree path: %s", mainCtx.MainWorktreePath)
	}

	worktreeCtx, err := ghgit.ResolveRepoContext(context.Background(), ghgit.CLI{}, worktreePath)
	if err != nil {
		t.Fatalf("ResolveRepoContext(worktree) failed: %v", err)
	}
	if worktreeCtx.IsMainWorktree || !worktreeCtx.IsLinkedWorktree {
		t.Fatalf("unexpected worktree context: %+v", worktreeCtx)
	}
	if filepath.Clean(worktreeCtx.MainWorktreePath) != filepath.Clean(resolvedRoot) {
		t.Fatalf("unexpected linked main worktree path: %s", worktreeCtx.MainWorktreePath)
	}
}
