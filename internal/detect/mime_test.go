package detect

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestDetectFromFile_EmptyFileIsText(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(empty, nil, 0644); err != nil {
		t.Fatal(err)
	}

	mime, isBinary, err := DetectFromFile(empty)
	if err != nil {
		t.Fatal(err)
	}
	if mime != "text/plain; charset=utf-8" {
		t.Errorf("empty file mime: got %q, want text/plain; charset=utf-8", mime)
	}
	if isBinary {
		t.Errorf("empty file should not be binary")
	}
}

func TestMIME_EmptyFileIsText(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(empty, nil, 0644); err != nil {
		t.Fatal(err)
	}

	if got := MIME(empty); got != "text/plain; charset=utf-8" {
		t.Errorf("empty file mime: got %q, want text/plain; charset=utf-8", got)
	}
}

func TestDetectFromFile_BinaryStillBinary(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin.dat")
	if err := os.WriteFile(bin, []byte{0x00, 0x01, 0x02, 0x03}, 0644); err != nil {
		t.Fatal(err)
	}

	if _, isBinary, err := DetectFromFile(bin); err != nil {
		t.Fatal(err)
	} else if !isBinary {
		t.Errorf("NUL-containing file should be binary")
	}
}

func TestDetectFromFile_UnreadableIsOpaque(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}

	mime, isBinary, err := DetectFromFile(sub)
	if err != nil {
		t.Fatal(err)
	}
	if mime != "application/octet-stream" {
		t.Errorf("unreadable entry mime: got %q, want application/octet-stream", mime)
	}
	if !isBinary {
		t.Errorf("unreadable entry should be binary")
	}
}

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

// A symlink to a regular file is followed: it is sniffed, not classified
// opaque the way a dangling or non-regular path would be.
func TestDetectFromFile_SymlinkIsFollowed(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("plain text\n"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	mime, isBinary, err := DetectFromFile(link)
	if err != nil {
		t.Fatal(err)
	}
	if mime != "text/plain; charset=utf-8" {
		t.Errorf("symlink mime: got %q, want text/plain; charset=utf-8", mime)
	}
	if isBinary {
		t.Error("symlink to a text file should not be binary")
	}
}
