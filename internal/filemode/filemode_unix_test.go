//go:build !windows

package filemode

import (
	"os"
	"testing"
)

func TestUID_RealFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/f.txt"
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	// A file we just created must be owned by the current user; before the
	// *syscall.Stat_t case existed, this returned 0 (root) for everything.
	if got, want := UID(info.Sys()), uint32(os.Getuid()); got != want {
		t.Errorf("UID = %d, want %d", got, want)
	}
	if got, want := GID(info.Sys()), uint32(os.Getgid()); got != want {
		t.Errorf("GID = %d, want %d", got, want)
	}
}
