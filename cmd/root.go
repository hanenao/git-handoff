package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	ghgit "github.com/hanenao/git-handoff/internal/git"
	ghwt "github.com/hanenao/git-handoff/internal/worktree"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	runner      ghgit.Runner
	stdout      io.Writer
	stderr      io.Writer
	initShell   string
	basedir     string
	copyIgnored bool
	hooks       []string
	noCD        bool
}

func Execute() error {
	return NewRootCommand().Execute()
}

func NewRootCommand() *cobra.Command {
	options := &rootOptions{
		runner: ghgit.CLI{},
		stdout: os.Stdout,
		stderr: os.Stderr,
	}

	rootCmd := &cobra.Command{
		Use:   "git-ho",
		Short: "Safely hand off branches between local and worktree checkouts",
		Long: `git-ho safely hands off branches between local and worktree checkouts.

Shell integration:
  eval "$(git ho --init zsh)"
  eval "$(git ho --init bash)"
  git-ho --init fish | source
`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if options.initShell == "" {
				return cmd.Help()
			}
			script, err := renderInitScript(options.initShell, options.noCD)
			if err != nil {
				return err
			}
			return printLine(options.stdout, "%s", script)
		},
	}

	rootCmd.PersistentFlags().StringVar(&options.initShell, "init", "", "print shell integration for bash, zsh, or fish")
	rootCmd.PersistentFlags().StringVar(&options.basedir, "basedir", "", "override ho.basedir")
	rootCmd.PersistentFlags().BoolVar(&options.copyIgnored, "copyignored", false, "override ho.copyignored")
	rootCmd.PersistentFlags().StringArrayVar(&options.hooks, "hook", nil, "override ho.hook")
	rootCmd.PersistentFlags().BoolVar(&options.noCD, "nocd", false, "override ho.nocd")
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(newWorktreeCommand(options))
	rootCmd.AddCommand(newSwitchCommand(options))
	rootCmd.AddCommand(newGoCommand(options))
	rootCmd.AddCommand(newVersionCommand(options))

	return rootCmd
}

func (o *rootOptions) resolveRepo(ctx context.Context) (*ghgit.RepoContext, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return ghgit.ResolveRepoContext(ctx, o.runner, cwd)
}

func (o *rootOptions) resolveConfig(ctx context.Context, cmd *cobra.Command, repo *ghgit.RepoContext) (ghgit.Config, error) {
	return ghgit.LoadConfig(ctx, o.runner, repo, o.overrides(cmd))
}

func (o *rootOptions) overrides(cmd *cobra.Command) ghgit.ConfigOverrides {
	flags := cmd.Flags()
	overrides := ghgit.ConfigOverrides{}
	if flags.Changed("basedir") {
		overrides.Basedir = &o.basedir
	}
	if flags.Changed("copyignored") {
		overrides.CopyIgnored = &o.copyIgnored
	}
	if flags.Changed("hook") {
		hooks := append([]string(nil), o.hooks...)
		overrides.Hooks = &hooks
	}
	if flags.Changed("nocd") {
		overrides.NoCD = &o.noCD
	}
	return overrides
}

func worktreeIDExists(commonDir string) func(context.Context, string) (bool, error) {
	return func(_ context.Context, id string) (bool, error) {
		_, err := ghwt.ReadMetadata(commonDir, id)
		if err == nil {
			return true, nil
		}
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
}

func printLine(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format+"\n", args...)
	return err
}
