package detect

import (
	"os"
	"path/filepath"
	"testing"
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
