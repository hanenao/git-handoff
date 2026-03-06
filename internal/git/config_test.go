package git_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	ghgit "github.com/hanenao/git-handoff/internal/git"
)

type fakeRunner struct {
	responses map[string]ghgit.Result
	errors    map[string]error
}

func (f fakeRunner) Run(_ context.Context, _ string, args ...string) (ghgit.Result, error) {
	key := strings.Join(args, "\x00")
	if err, ok := f.errors[key]; ok {
		return ghgit.Result{}, err
	}
	if result, ok := f.responses[key]; ok {
		return result, nil
	}
	return ghgit.Result{}, &ghgit.CommandError{Name: "git", Args: args, ExitCode: 1, Err: errors.New("not found")}
}

func TestLoadConfigAppliesPrecedenceAndRelativePath(t *testing.T) {
	t.Parallel()

	repo := &ghgit.RepoContext{
		Root:                "/repo/root",
		CurrentWorktreePath: "/repo/root",
	}
	runner := fakeRunner{
		responses: map[string]ghgit.Result{
			strings.Join([]string{"config", "--global", "--get", "ho.basedir"}, "\x00"):     {Stdout: "~/global-ho"},
			strings.Join([]string{"config", "--global", "--get", "ho.copyignored"}, "\x00"): {Stdout: "true"},
			strings.Join([]string{"config", "--global", "--get-all", "ho.hook"}, "\x00"):    {Stdout: "echo global"},
			strings.Join([]string{"config", "--local", "--get", "ho.basedir"}, "\x00"):      {Stdout: ".local-ho"},
			strings.Join([]string{"config", "--local", "--get", "ho.nocd"}, "\x00"):         {Stdout: "true"},
			strings.Join([]string{"config", "--local", "--get-all", "ho.hook"}, "\x00"):     {Stdout: "echo local 1\necho local 2"},
		},
		errors: map[string]error{},
	}

	overrideCopyIgnored := false
	overrideHooks := []string{"echo override"}
	cfg, err := ghgit.LoadConfig(context.Background(), runner, repo, ghgit.ConfigOverrides{
		CopyIgnored: &overrideCopyIgnored,
		Hooks:       &overrideHooks,
	})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Basedir != filepath.Join(repo.Root, ".local-ho") {
		t.Fatalf("unexpected basedir: %s", cfg.Basedir)
	}
	if cfg.CopyIgnored {
		t.Fatalf("expected override to disable copyignored")
	}
	if !cfg.NoCD {
		t.Fatalf("expected local ho.nocd to be true")
	}
	if len(cfg.Hooks) != 1 || cfg.Hooks[0] != "echo override" {
		t.Fatalf("unexpected hooks: %#v", cfg.Hooks)
	}
}

func TestLoadConfigFallsBackToDefaults(t *testing.T) {
	t.Parallel()

	repo := &ghgit.RepoContext{
		Root:                "/repo/root",
		CurrentWorktreePath: "/repo/root",
	}
	cfg, err := ghgit.LoadConfig(context.Background(), fakeRunner{}, repo, ghgit.ConfigOverrides{})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Basedir != filepath.Join(repo.Root, ".ho") {
		t.Fatalf("unexpected default basedir: %s", cfg.Basedir)
	}
	if cfg.CopyIgnored || cfg.NoCD || len(cfg.Hooks) != 0 {
		t.Fatalf("unexpected default config: %+v", cfg)
	}
}
