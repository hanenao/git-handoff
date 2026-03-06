package handoff

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type lockMetadata struct {
	PID       int       `json:"pid"`
	CreatedAt time.Time `json:"created_at"`
}

type Lock struct {
	path string
}

func AcquireLock(_ context.Context, commonDir string) (*Lock, error) {
	lockPath := filepath.Join(commonDir, "git-handoff", "handoff.lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, err
	}
	if err := removeStaleLock(lockPath); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("another handoff is already running")
		}
		return nil, err
	}
	payload, err := json.Marshal(lockMetadata{
		PID:       os.Getpid(),
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if _, err := file.Write(append(payload, '\n')); err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return &Lock{path: lockPath}, nil
}

func (l *Lock) Release() error {
	if l == nil || l.path == "" {
		return nil
	}
	return os.Remove(l.path)
}

func removeStaleLock(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if time.Since(info.ModTime()) < 10*time.Minute {
		return nil
	}
	return os.Remove(path)
}
