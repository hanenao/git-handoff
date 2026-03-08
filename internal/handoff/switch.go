package handoff

import (
	"context"
	"fmt"
	"path/filepath"

	ghgit "github.com/hanenao/git-handoff/internal/git"
	ghwt "github.com/hanenao/git-handoff/internal/worktree"
)

type Service struct {
	Runner    ghgit.Runner
	Worktrees *ghwt.Manager
}

func NewService(runner ghgit.Runner) *Service {
	return &Service{
		Runner:    runner,
		Worktrees: ghwt.NewManager(runner),
	}
}

func (s *Service) Switch(ctx context.Context, repo *ghgit.RepoContext, worktreeID string, cfg ghgit.Config) (string, error) {
	currentWorktree, _, err := s.Worktrees.ResolveCurrentBackground(ctx, repo)
	if err == nil && filepath.Clean(currentWorktree.Path) == filepath.Clean(repo.CurrentWorktreePath) {
		if worktreeID != "" {
			return "", fmt.Errorf("worktree-id cannot be specified when switching from worktree to local")
		}
		return s.switchToLocal(ctx, repo, currentWorktree)
	}

	if worktreeID != "" {
		target, _, err := s.Worktrees.Resolve(ctx, repo, worktreeID)
		if err != nil {
			return "", err
		}
		if target.State != ghwt.StateIdle {
			return "", fmt.Errorf("worktree %s is not idle", worktreeID)
		}
		return s.switchToWorktree(ctx, repo, target, cfg.BaseBranch)
	}

	target, _, err := s.Worktrees.SelectIdle(ctx, repo)
	if err != nil {
		return "", err
	}
	return s.switchToWorktree(ctx, repo, target, cfg.BaseBranch)
}

func (s *Service) Go(ctx context.Context, repo *ghgit.RepoContext, branch string) (string, error) {
	worktrees, err := ghgit.ListWorktrees(ctx, s.Runner, repo.CurrentWorktreePath)
	if err != nil {
		return "", err
	}
	owner := ghgit.FindBranchOwner(worktrees, branch)
	if owner == nil {
		return "", fmt.Errorf("branch %s is not checked out in any worktree", branch)
	}
	return owner.Path, nil
}

func (s *Service) switchToWorktree(ctx context.Context, repo *ghgit.RepoContext, target ghwt.Environment, baseBranch string) (string, error) {
	branch, detached, err := ghgit.CurrentBranch(ctx, s.Runner, repo.CurrentWorktreePath)
	if err != nil {
		return "", err
	}
	if detached {
		return "", fmt.Errorf("local is detached HEAD; checkout a branch before handoff")
	}
	if err := moveBranch(ctx, s.Runner, repo.CommonDir, repo.CurrentWorktreePath, target.Path, branch, baseBranch); err != nil {
		return "", err
	}
	if err := s.Worktrees.Touch(repo.CommonDir, target.ID); err != nil {
		return "", err
	}
	return target.Path, nil
}

func (s *Service) switchToLocal(ctx context.Context, repo *ghgit.RepoContext, source ghwt.Environment) (string, error) {
	branch, detached, err := ghgit.CurrentBranch(ctx, s.Runner, source.Path)
	if err != nil {
		return "", err
	}
	if detached {
		return "", fmt.Errorf("worktree %s is detached HEAD", source.ID)
	}
	clean, err := ghgit.IsWorktreeClean(ctx, s.Runner, repo.MainWorktreePath)
	if err != nil {
		return "", err
	}
	if !clean {
		return "", fmt.Errorf("local has uncommitted changes; commit, stash, or clean it before handoff back")
	}
	if err := moveBranch(ctx, s.Runner, repo.CommonDir, source.Path, repo.MainWorktreePath, branch, ""); err != nil {
		return "", err
	}
	if err := s.Worktrees.Touch(repo.CommonDir, source.ID); err != nil {
		return "", err
	}
	return repo.MainWorktreePath, nil
}

func moveBranch(ctx context.Context, runner ghgit.Runner, commonDir, sourcePath, targetPath, branch, sourceFallbackBranch string) (err error) {
	lock, err := AcquireLock(ctx, commonDir)
	if err != nil {
		return err
	}
	defer func() {
		releaseErr := lock.Release()
		if err == nil {
			err = releaseErr
		}
	}()

	targetClean, err := ghgit.IsWorktreeClean(ctx, runner, targetPath)
	if err != nil {
		return err
	}
	if !targetClean {
		return fmt.Errorf("destination worktree at %s is not clean", targetPath)
	}

	targetBranch, targetDetached, err := ghgit.CurrentBranch(ctx, runner, targetPath)
	if err != nil {
		return err
	}

	stashRef, hasStash, err := ghgit.CreateStash(ctx, runner, sourcePath, "git-handoff")
	if err != nil {
		return err
	}

	rollbackSource := func() error {
		if err := ghgit.CheckoutBranch(ctx, runner, sourcePath, branch); err != nil {
			return err
		}
		if hasStash {
			if err := ghgit.ApplyStash(ctx, runner, sourcePath, stashRef); err != nil {
				return err
			}
			return ghgit.DropStash(ctx, runner, sourcePath, stashRef)
		}
		return nil
	}

	if err := ghgit.DetachHead(ctx, runner, sourcePath); err != nil {
		if hasStash {
			_ = ghgit.ApplyStash(ctx, runner, sourcePath, stashRef)
			_ = ghgit.DropStash(ctx, runner, sourcePath, stashRef)
		}
		return err
	}

	if err := ghgit.CheckoutBranch(ctx, runner, targetPath, branch); err != nil {
		if rollbackErr := rollbackSource(); rollbackErr != nil {
			return fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
		}
		return err
	}

	if !hasStash {
		return restoreSourceAfterMove(ctx, runner, sourcePath, sourceFallbackBranch)
	}

	if err := ghgit.ApplyStash(ctx, runner, targetPath, stashRef); err != nil {
		restoreErr := restoreTarget(ctx, runner, targetPath, targetBranch, targetDetached)
		rollbackErr := rollbackSource()
		if restoreErr != nil || rollbackErr != nil {
			return fmt.Errorf("%w; restore target: %v; rollback source: %v", err, restoreErr, rollbackErr)
		}
		return err
	}

	if err := ghgit.DropStash(ctx, runner, targetPath, stashRef); err != nil {
		return err
	}
	return restoreSourceAfterMove(ctx, runner, sourcePath, sourceFallbackBranch)
}

func restoreTarget(ctx context.Context, runner ghgit.Runner, targetPath, branch string, detached bool) error {
	if detached || branch == "" {
		return ghgit.DetachHead(ctx, runner, targetPath)
	}
	return ghgit.CheckoutBranch(ctx, runner, targetPath, branch)
}

func restoreSourceAfterMove(ctx context.Context, runner ghgit.Runner, sourcePath, fallbackBranch string) error {
	if fallbackBranch == "" {
		return nil
	}
	if err := ghgit.CheckoutBranch(ctx, runner, sourcePath, fallbackBranch); err != nil {
		return nil
	}
	return nil
}
