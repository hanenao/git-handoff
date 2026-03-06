package worktree

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Metadata struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func MetadataDir(commonDir string) string {
	return filepath.Join(commonDir, "git-handoff", "worktrees")
}

func legacyMetadataDir(commonDir string) string {
	return filepath.Join(commonDir, "git-handoff", "environments")
}

func metadataPath(dir, id string) string {
	return filepath.Join(dir, id+".json")
}

func MetadataPath(commonDir, id string) string {
	return metadataPath(MetadataDir(commonDir), id)
}

func WriteMetadata(commonDir string, metadata Metadata) error {
	if metadata.ID == "" {
		return fmt.Errorf("worktree id is required")
	}
	if err := os.MkdirAll(MetadataDir(commonDir), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(MetadataPath(commonDir, metadata.ID), append(payload, '\n'), 0o644)
}

func ReadMetadata(commonDir, id string) (Metadata, error) {
	var metadata Metadata
	for _, path := range []string{MetadataPath(commonDir, id), metadataPath(legacyMetadataDir(commonDir), id)} {
		payload, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return metadata, err
		}
		if err := json.Unmarshal(payload, &metadata); err != nil {
			return metadata, err
		}
		return metadata, nil
	}
	return metadata, os.ErrNotExist
}

func DeleteMetadata(commonDir, id string) error {
	var found bool
	for _, path := range []string{MetadataPath(commonDir, id), metadataPath(legacyMetadataDir(commonDir), id)} {
		err := os.Remove(path)
		if err == nil {
			found = true
			continue
		}
		if os.IsNotExist(err) {
			continue
		}
		return err
	}
	if !found {
		return os.ErrNotExist
	}
	return nil
}

func ReadAllMetadata(commonDir string) ([]Metadata, error) {
	metadataByID := make(map[string]Metadata)
	for _, dir := range []string{legacyMetadataDir(commonDir), MetadataDir(commonDir)} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			item, err := readMetadataFromPath(filepath.Join(dir, entry.Name()))
			if err != nil {
				return nil, err
			}
			metadataByID[item.ID] = item
		}
	}
	metadata := make([]Metadata, 0, len(metadataByID))
	for _, item := range metadataByID {
		metadata = append(metadata, item)
	}
	return metadata, nil
}

func readMetadataFromPath(path string) (Metadata, error) {
	var metadata Metadata
	payload, err := os.ReadFile(path)
	if err != nil {
		return metadata, err
	}
	if err := json.Unmarshal(payload, &metadata); err != nil {
		return metadata, err
	}
	if metadata.ID == "" {
		return metadata, errors.New("metadata id is required")
	}
	return metadata, nil
}
