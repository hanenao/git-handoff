package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSwitchRoundTripPreservesDirtyChanges(t *testing.T) {
	binary := buildBinary(t)
	repo := newTestRepo(t)

	runGit(t, repo, "checkout", "-b", "feature/dirty")
	mustWriteFile(t, filepath.Join(repo, "tracked.txt"), "base\nunstaged\n")
	mustWriteFile(t, filepath.Join(repo, "staged.txt"), "staged\n")
	runGit(t, repo, "add", "staged.txt")
	mustWriteFile(t, filepath.Join(repo, "untracked.txt"), "hello\n")
	mustWriteFile(t, filepath.Join(repo, "local.secret"), "keep-local\n")

	create := runBinary(t, binary, repo, "worktree", "create")
	worktreeID := strings.TrimSpace(strings.TrimPrefix(trimLine(create.stdout), "created worktree: "))
	worktreePath := expectedWorktreePath(t, worktreeID)

	switchOut := runBinary(t, binary, repo, "switch")
	if resolvedPath(t, trimLine(switchOut.stdout)) != worktreePath {
		t.Fatalf("unexpected switch output: %q", switchOut.stdout)
	}
	if branchName(t, repo) != "main" {
		t.Fatalf("local should switch back to main after handoff, got %s", branchName(t, repo))
	}
	if branchName(t, worktreePath) != "feature/dirty" {
		t.Fatalf("worktree branch mismatch: %s", branchName(t, worktreePath))
	}

	if data, err := os.ReadFile(filepath.Join(worktreePath, "tracked.txt")); err != nil || string(data) != "base\nunstaged\n" {
		t.Fatalf("tracked change did not move: %v %q", err, string(data))
	}
	if _, err := os.Stat(filepath.Join(worktreePath, "untracked.txt")); err != nil {
		t.Fatalf("untracked file did not move: %v", err)
	}
	if _, err := os.Stat(filepath.Join(worktreePath, "local.secret")); err == nil {
		t.Fatal("ignored file should not move during handoff")
	}
	if cached := runGit(t, worktreePath, "diff", "--cached", "--name-only").stdout; !strings.Contains(cached, "staged.txt") {
		t.Fatalf("staged change did not preserve index state: %q", cached)
	}

	back := runBinary(t, binary, worktreePath, "switch")
	if trimLine(back.stdout) != resolvedPath(t, repo) {
		t.Fatalf("unexpected switch-back output: %q", back.stdout)
	}
	if branchName(t, repo) != "feature/dirty" {
		t.Fatalf("local branch mismatch after return: %s", branchName(t, repo))
	}
	if branchName(t, worktreePath) != "HEAD" {
		t.Fatalf("worktree should be detached after return, got %s", branchName(t, worktreePath))
	}
	if cached := runGit(t, repo, "diff", "--cached", "--name-only").stdout; !strings.Contains(cached, "staged.txt") {
		t.Fatalf("staged change missing after return: %q", cached)
	}
	if _, err := os.Stat(filepath.Join(repo, "untracked.txt")); err != nil {
		t.Fatalf("untracked file missing after return: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "local.secret")); err != nil {
		t.Fatalf("ignored file should stay in local: %v", err)
	}
}

func TestSwitchRejectsDetachedForeground(t *testing.T) {
	binary := buildBinary(t)
	repo := newTestRepo(t)

	runBinary(t, binary, repo, "worktree", "create")
	runGit(t, repo, "checkout", "--detach")

	result := runBinaryExpectError(t, binary, repo, "switch")
	if !strings.Contains(result.stderr, "detached HEAD") {
		t.Fatalf("unexpected error: %q", result.stderr)
	}
}

func TestSwitchRejectsAttachedWorktree(t *testing.T) {
	binary := buildBinary(t)
	repo := newTestRepo(t)

	runGit(t, repo, "checkout", "-b", "feature/attached")
	create := runBinary(t, binary, repo, "worktree", "create")
	worktreeID := strings.TrimSpace(strings.TrimPrefix(trimLine(create.stdout), "created worktree: "))
	worktreePath := expectedWorktreePath(t, worktreeID)
	runGit(t, worktreePath, "checkout", "main")

	result := runBinaryExpectError(t, binary, repo, "switch", worktreeID)
	if !strings.Contains(result.stderr, "not idle") {
		t.Fatalf("unexpected error: %q", result.stderr)
	}
}

func TestSwitchRejectsDirtyDestinationWorktree(t *testing.T) {
	binary := buildBinary(t)
	repo := newTestRepo(t)

	runGit(t, repo, "checkout", "-b", "feature/dirty-target")
	create := runBinary(t, binary, repo, "worktree", "create")
	worktreeID := strings.TrimSpace(strings.TrimPrefix(trimLine(create.stdout), "created worktree: "))
	worktreePath := expectedWorktreePath(t, worktreeID)
	mustWriteFile(t, filepath.Join(worktreePath, "dirty.txt"), "dirty\n")

	result := runBinaryExpectError(t, binary, repo, "switch", worktreeID)
	if !strings.Contains(result.stderr, "destination worktree at "+rawExpectedWorktreePath(t, worktreeID)+" is not clean") {
		t.Fatalf("unexpected error: %q", result.stderr)
	}
}

func TestSwitchRejectsDirtyForegroundOnReturn(t *testing.T) {
	binary := buildBinary(t)
	repo := newTestRepo(t)

	runGit(t, repo, "checkout", "-b", "feature/return-block")
	create := runBinary(t, binary, repo, "worktree", "create")
	worktreeID := strings.TrimSpace(strings.TrimPrefix(trimLine(create.stdout), "created worktree: "))
	worktreePath := expectedWorktreePath(t, worktreeID)

	runBinary(t, binary, repo, "switch")
	mustWriteFile(t, filepath.Join(repo, "blocking.txt"), "local\n")

	result := runBinaryExpectError(t, binary, worktreePath, "switch")
	if !strings.Contains(result.stderr, "local has uncommitted changes") {
		t.Fatalf("unexpected error: %q", result.stderr)
	}
}

func TestSwitchRejectsDetachedWorktreeSourceOnReturn(t *testing.T) {
	binary := buildBinary(t)
	repo := newTestRepo(t)

	runGit(t, repo, "checkout", "-b", "feature/worktree-detached-source")
	create := runBinary(t, binary, repo, "worktree", "create")
	worktreeID := strings.TrimSpace(strings.TrimPrefix(trimLine(create.stdout), "created worktree: "))
	worktreePath := expectedWorktreePath(t, worktreeID)

	runBinary(t, binary, repo, "switch", worktreeID)
	runGit(t, worktreePath, "checkout", "--detach")

	result := runBinaryExpectError(t, binary, worktreePath, "switch")
	if !strings.Contains(result.stderr, "worktree "+worktreeID+" is detached HEAD") {
		t.Fatalf("unexpected error: %q", result.stderr)
	}
	if branchName(t, worktreePath) != "HEAD" {
		t.Fatalf("worktree should stay detached, got %s", branchName(t, worktreePath))
	}
	if branchName(t, repo) != "main" {
		t.Fatalf("local should stay on base branch after failed return, got %s", branchName(t, repo))
	}
}

func TestSwitchLeavesLocalDetachedWhenBaseBranchIsBusy(t *testing.T) {
	binary := buildBinary(t)
	repo := newTestRepo(t)

	runGit(t, repo, "checkout", "-b", "feature/basebranch-busy")
	mainOwnerPath := filepath.Join(t.TempDir(), "main-owner")
	runGit(t, repo, "worktree", "add", mainOwnerPath, "main")
	create := runBinary(t, binary, repo, "worktree", "create")
	worktreeID := strings.TrimSpace(strings.TrimPrefix(trimLine(create.stdout), "created worktree: "))
	worktreePath := expectedWorktreePath(t, worktreeID)

	switchOut := runBinary(t, binary, repo, "switch", worktreeID)
	if resolvedPath(t, trimLine(switchOut.stdout)) != worktreePath {
		t.Fatalf("unexpected switch output: %q", switchOut.stdout)
	}
	if branchName(t, repo) != "HEAD" {
		t.Fatalf("local should stay detached when main is busy, got %s", branchName(t, repo))
	}
}

func TestSwitchUsesConfiguredBaseBranch(t *testing.T) {
	binary := buildBinary(t)
	repo := newTestRepo(t)

	runGit(t, repo, "branch", "master", "main")
	runGit(t, repo, "config", "--local", "ho.basebranch", "master")
	runGit(t, repo, "checkout", "-b", "feature/config-base")
	create := runBinary(t, binary, repo, "worktree", "create")
	worktreeID := strings.TrimSpace(strings.TrimPrefix(trimLine(create.stdout), "created worktree: "))
	worktreePath := expectedWorktreePath(t, worktreeID)

	switchOut := runBinary(t, binary, repo, "switch", worktreeID)
	if resolvedPath(t, trimLine(switchOut.stdout)) != worktreePath {
		t.Fatalf("unexpected switch output: %q", switchOut.stdout)
	}
	if branchName(t, repo) != "master" {
		t.Fatalf("local should switch to configured base branch, got %s", branchName(t, repo))
	}
}
