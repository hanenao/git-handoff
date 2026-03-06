package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hanenao/git-handoff/version"
)

func TestCLIHelpAndVersion(t *testing.T) {
	binary := buildBinary(t)

	help := runBinary(t, binary, filepath.Dir(binary), "--help")
	if !strings.Contains(help.stdout, "worktree") || !strings.Contains(help.stdout, "switch") || !strings.Contains(help.stdout, "--init") {
		t.Fatalf("help output is missing commands:\n%s", help.stdout)
	}

	initOut := runBinary(t, binary, filepath.Dir(binary), "--init", "zsh")
	if !strings.Contains(initOut.stdout, "command git-ho") || !strings.Contains(initOut.stdout, "GIT_HO_SHELL_INTEGRATION=1") {
		t.Fatalf("unexpected init output:\n%s", initOut.stdout)
	}

	versionResult := runBinary(t, binary, filepath.Dir(binary), "version")
	if trimLine(versionResult.stdout) != version.Version {
		t.Fatalf("unexpected version output: %q", versionResult.stdout)
	}
}

func TestInitFlagPrintsShellIntegration(t *testing.T) {
	binary := buildBinary(t)

	result := runBinary(t, binary, filepath.Dir(binary), "--init", "zsh")
	if !strings.Contains(result.stdout, "GIT_HO_SHELL_INTEGRATION=1") {
		t.Fatalf("unexpected init output:\n%s", result.stdout)
	}
}

func TestEnvCreateListAndRemove(t *testing.T) {
	binary := buildBinary(t)
	repo := newTestRepo(t)

	create := runBinary(t, binary, repo, "worktree", "create", "--hook", "touch .hooked")
	if !strings.Contains(create.stdout, "created worktree: ") {
		t.Fatalf("unexpected create output: %q", create.stdout)
	}
	worktreeID := strings.TrimSpace(strings.TrimPrefix(trimLine(create.stdout), "created worktree: "))
	worktreePath := resolvedPath(t, filepath.Join(repo, ".ho", worktreeID))

	if _, err := os.Stat(filepath.Join(worktreePath, ".hooked")); err != nil {
		t.Fatalf("hook output missing: %v", err)
	}

	list := runBinary(t, binary, repo, "worktree", "list")
	if !strings.Contains(list.stdout, "local") || !strings.Contains(list.stdout, worktreeID) {
		t.Fatalf("list output missing worktree:\n%s", list.stdout)
	}

	if err := os.Remove(filepath.Join(worktreePath, ".hooked")); err != nil {
		t.Fatalf("failed to clean worktree before remove: %v", err)
	}

	remove := runBinary(t, binary, repo, "worktree", "remove", worktreeID)
	if !strings.Contains(remove.stdout, "removed worktree: "+worktreeID) {
		t.Fatalf("unexpected remove output: %q", remove.stdout)
	}

	listAfter := runBinary(t, binary, repo, "worktree", "list")
	if strings.Contains(listAfter.stdout, worktreeID) {
		t.Fatalf("worktree still listed after remove:\n%s", listAfter.stdout)
	}
}

func TestEnvRemoveRejectsWorktreeWithUncommittedChanges(t *testing.T) {
	binary := buildBinary(t)
	repo := newTestRepo(t)

	create := runBinary(t, binary, repo, "worktree", "create")
	worktreeID := strings.TrimSpace(strings.TrimPrefix(trimLine(create.stdout), "created worktree: "))
	worktreePath := resolvedPath(t, filepath.Join(repo, ".ho", worktreeID))

	mustWriteFile(t, filepath.Join(worktreePath, "dirty.txt"), "dirty\n")

	result := runBinaryExpectError(t, binary, repo, "worktree", "remove", worktreeID)
	if !strings.Contains(result.stderr, "worktree "+worktreeID+" has uncommitted changes") {
		t.Fatalf("unexpected remove error: %q", result.stderr)
	}
}
