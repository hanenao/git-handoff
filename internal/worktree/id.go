package worktree

import (
	"context"
	"crypto/rand"
	"fmt"
)

const idAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

func GenerateID(ctx context.Context, exists func(context.Context, string) (bool, error)) (string, error) {
	const size = 6
	for range 32 {
		value, err := randomString(size)
		if err != nil {
			return "", err
		}
		taken, err := exists(ctx, value)
		if err != nil {
			return "", err
		}
		if !taken {
			return value, nil
		}
	}
	return "", fmt.Errorf("failed to allocate unique worktree id")
}

func randomString(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, size)
	for i, b := range buf {
		out[i] = idAlphabet[int(b)%len(idAlphabet)]
	}
	return string(out), nil
}
