//go:build !windows

package filemode

import "syscall"

// UID extracts the numeric UID from an fs.FileInfo Sys() value.
// On unix platforms Sys() returns *syscall.Stat_t, whose Uid is a field,
// not a method — the struct check must come first.
func UID(sysInfo any) uint32 {
	switch v := sysInfo.(type) {
	case *syscall.Stat_t:
		return v.Uid
	case interface{ Uid() uint32 }:
		return v.Uid()
	case interface{ UID() uint32 }:
		return v.UID()
	default:
		return 0
	}
}

// GID extracts the numeric GID from an fs.FileInfo Sys() value.
func GID(sysInfo any) uint32 {
	switch v := sysInfo.(type) {
	case *syscall.Stat_t:
		return v.Gid
	case interface{ Gid() uint32 }:
		return v.Gid()
	case interface{ GID() uint32 }:
		return v.GID()
	default:
		return 0
	}
}
