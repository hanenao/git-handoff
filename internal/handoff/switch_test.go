package handoff

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	ghgit "github.com/hanenao/git-handoff/internal/git"
	"github.com/hanenao/git-handoff/testutil"
)

func TestMoveBranchPreservesDirtyStateAndRestoresBaseBranch(t *testing.T) {
	t.Parallel()

	repo := testutil.NewRepo(t)
	repo.CreateBranch("feature/dirty-handoff")
	repo.WriteFile("staged.txt", "staged change\n")
	repo.Stage("staged.txt")
	repo.WriteFile("unstaged.txt", "unstaged change\n")
	repo.WriteFile("notes.txt", "untracked\n")

	targetPath := repo.CreateDetachedWorktree("wt1")
	ctx := repo.RepoContext(repo.Root)

	if err := moveBranch(context.Background(), ghgit.CLI{}, ctx.CommonDir, repo.Root, targetPath, "feature/dirty-handoff", "main"); err != nil {
		t.Fatalf("moveBranch failed: %v", err)
	}

	branch, detached := repo.CurrentBranch(targetPath)
	if detached || branch != "feature/dirty-handoff" {
		t.Fatalf("unexpected target branch state: branch=%q detached=%v", branch, detached)
	}

	targetStatus := repo.GitStatus(targetPath)
	for _, expected := range []string{"A  staged.txt", "?? notes.txt", "?? unstaged.txt"} {
		if !strings.Contains(targetStatus, expected) {
			t.Fatalf("expected target status to contain %q, got:\n%s", expected, targetStatus)
		}
	}

	sourceBranch, sourceDetached := repo.CurrentBranch(repo.Root)
	if sourceDetached || sourceBranch != "main" {
		t.Fatalf("expected source local checkout to move to main, got branch=%q detached=%v", sourceBranch, sourceDetached)
	}
	if sourceStatus := repo.GitStatus(repo.Root); sourceStatus != "" {
		t.Fatalf("expected source local checkout to be clean, got:\n%s", sourceStatus)
	}
}

func TestMoveBranchLeavesSourceDetachedWhenBaseBranchCheckoutFails(t *testing.T) {
	t.Parallel()

	repo := testutil.NewRepo(t)
	repo.CreateBranch("feature/fallback-detached")

	mainOwnerPath := filepath.Join(t.TempDir(), "main-owner")
	repo.Git(repo.Root, "worktree", "add", mainOwnerPath, "main")

	targetPath := repo.CreateDetachedWorktree("wt-fallback")
	ctx := repo.RepoContext(repo.Root)

	if err := moveBranch(context.Background(), ghgit.CLI{}, ctx.CommonDir, repo.Root, targetPath, "feature/fallback-detached", "main"); err != nil {
		t.Fatalf("moveBranch failed: %v", err)
	}

	sourceBranch, sourceDetached := repo.CurrentBranch(repo.Root)
	if !sourceDetached || sourceBranch != "" {
		t.Fatalf("expected source local checkout to stay detached, got branch=%q detached=%v", sourceBranch, sourceDetached)
	}
}
