//go:build !windows

package ls

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A FIFO with no writer makes os.Open block until one appears, so ls must not
// run content detection on non-regular files. The whole listing has to return.
func TestLS_FIFODoesNotHang(t *testing.T) {
	dir := t.TempDir()
	pipe := filepath.Join(dir, "pipe")
	if err := syscall.Mkfifo(pipe, 0644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}

	type outcome struct {
		res *LSResult
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		res, err := runLSWithOutput([]string{dir}, Config{XML: true})
		done <- outcome{res, err}
	}()

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("ls on a directory containing a FIFO failed: %v", got.err)
		}
		if got.res == nil {
			t.Fatal("ls returned no result")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ls hung on a directory containing a FIFO")
	}
}
