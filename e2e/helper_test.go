package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type commandResult struct {
	stdout string
	stderr string
}

func buildBinary(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller")
	}
	repoRoot := filepath.Dir(filepath.Dir(filename))
	output := filepath.Join(t.TempDir(), "git-ho")

	cmd := exec.Command("go", "build", "-o", output, ".")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(repoRoot, ".cache", "go-build"))
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, string(data))
	}
	return output
}

func newTestRepo(t *testing.T) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), "repo")
	mustMkdirAll(t, root)

	runGit(t, root, "init")
	runGit(t, root, "config", "user.name", "git-handoff")
	runGit(t, root, "config", "user.email", "git-handoff@example.com")
	runGit(t, root, "checkout", "-b", "main")

	mustWriteFile(t, filepath.Join(root, ".gitignore"), "*.secret\n")
	mustWriteFile(t, filepath.Join(root, "tracked.txt"), "base\n")
	runGit(t, root, "add", ".gitignore", "tracked.txt")
	runGit(t, root, "commit", "-m", "initial")
	return root
}

func runBinary(t *testing.T, binary, dir string, args ...string) commandResult {
	t.Helper()

	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	cmd.Env = isolatedEnv(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("command failed: %s %s\nstdout:\n%s\nstderr:\n%s\nerror: %v", binary, strings.Join(args, " "), stdout.String(), stderr.String(), err)
	}
	return commandResult{stdout: stdout.String(), stderr: stderr.String()}
}

func runBinaryExpectError(t *testing.T, binary, dir string, args ...string) commandResult {
	t.Helper()

	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	cmd.Env = isolatedEnv(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatalf("command unexpectedly succeeded: %s %s", binary, strings.Join(args, " "))
	}
	return commandResult{stdout: stdout.String(), stderr: stderr.String()}
}

func runGit(t *testing.T, dir string, args ...string) commandResult {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = isolatedEnv(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s failed\nstdout:\n%s\nstderr:\n%s\nerror: %v", strings.Join(args, " "), stdout.String(), stderr.String(), err)
	}
	return commandResult{stdout: strings.TrimSpace(stdout.String()), stderr: strings.TrimSpace(stderr.String())}
}

func isolatedEnv(t *testing.T) []string {
	t.Helper()

	home := filepath.Join(t.TempDir(), "home")
	mustMkdirAll(t, home)
	return append(
		os.Environ(),
		"HOME="+home,
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"XDG_CONFIG_HOME="+home,
	)
}

func mustWriteFile(t *testing.T, path, contents string) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create %s: %v", path, err)
	}
}

func branchName(t *testing.T, dir string) string {
	t.Helper()
	return runGit(t, dir, "rev-parse", "--abbrev-ref", "HEAD").stdout
}

func resolvedPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("failed to resolve %s: %v", path, err)
	}
	return resolved
}

func trimLine(value string) string {
	return strings.TrimSpace(value)
}
