package handoff

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAcquireLockAndRelease(t *testing.T) {
	t.Parallel()

	commonDir := t.TempDir()
	lock, err := AcquireLock(context.Background(), commonDir)
	if err != nil {
		t.Fatalf("AcquireLock failed: %v", err)
	}

	if _, err := AcquireLock(context.Background(), commonDir); err == nil {
		t.Fatalf("expected second AcquireLock to fail")
	}

	lockPath := filepath.Join(commonDir, "git-handoff", "handoff.lock")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("expected lock file to exist: %v", err)
	}

	if err := lock.Release(); err != nil {
		t.Fatalf("Release failed: %v", err)
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected lock file to be removed, got err=%v", err)
	}
}

func TestAcquireLockRemovesStaleLock(t *testing.T) {
	t.Parallel()

	commonDir := t.TempDir()
	lockPath := filepath.Join(commonDir, "git-handoff", "handoff.lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(lockPath, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	old := time.Now().Add(-11 * time.Minute)
	if err := os.Chtimes(lockPath, old, old); err != nil {
		t.Fatalf("Chtimes failed: %v", err)
	}

	lock, err := AcquireLock(context.Background(), commonDir)
	if err != nil {
		t.Fatalf("AcquireLock with stale file failed: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("Release failed: %v", err)
	}
}
