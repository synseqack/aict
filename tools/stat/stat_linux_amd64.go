//go:build linux && amd64

package stat

import (
	"syscall"
	"unsafe"
)

type statxTimestamp struct {
	Sec  int64
	Nsec uint32
	_    int32
}

type statxT struct {
	Mask           uint32
	Blksize        uint32
	Attributes     uint64
	Nlink          uint32
	UID            uint32
	GID            uint32
	Mode           uint16
	_              [1]uint16
	Ino            uint64
	Size           uint64
	Blocks         uint64
	AttributesMask uint64
	Atime          statxTimestamp
	Btime          statxTimestamp
	Ctime          statxTimestamp
	Mtime          statxTimestamp
	RdevMajor      uint32
	RdevMinor      uint32
	DevMajor       uint32
	DevMinor       uint32
	_              [14]uint64
}

// getBirthSec returns the birth time (btime) using statx(2).
// Returns 0 if the filesystem doesn't support birth time.
func getBirthSec(path string) int64 {
	const sysStatx = 332        // __NR_statx on x86_64
	const statxBtime = 0x800    // STATX_BTIME
	atFdcwd := -100             // AT_FDCWD — must be a variable, not constant, for uintptr cast

	pathBytes, err := syscall.BytePtrFromString(path)
	if err != nil {
		return 0
	}

	var sx statxT
	_, _, errno := syscall.Syscall6(
		sysStatx,
		uintptr(atFdcwd),
		uintptr(unsafe.Pointer(pathBytes)),
		0,
		statxBtime,
		uintptr(unsafe.Pointer(&sx)),
		0,
	)
	if errno != 0 {
		return 0
	}
	if sx.Mask&statxBtime == 0 {
		return 0
	}
	return sx.Btime.Sec
}
