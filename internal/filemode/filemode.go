package filemode

import (
	"io/fs"
	"os"
	"strconv"
	"strings"
)

// FormatPermissions returns a 10-character Unix permission string (e.g. "-rwxr-xr-x").
func FormatPermissions(mode os.FileMode, isDir bool, isSymlink bool) string {
	var b strings.Builder
	b.Grow(10)

	if isSymlink {
		b.WriteByte('l')
	} else if isDir {
		b.WriteByte('d')
	} else {
		b.WriteByte('-')
	}

	for i := 8; i >= 0; i-- {
		bit := uint(1) << uint(i)
		switch {
		case mode&os.FileMode(bit) != 0:
			switch i % 3 {
			case 0:
				b.WriteByte('x')
			case 1:
				b.WriteByte('w')
			case 2:
				b.WriteByte('r')
			}
		default:
			b.WriteByte('-')
		}
	}

	return b.String()
}

// ModeOctal returns the octal mode string (e.g. "0755").
func ModeOctal(mode os.FileMode) string {
	return "0" + strconv.FormatUint(uint64(mode.Perm()), 8)
}

// FileType returns a string describing the kind of file.
func FileType(info os.FileInfo) string {
	mode := info.Mode()
	if mode&os.ModeSymlink != 0 {
		return "symlink"
	}
	if mode.IsDir() {
		return "directory"
	}
	if mode.IsRegular() {
		return "file"
	}
	if mode&os.ModeDevice != 0 {
		return "block"
	}
	if mode&os.ModeCharDevice != 0 {
		return "character"
	}
	if mode&os.ModeNamedPipe != 0 {
		return "pipe"
	}
	if mode&os.ModeSocket != 0 {
		return "socket"
	}
	return "unknown"
}

// IsSymlink reports whether the FileMode indicates a symlink.
func IsSymlink(mode fs.FileMode) bool {
	return mode&fs.ModeSymlink != 0
}
