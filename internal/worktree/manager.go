package worktree

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	ghgit "github.com/hanenao/git-handoff/internal/git"
)

type State string

const (
	StateIdle     State = "idle"
	StateAttached State = "attached"
)

type Environment struct {
	ID        string
	Path      string
	Branch    string
	State     State
	IsCurrent bool
	UpdatedAt time.Time
}

type ListRow struct {
	Kind      string
	ID        string
	State     string
	Branch    string
	Path      string
	UpdatedAt time.Time
	IsCurrent bool
}

type Manager struct {
	Runner ghgit.Runner
}

func NewManager(runner ghgit.Runner) *Manager {
	return &Manager{Runner: runner}
}

func (m *Manager) Exists(ctx context.Context, commonDir, id string) (bool, error) {
	_, err := ReadMetadata(commonDir, id)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (m *Manager) Create(ctx context.Context, repo *ghgit.RepoContext, path string) (Metadata, error) {
	id, err := GenerateID(ctx, func(ctx context.Context, value string) (bool, error) {
		_, err := ReadMetadata(repo.CommonDir, value)
		if err == nil {
			return true, nil
		}
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	})
	if err != nil {
		return Metadata{}, err
	}
	now := time.Now().UTC()
	metadata := Metadata{
		ID:        id,
		Path:      path,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return metadata, WriteMetadata(repo.CommonDir, metadata)
}

func (m *Manager) Resolve(ctx context.Context, repo *ghgit.RepoContext, id string) (Environment, Metadata, error) {
	metadata, err := ReadMetadata(repo.CommonDir, id)
	if err != nil {
		return Environment{}, Metadata{}, err
	}
	worktrees, err := ghgit.ListWorktrees(ctx, m.Runner, repo.CurrentWorktreePath)
	if err != nil {
		return Environment{}, Metadata{}, err
	}
	path, err := ghgit.EvalPath(metadata.Path)
	if err != nil {
		return Environment{}, Metadata{}, err
	}
	for _, wt := range worktrees {
		wtPath, err := ghgit.EvalPath(wt.Path)
		if err != nil {
			return Environment{}, Metadata{}, err
		}
		if wtPath == path {
			return environmentFrom(metadata, wt, repo.CurrentWorktreePath), metadata, nil
		}
	}
	return Environment{}, Metadata{}, fmt.Errorf("worktree %s does not exist at %s", id, metadata.Path)
}

func (m *Manager) ResolveCurrentBackground(ctx context.Context, repo *ghgit.RepoContext) (Environment, Metadata, error) {
	environments, metadata, err := m.List(ctx, repo)
	if err != nil {
		return Environment{}, Metadata{}, err
	}
	for i := range environments {
		if environments[i].IsCurrent {
			return environments[i], metadata[environments[i].ID], nil
		}
	}
	return Environment{}, Metadata{}, fmt.Errorf("current worktree is not a git-ho worktree")
}

func (m *Manager) List(ctx context.Context, repo *ghgit.RepoContext) ([]Environment, map[string]Metadata, error) {
	worktrees, err := ghgit.ListWorktrees(ctx, m.Runner, repo.CurrentWorktreePath)
	if err != nil {
		return nil, nil, err
	}
	metadataList, err := ReadAllMetadata(repo.CommonDir)
	if err != nil {
		return nil, nil, err
	}
	metadataByPath := make(map[string]Metadata, len(metadataList))
	metadataByID := make(map[string]Metadata, len(metadataList))
	for _, item := range metadataList {
		path, err := ghgit.EvalPath(item.Path)
		if err != nil {
			return nil, nil, err
		}
		item.Path = path
		metadataByPath[path] = item
		metadataByID[item.ID] = item
	}

	environments := make([]Environment, 0, len(metadataList))
	for _, wt := range worktrees {
		path, err := ghgit.EvalPath(wt.Path)
		if err != nil {
			return nil, nil, err
		}
		item, ok := metadataByPath[path]
		if !ok {
			continue
		}
		environments = append(environments, environmentFrom(item, wt, repo.CurrentWorktreePath))
	}
	sort.Slice(environments, func(i, j int) bool {
		return environments[i].ID < environments[j].ID
	})
	return environments, metadataByID, nil
}

func (m *Manager) SelectIdle(ctx context.Context, repo *ghgit.RepoContext) (Environment, Metadata, error) {
	environments, metadata, err := m.List(ctx, repo)
	if err != nil {
		return Environment{}, Metadata{}, err
	}
	for _, environment := range environments {
		if environment.State == StateIdle {
			return environment, metadata[environment.ID], nil
		}
	}
	return Environment{}, Metadata{}, fmt.Errorf("no idle worktree is available")
}

func (m *Manager) Touch(commonDir, id string) error {
	metadata, err := ReadMetadata(commonDir, id)
	if err != nil {
		return err
	}
	metadata.UpdatedAt = time.Now().UTC()
	return WriteMetadata(commonDir, metadata)
}

func (m *Manager) Rows(ctx context.Context, repo *ghgit.RepoContext) ([]ListRow, error) {
	worktrees, err := ghgit.ListWorktrees(ctx, m.Runner, repo.CurrentWorktreePath)
	if err != nil {
		return nil, err
	}
	environments, metadata, err := m.List(ctx, repo)
	if err != nil {
		return nil, err
	}

	rows := make([]ListRow, 0, len(environments)+1)
	for _, wt := range worktrees {
		path, err := ghgit.EvalPath(wt.Path)
		if err != nil {
			return nil, err
		}
		if path != repo.MainWorktreePath {
			continue
		}
		rows = append(rows, ListRow{
			Kind:      "local",
			ID:        "-",
			State:     renderState(wt),
			Branch:    renderBranch(wt),
			Path:      path,
			UpdatedAt: time.Time{},
			IsCurrent: path == repo.CurrentWorktreePath,
		})
		break
	}
	for _, environment := range environments {
		rows = append(rows, ListRow{
			Kind:      "worktree",
			ID:        environment.ID,
			State:     string(environment.State),
			Branch:    environment.Branch,
			Path:      environment.Path,
			UpdatedAt: metadata[environment.ID].UpdatedAt,
			IsCurrent: environment.IsCurrent,
		})
	}
	return rows, nil
}

func environmentFrom(metadata Metadata, wt ghgit.Worktree, currentPath string) Environment {
	branch := renderBranch(wt)
	state := StateIdle
	if !wt.Detached && wt.BranchName != "" {
		state = StateAttached
	}
	return Environment{
		ID:        metadata.ID,
		Path:      metadata.Path,
		Branch:    branch,
		State:     state,
		IsCurrent: filepath.Clean(metadata.Path) == filepath.Clean(currentPath),
		UpdatedAt: metadata.UpdatedAt,
	}
}

func renderBranch(wt ghgit.Worktree) string {
	if wt.Detached || wt.BranchName == "" {
		return "[detached]"
	}
	return wt.BranchName
}

func renderState(wt ghgit.Worktree) string {
	if wt.Detached || wt.BranchName == "" {
		return string(StateIdle)
	}
	return string(StateAttached)
}
