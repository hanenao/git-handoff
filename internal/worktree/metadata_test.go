package worktree

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetadataRoundTrip(t *testing.T) {
	t.Parallel()

	commonDir := t.TempDir()
	now := time.Now().UTC().Round(time.Second)
	metadata := Metadata{
		ID:        "abc123",
		Path:      "/tmp/worktrees/abc123",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := WriteMetadata(commonDir, metadata); err != nil {
		t.Fatalf("WriteMetadata failed: %v", err)
	}

	got, err := ReadMetadata(commonDir, metadata.ID)
	if err != nil {
		t.Fatalf("ReadMetadata failed: %v", err)
	}
	if got != metadata {
		t.Fatalf("unexpected metadata: %+v", got)
	}

	all, err := ReadAllMetadata(commonDir)
	if err != nil {
		t.Fatalf("ReadAllMetadata failed: %v", err)
	}
	if len(all) != 1 || all[0] != metadata {
		t.Fatalf("unexpected metadata list: %+v", all)
	}

	if err := DeleteMetadata(commonDir, metadata.ID); err != nil {
		t.Fatalf("DeleteMetadata failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(MetadataDir(commonDir), metadata.ID+".json")); !os.IsNotExist(err) {
		t.Fatalf("expected metadata file to be removed, got err=%v", err)
	}
}

func TestReadMetadataFallsBackToLegacyDirectory(t *testing.T) {
	t.Parallel()

	commonDir := t.TempDir()
	now := time.Now().UTC().Round(time.Second)
	metadata := Metadata{
		ID:        "legacy01",
		Path:      "/tmp/legacy/worktree",
		CreatedAt: now,
		UpdatedAt: now,
	}
	payload, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent failed: %v", err)
	}
	legacyPath := filepath.Join(legacyMetadataDir(commonDir), metadata.ID+".json")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(legacyPath, append(payload, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	got, err := ReadMetadata(commonDir, metadata.ID)
	if err != nil {
		t.Fatalf("ReadMetadata failed: %v", err)
	}
	if got != metadata {
		t.Fatalf("unexpected metadata: %+v", got)
	}
}
