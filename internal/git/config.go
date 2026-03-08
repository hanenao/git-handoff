package git

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Basedir     string
	BaseBranch  string
	CopyIgnored bool
	Hooks       []string
	NoCD        bool
}

type ConfigOverrides struct {
	Basedir     *string
	BaseBranch  *string
	CopyIgnored *bool
	Hooks       *[]string
	NoCD        *bool
}

func DefaultConfig() Config {
	return Config{
		Basedir:     ".ho",
		BaseBranch:  "main",
		CopyIgnored: false,
		Hooks:       nil,
		NoCD:        false,
	}
}

func LoadConfig(ctx context.Context, runner Runner, repo *RepoContext, overrides ConfigOverrides) (Config, error) {
	cfg := DefaultConfig()

	if repo != nil {
		if value, ok, err := readConfigValue(ctx, runner, repo.CurrentWorktreePath, "--global", "ho.basedir"); err != nil {
			return cfg, err
		} else if ok {
			cfg.Basedir = value
		}
		if value, ok, err := readConfigValue(ctx, runner, repo.CurrentWorktreePath, "--global", "ho.basebranch"); err != nil {
			return cfg, err
		} else if ok {
			cfg.BaseBranch = value
		}
		if value, ok, err := readConfigBool(ctx, runner, repo.CurrentWorktreePath, "--global", "ho.copyignored"); err != nil {
			return cfg, err
		} else if ok {
			cfg.CopyIgnored = value
		}
		if values, ok, err := readConfigValues(ctx, runner, repo.CurrentWorktreePath, "--global", "ho.hook"); err != nil {
			return cfg, err
		} else if ok {
			cfg.Hooks = values
		}
		if value, ok, err := readConfigBool(ctx, runner, repo.CurrentWorktreePath, "--global", "ho.nocd"); err != nil {
			return cfg, err
		} else if ok {
			cfg.NoCD = value
		}

		if value, ok, err := readConfigValue(ctx, runner, repo.CurrentWorktreePath, "--local", "ho.basedir"); err != nil {
			return cfg, err
		} else if ok {
			cfg.Basedir = value
		}
		if value, ok, err := readConfigValue(ctx, runner, repo.CurrentWorktreePath, "--local", "ho.basebranch"); err != nil {
			return cfg, err
		} else if ok {
			cfg.BaseBranch = value
		}
		if value, ok, err := readConfigBool(ctx, runner, repo.CurrentWorktreePath, "--local", "ho.copyignored"); err != nil {
			return cfg, err
		} else if ok {
			cfg.CopyIgnored = value
		}
		if values, ok, err := readConfigValues(ctx, runner, repo.CurrentWorktreePath, "--local", "ho.hook"); err != nil {
			return cfg, err
		} else if ok {
			cfg.Hooks = values
		}
		if value, ok, err := readConfigBool(ctx, runner, repo.CurrentWorktreePath, "--local", "ho.nocd"); err != nil {
			return cfg, err
		} else if ok {
			cfg.NoCD = value
		}
	}

	if overrides.Basedir != nil {
		cfg.Basedir = *overrides.Basedir
	}
	if overrides.BaseBranch != nil {
		cfg.BaseBranch = *overrides.BaseBranch
	}
	if overrides.CopyIgnored != nil {
		cfg.CopyIgnored = *overrides.CopyIgnored
	}
	if overrides.Hooks != nil {
		cfg.Hooks = append([]string(nil), (*overrides.Hooks)...)
	}
	if overrides.NoCD != nil {
		cfg.NoCD = *overrides.NoCD
	}

	expanded, err := ExpandHome(cfg.Basedir)
	if err != nil {
		return cfg, err
	}
	cfg.Basedir = expanded
	if repo != nil && cfg.Basedir != "" && !filepath.IsAbs(cfg.Basedir) {
		cfg.Basedir = filepath.Join(repo.Root, cfg.Basedir)
	}
	return cfg, nil
}

func readConfigValue(ctx context.Context, runner Runner, dir, scope, key string) (string, bool, error) {
	result, err := runner.Run(ctx, dir, "config", scope, "--get", key)
	if err != nil {
		if IsExitCode(err, 1) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to read %s: %w", key, err)
	}
	return result.Stdout, true, nil
}

func readConfigValues(ctx context.Context, runner Runner, dir, scope, key string) ([]string, bool, error) {
	result, err := runner.Run(ctx, dir, "config", scope, "--get-all", key)
	if err != nil {
		if IsExitCode(err, 1) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to read %s: %w", key, err)
	}
	if result.Stdout == "" {
		return nil, false, nil
	}
	return splitLines(result.Stdout), true, nil
}

func readConfigBool(ctx context.Context, runner Runner, dir, scope, key string) (bool, bool, error) {
	value, ok, err := readConfigValue(ctx, runner, dir, scope, key)
	if err != nil || !ok {
		return false, ok, err
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, false, fmt.Errorf("failed to parse %s as bool: %w", key, err)
	}
	return parsed, true, nil
}

func splitLines(raw string) []string {
	out := make([]string, 0)
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
