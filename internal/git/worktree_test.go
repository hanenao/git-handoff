package git_test

import (
	"testing"

	ghgit "github.com/hanenao/git-handoff/internal/git"
)

func TestParseWorktreeList(t *testing.T) {
	t.Parallel()

	raw := `
worktree /tmp/repo
HEAD 1111111111111111111111111111111111111111
branch refs/heads/main

worktree /tmp/repo/.ho/wt1
HEAD 2222222222222222222222222222222222222222
detached
`

	worktrees, err := ghgit.ParseWorktreeList(raw)
	if err != nil {
		t.Fatalf("ParseWorktreeList failed: %v", err)
	}
	if len(worktrees) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(worktrees))
	}

	if worktrees[0].Path != "/tmp/repo" || worktrees[0].BranchName != "main" || worktrees[0].Detached {
		t.Fatalf("unexpected first worktree: %+v", worktrees[0])
	}
	if worktrees[1].Path != "/tmp/repo/.ho/wt1" || !worktrees[1].Detached || worktrees[1].BranchName != "" {
		t.Fatalf("unexpected second worktree: %+v", worktrees[1])
	}

	owner := ghgit.FindBranchOwner(worktrees, "main")
	if owner == nil || owner.Path != "/tmp/repo" {
		t.Fatalf("expected branch owner to be /tmp/repo, got %+v", owner)
	}
}
