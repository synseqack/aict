//go:build windows

package df

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/synseqack/aict/internal/format"
)

var (
	modkernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetLogicalDrives  = modkernel32.NewProc("GetLogicalDrives")
	procGetDriveTypeW     = modkernel32.NewProc("GetDriveTypeW")
	procGetDiskFreeSpaceW = modkernel32.NewProc("GetDiskFreeSpaceExW")
	procGetVolumeInfoW    = modkernel32.NewProc("GetVolumeInformationW")
)

const driveFixed = 3 // DRIVE_FIXED

func getMounts() ([]mountInfo, error) {
	ret, _, err := procGetLogicalDrives.Call()
	if ret == 0 {
		return nil, fmt.Errorf("GetLogicalDrives: %w", err)
	}

	mask := uint32(ret)
	var mounts []mountInfo
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		letter := string(rune('A' + i))
		root := letter + `:\`
		rootPtr, _ := syscall.UTF16PtrFromString(root)
		driveType, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(rootPtr)))
		if driveType != driveFixed {
			continue
		}
		mounts = append(mounts, mountInfo{
			Device:     letter + ":",
			Mountpoint: root,
			Fstype:     volumeFsType(root),
		})
	}
	return mounts, nil
}

func volumeFsType(root string) string {
	rootPtr, _ := syscall.UTF16PtrFromString(root)
	buf := make([]uint16, 256)
	ret, _, _ := procGetVolumeInfoW.Call(
		uintptr(unsafe.Pointer(rootPtr)),
		0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if ret == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func getFsInfo(device, mount, fstype string) (FsEntry, error) {
	rootPtr, err := syscall.UTF16PtrFromString(mount)
	if err != nil {
		return FsEntry{}, err
	}

	var freeBytes, totalBytes uint64
	ret, _, callErr := procGetDiskFreeSpaceW.Call(
		uintptr(unsafe.Pointer(rootPtr)),
		uintptr(unsafe.Pointer(&freeBytes)),
		uintptr(unsafe.Pointer(&totalBytes)),
		0, // lpTotalNumberOfFreeBytes not needed
	)
	if ret == 0 {
		return FsEntry{}, fmt.Errorf("GetDiskFreeSpaceEx %s: %w", mount, callErr)
	}

	total := int64(totalBytes)
	avail := int64(freeBytes)
	used := total - avail
	usePct := 0
	if total > 0 {
		usePct = int((float64(used) / float64(total)) * 100)
	}

	return FsEntry{
		Device:     device,
		Mount:      mount,
		Type:       fstype,
		SizeBytes:  total,
		SizeHuman:  format.Size(uint64(total)),
		UsedBytes:  used,
		UsedHuman:  format.Size(uint64(used)),
		AvailBytes: avail,
		AvailHuman: format.Size(uint64(avail)),
		UsePct:     usePct,
		// Windows does not expose inode stats
	}, nil
}
