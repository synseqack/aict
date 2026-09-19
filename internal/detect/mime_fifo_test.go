//go:build !windows

package detect

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A FIFO opens fine and then blocks on read forever, waiting for a writer
// that never comes. Detection must classify it without reading.
func TestDetectFromFile_FIFODoesNotBlock(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "pipe")
	if err := syscall.Mkfifo(fifo, 0644); err != nil {
		t.Skipf("cannot create fifo: %v", err)
	}

	type result struct {
		mime     string
		isBinary bool
		err      error
	}
	done := make(chan result, 1)
	go func() {
		mime, isBinary, err := DetectFromFile(fifo)
		done <- result{mime, isBinary, err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("fifo: unexpected error %v", r.err)
		}
		if r.mime != "application/octet-stream" {
			t.Errorf("fifo mime: got %q, want application/octet-stream", r.mime)
		}
		if !r.isBinary {
			t.Error("fifo should be reported binary")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("DetectFromFile blocked on a fifo; it must not read non-regular files")
	}

	if got := MIME(fifo); got != "application/octet-stream" {
		t.Errorf("MIME on a fifo: got %q, want application/octet-stream", got)
	}
}
