# git-handoff

`git-handoff` is a Git subcommand for handing your current branch to an AI workspace, continuing other work locally, and bringing the branch and changes back later. It uses Git worktrees so AI work can happen in a separate worktree while keeping verification and additional edits easy in your main local directory. It is inspired by the handoff workflow in the OpenAI Codex app.

## Purpose

Sometimes you want an AI to keep implementing a feature while you continue working on something else locally.

Git does not allow the same branch to be checked out in multiple locations at the same time.  
`git-handoff` works within that constraint by safely handing the branch off to a worktree and bringing it back to `local` later.

This tool is designed to make the following workflow easier:

- Use Git worktrees so AI work can happen in a separate worktree while keeping local verification easy
- Give the AI a dedicated directory for implementation
- Keep working on another branch in the main `local` directory
- Hand a branch off to the AI side when needed, then bring it back later
- Include staged / unstaged changes and untracked files in the handoff
- Do not include ignored files in the handoff

## Installation

```console
$ go install github.com/hanenao/git-handoff@latest
```

```console
$ brew install hanenao/homebrew-tap/git-handoff
```

If you use `direnv`, running `direnv allow` in this repo adds `bin/` to `PATH` and points `GOCACHE` to a repo-local directory.

## Shell Integration

If you want `git ho switch` or `git ho go <branch>` to automatically move you into the destination directory, load the shell integration:

```console
$ eval "$(git ho --init zsh)"
$ eval "$(git ho --init bash)"
$ git-ho --init fish | source
```

The `git()` wrapper only intercepts `git ho ...` commands and changes directory if the command succeeds and the last line of stdout is an existing directory.  
If another tool also overrides `git()`, they may conflict.

If you do not want automatic directory changes, add `--nocd` during initialization, set `git config ho.nocd true`, or run `git ho --nocd switch`.

## Commands

The main commands are:

```console
$ git ho worktree create
$ git ho worktree list
$ git ho worktree remove <worktree-id>

$ git ho switch [<worktree-id>]
$ git ho go <branch>
```

### `git ho worktree create`

Create a new worktree for AI work.

### `git ho worktree list`

Show the state of `local` and each worktree in a single list.  
You can see which branch lives where at a glance.

### `git ho worktree remove <worktree-id>`

Remove a worktree you no longer need.

### `git ho switch [<worktree-id>]`

- When run in `local`:
  - Hand off the current branch to an available worktree
  - After handoff, try to check out `ho.basebranch` in `local`; if that branch is unavailable, `local` stays detached
- When run in a worktree:
  - Bring the current branch back to `local`

To avoid checking out the same branch in both `local` and a worktree at the same time, the branch is detached from the source location during the move.
If `worktree-id` is omitted, an available worktree is selected automatically.
If shell integration is enabled, your current directory switches to the destination directory after success.

### `git ho go <branch>`

Helper command for moving the specified branch into the currently checked out directory.  
Even without shell integration, it prints the destination path on the last line of stdout.

## Terms

- `local`
  - The main directory you normally work in
  - The place where a human does verification and additional work
- `worktree`
  - A worktree for AI use
  - The place where the AI works in a separate directory
- `handoff`
  - The operation that moves a branch and its in-progress changes between `local` and a worktree

## Typical Workflow

### 1. Create a worktree

```console
$ git ho worktree create
created worktree: a8k2m9
```

Create a workspace for the AI first.

### 2. Create a branch in `local`

```console
$ git switch -c feature/order-cache
```

Create or check out the target branch in `local`.

### 3. Hand the branch off to a worktree

```console
$ git ho switch
/home/alice/.ho/worktree/a8k2m9
```

At this point, the branch and the handoff-eligible changes move to an available worktree.
After the command succeeds, you are in that worktree directory.

### 4. Let the AI work in the worktree

The AI can implement, run tests, and verify generated files in the worktree.  
While that happens, you can keep working on something else in `local`.

### 5. Bring the branch back to `local`

```console
$ git ho switch
/path/to/repo
```

When the AI work is done, hand the branch back to `local` so a human can review it and make any final edits.
After the command succeeds, you are back in the main local directory.

## Configuration

Configuration uses the `ho.*` namespace in `git config`.

1. command line flag
2. local config
3. global config
4. builtin default

Main settings:

- `ho.basedir`
  - Directory where worktrees are created
  - default: `$HOME/.ho/worktree`
- `ho.basebranch`
  - Branch to check out in `local` after handing work off to a worktree
  - If checkout fails because the branch does not exist or is already checked out elsewhere, `local` stays detached
  - default: `main`
- `ho.copyignored`
  - Whether ignored files are copied when creating a worktree
  - default: `false`
- `ho.hook`
  - Initialization command to run when creating a worktree
  - default: unset
- `ho.nocd`
  - Do not change directories automatically when using shell integration
  - default: `false`

## Reference

- [OpenAI Codex: Worktrees and Handoff](https://developers.openai.com/codex/app/worktrees/)
