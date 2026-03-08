package handoff

import (
	"context"
	"path/filepath"
	"testing"

	ghgit "github.com/hanenao/git-handoff/internal/git"
	"github.com/hanenao/git-handoff/testutil"
)

func TestServiceGoFindsLocalAndWorktreeOwners(t *testing.T) {
	t.Parallel()

	repo := testutil.NewRepo(t)
	service := NewService(ghgit.CLI{})

	ctx := context.Background()
	repoCtx := repo.RepoContext(repo.Root)

	mainPath, err := service.Go(ctx, repoCtx, "main")
	if err != nil {
		t.Fatalf("Go(main) failed: %v", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(repo.Root)
	if err != nil {
		t.Fatalf("EvalSymlinks failed: %v", err)
	}
	if filepath.Clean(mainPath) != filepath.Clean(resolvedRoot) {
		t.Fatalf("unexpected main owner: %s", mainPath)
	}

	repo.CreateBranch("feature/worktree-owner")
	worktreePath := repo.CreateDetachedWorktree("wt-go")
	if err := moveBranch(ctx, ghgit.CLI{}, repoCtx.CommonDir, repo.Root, worktreePath, "feature/worktree-owner", "main"); err != nil {
		t.Fatalf("moveBranch failed: %v", err)
	}

	worktreeRepoCtx := repo.RepoContext(repo.Root)
	ownerPath, err := service.Go(ctx, worktreeRepoCtx, "feature/worktree-owner")
	if err != nil {
		t.Fatalf("Go(feature/worktree-owner) failed: %v", err)
	}
	resolvedWorktree, err := filepath.EvalSymlinks(worktreePath)
	if err != nil {
		t.Fatalf("EvalSymlinks failed: %v", err)
	}
	if filepath.Clean(ownerPath) != filepath.Clean(resolvedWorktree) {
		t.Fatalf("unexpected worktree owner: %s", ownerPath)
	}
}
