package worktree

import (
	"context"
	"strings"
	"testing"

	ghgit "github.com/hanenao/git-handoff/internal/git"
	"github.com/hanenao/git-handoff/testutil"
)

func TestResolveCurrentBackgroundReportsGitHoWorktree(t *testing.T) {
	t.Parallel()

	repo := testutil.NewRepo(t)
	repoCtx := repo.RepoContext(repo.Root)

	_, _, err := NewManager(ghgit.CLI{}).ResolveCurrentBackground(context.Background(), repoCtx)
	if err == nil {
		t.Fatal("expected ResolveCurrentBackground to fail for local checkout")
	}
	if !strings.Contains(err.Error(), "current worktree is not a git-ho worktree") {
		t.Fatalf("unexpected error: %v", err)
	}
}
