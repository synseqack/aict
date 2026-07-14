//go:build windows

package filemode

// UID has no meaningful value on windows; Sys() returns file attribute data
// without ownership. Callers fall back to the numeric string.
func UID(sysInfo any) uint32 {
	switch v := sysInfo.(type) {
	case interface{ Uid() uint32 }:
		return v.Uid()
	case interface{ UID() uint32 }:
		return v.UID()
	default:
		return 0
	}
}

// GID mirrors UID for group ownership.
func GID(sysInfo any) uint32 {
	switch v := sysInfo.(type) {
	case interface{ Gid() uint32 }:
		return v.Gid()
	case interface{ GID() uint32 }:
		return v.GID()
	default:
		return 0
	}
}
