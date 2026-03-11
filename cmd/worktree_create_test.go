package cmd

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

type notifyingBuffer struct {
	bytes.Buffer
	wrote chan struct{}
	once  sync.Once
}

func newNotifyingBuffer() *notifyingBuffer {
	return &notifyingBuffer{wrote: make(chan struct{})}
}

func (b *notifyingBuffer) Write(p []byte) (int, error) {
	n, err := b.Buffer.Write(p)
	if n > 0 {
		b.once.Do(func() {
			close(b.wrote)
		})
	}
	return n, err
}

func TestRunHooksStreamsOutput(t *testing.T) {
	t.Parallel()

	stdout := newNotifyingBuffer()
	done := make(chan error, 1)

	go func() {
		done <- runHooks(
			context.Background(),
			t.TempDir(),
			[]string{"printf 'hook stdout\\n'; sleep 1"},
			stdout,
			io.Discard,
		)
	}()

	select {
	case <-stdout.wrote:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("hook stdout was not streamed before hook completion")
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runHooks returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runHooks did not finish")
	}

	if got := stdout.String(); got != "hook stdout\n" {
		t.Fatalf("unexpected stdout: %q", got)
	}
}

func TestRunHooksReturnsOutputOnFailure(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := runHooks(
		context.Background(),
		t.TempDir(),
		[]string{"printf 'hook stdout\\n'; printf 'hook stderr\\n' >&2; exit 1"},
		&stdout,
		&stderr,
	)
	if err == nil {
		t.Fatal("runHooks unexpectedly succeeded")
	}

	if got := stdout.String(); got != "hook stdout\n" {
		t.Fatalf("unexpected stdout: %q", got)
	}
	if got := stderr.String(); got != "hook stderr\n" {
		t.Fatalf("unexpected stderr: %q", got)
	}

	message := err.Error()
	if !strings.Contains(message, "hook stdout") || !strings.Contains(message, "hook stderr") {
		t.Fatalf("unexpected error message: %q", message)
	}
}
