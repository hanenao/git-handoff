package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitScripts(t *testing.T) {
	binary := buildBinary(t)

	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "bash",
			args:     []string{"--init", "bash"},
			contains: []string{"git-ho shell integration for bash", "GIT_HO_SHELL_INTEGRATION=1", `if [ "$1" != "ho" ]; then`, "_git_ho()", "_git_ho_direct()"},
		},
		{
			name:     "zsh",
			args:     []string{"--init", "zsh"},
			contains: []string{"git-ho shell integration for zsh", "GIT_HO_SHELL_INTEGRATION=1", `if [ "$1" != "ho" ]; then`, "_git-ho()", "compdef _git-ho git-ho"},
		},
		{
			name:     "fish",
			args:     []string{"--init", "fish"},
			contains: []string{"git-ho shell integration for fish", "set -lx GIT_HO_SHELL_INTEGRATION 1", `if test "$argv[1]" != "ho"`, "__fish_git_ho_completions", "complete -x -c git-ho"},
		},
		{
			name:     "bash_nocd",
			args:     []string{"--init", "bash", "--nocd"},
			contains: []string{"Automatic cd is disabled because --nocd was specified.", "_git_ho()", "_git_ho_direct()"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runBinary(t, binary, filepath.Dir(binary), tt.args...)
			for _, needle := range tt.contains {
				if !strings.Contains(result.stdout, needle) {
					t.Fatalf("expected init output to contain %q:\n%s", needle, result.stdout)
				}
			}
			if strings.Contains(tt.name, "nocd") && strings.Contains(result.stdout, "GIT_HO_SHELL_INTEGRATION=1") {
				t.Fatalf("wrapper should not be emitted for --init --nocd:\n%s", result.stdout)
			}
		})
	}

	errResult := runBinaryExpectError(t, binary, filepath.Dir(binary), "--init", "unsupported")
	if !strings.Contains(errResult.stderr, `unsupported shell "unsupported"`) {
		t.Fatalf("unexpected unsupported-shell error: %q", errResult.stderr)
	}
}

func TestShellIntegrationSwitchChangesDirectory(t *testing.T) {
	binary := buildBinary(t)

	tests := []struct {
		name  string
		shell string
	}{
		{name: "bash", shell: "bash"},
		{name: "zsh", shell: "zsh"},
		{name: "fish", shell: "fish"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureShellAvailable(t, tt.shell)

			repo := newTestRepo(t)
			runGit(t, repo, "checkout", "-b", "feature/shell-"+tt.shell)
			create := runBinary(t, binary, repo, "worktree", "create")
			worktreeID := strings.TrimSpace(strings.TrimPrefix(trimLine(create.stdout), "created worktree: "))
			worktreePath := expectedWorktreePath(t, worktreeID)

			stdout, stderr, err := runShellScript(t, tt.shell, binary, repo, shellInitScript(tt.shell)+`
git ho switch
pwd
`)
			if err != nil {
				t.Fatalf("%s shell integration failed: %v\nstdout:\n%s\nstderr:\n%s", tt.shell, err, stdout, stderr)
			}

			lines := nonEmptyLines(stdout)
			if len(lines) < 2 {
				t.Fatalf("expected switch output and pwd, got:\n%s", stdout)
			}
			if lines[0] != worktreePath {
				t.Fatalf("expected shell wrapper to print target path %q, got %q", worktreePath, lines[0])
			}
			if lines[len(lines)-1] != worktreePath {
				t.Fatalf("expected pwd to be %q, got %q", worktreePath, lines[len(lines)-1])
			}
		})
	}
}

func TestShellIntegrationSwitchRespectsNoCDConfig(t *testing.T) {
	binary := buildBinary(t)

	tests := []struct {
		name  string
		shell string
	}{
		{name: "bash", shell: "bash"},
		{name: "zsh", shell: "zsh"},
		{name: "fish", shell: "fish"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureShellAvailable(t, tt.shell)

			repo := newTestRepo(t)
			runGit(t, repo, "config", "ho.nocd", "true")
			runGit(t, repo, "checkout", "-b", "feature/config-"+tt.shell)
			create := runBinary(t, binary, repo, "worktree", "create")
			worktreeID := strings.TrimSpace(strings.TrimPrefix(trimLine(create.stdout), "created worktree: "))
			worktreePath := expectedWorktreePath(t, worktreeID)
			resolvedRepo := resolvedPath(t, repo)

			stdout, stderr, err := runShellScript(t, tt.shell, binary, repo, shellInitScript(tt.shell)+`
git ho switch
pwd
`)
			if err != nil {
				t.Fatalf("%s shell integration failed: %v\nstdout:\n%s\nstderr:\n%s", tt.shell, err, stdout, stderr)
			}

			lines := nonEmptyLines(stdout)
			if len(lines) < 2 {
				t.Fatalf("expected switch output and pwd, got:\n%s", stdout)
			}
			if lines[0] != worktreePath {
				t.Fatalf("expected switch output to stay machine-readable path %q, got %q", worktreePath, lines[0])
			}
			if lines[len(lines)-1] != resolvedRepo {
				t.Fatalf("expected pwd to remain at repo root %q, got %q", resolvedRepo, lines[len(lines)-1])
			}
		})
	}
}

func TestShellIntegrationSwitchRespectsNoCDFlag(t *testing.T) {
	binary := buildBinary(t)

	tests := []struct {
		name  string
		shell string
	}{
		{name: "bash", shell: "bash"},
		{name: "zsh", shell: "zsh"},
		{name: "fish", shell: "fish"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureShellAvailable(t, tt.shell)

			repo := newTestRepo(t)
			runGit(t, repo, "checkout", "-b", "feature/flag-"+tt.shell)
			create := runBinary(t, binary, repo, "worktree", "create")
			worktreeID := strings.TrimSpace(strings.TrimPrefix(trimLine(create.stdout), "created worktree: "))
			worktreePath := expectedWorktreePath(t, worktreeID)
			resolvedRepo := resolvedPath(t, repo)

			stdout, stderr, err := runShellScript(t, tt.shell, binary, repo, shellInitScript(tt.shell)+`
git ho --nocd switch
pwd
`)
			if err != nil {
				t.Fatalf("%s shell integration failed: %v\nstdout:\n%s\nstderr:\n%s", tt.shell, err, stdout, stderr)
			}

			lines := nonEmptyLines(stdout)
			if len(lines) < 2 {
				t.Fatalf("expected switch output and pwd, got:\n%s", stdout)
			}
			if lines[0] != worktreePath {
				t.Fatalf("expected switch output to stay machine-readable path %q, got %q", worktreePath, lines[0])
			}
			if lines[len(lines)-1] != resolvedRepo {
				t.Fatalf("expected pwd to remain at repo root %q, got %q", resolvedRepo, lines[len(lines)-1])
			}
		})
	}
}

func TestShellIntegrationGoChangesDirectory(t *testing.T) {
	binary := buildBinary(t)

	tests := []struct {
		name  string
		shell string
	}{
		{name: "bash", shell: "bash"},
		{name: "zsh", shell: "zsh"},
		{name: "fish", shell: "fish"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureShellAvailable(t, tt.shell)

			repo := newTestRepo(t)
			runGit(t, repo, "checkout", "-b", "feature/go-"+tt.shell)
			create := runBinary(t, binary, repo, "worktree", "create")
			worktreeID := strings.TrimSpace(strings.TrimPrefix(trimLine(create.stdout), "created worktree: "))
			worktreePath := expectedWorktreePath(t, worktreeID)
			runBinary(t, binary, repo, "switch")

			stdout, stderr, err := runShellScript(t, tt.shell, binary, repo, shellInitScript(tt.shell)+`
git ho go feature/go-`+tt.shell+`
pwd
`)
			if err != nil {
				t.Fatalf("%s shell integration failed: %v\nstdout:\n%s\nstderr:\n%s", tt.shell, err, stdout, stderr)
			}

			lines := nonEmptyLines(stdout)
			if len(lines) < 2 {
				t.Fatalf("expected go output and pwd, got:\n%s", stdout)
			}
			if lines[0] != worktreePath {
				t.Fatalf("expected go output to be %q, got %q", worktreePath, lines[0])
			}
			if lines[len(lines)-1] != worktreePath {
				t.Fatalf("expected pwd to be %q, got %q", worktreePath, lines[len(lines)-1])
			}
		})
	}
}

func ensureShellAvailable(t *testing.T, shell string) {
	t.Helper()
	if _, err := exec.LookPath(shell); err != nil {
		t.Skipf("%s is not available", shell)
	}
}

func runShellScript(t *testing.T, shell, binary, dir, script string) (string, string, error) {
	t.Helper()

	cmd := exec.Command(shell, "-c", script)
	cmd.Dir = dir
	cmd.Env = append(
		isolatedEnv(t),
		"PATH="+filepath.Dir(binary)+string(os.PathListSeparator)+os.Getenv("PATH"),
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

func shellInitScript(shell string) string {
	switch shell {
	case "fish":
		return "git-ho --init fish | source"
	default:
		return "eval \"$(git-ho --init " + shell + ")\""
	}
}

func nonEmptyLines(output string) []string {
	lines := make([]string, 0)
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}
