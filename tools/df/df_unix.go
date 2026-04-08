//go:build linux || darwin

package df

import (
	"os"
	"syscall"

	"github.com/synseqack/aict/internal/format"
)

func getMounts() ([]mountInfo, error) {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return nil, err
	}

	var mounts []mountInfo
	for _, line := range splitLines(string(data)) {
		fields := splitFields(line)
		if len(fields) < 3 {
			continue
		}
		mounts = append(mounts, mountInfo{
			Device:     fields[0],
			Mountpoint: fields[1],
			Fstype:     fields[2],
		})
	}
	return mounts, nil
}

func getFsInfo(device, mount, fstype string) (FsEntry, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(mount, &stat)
	if err != nil {
		return FsEntry{}, err
	}

	total := int64(stat.Blocks) * int64(stat.Bsize)
	avail := int64(stat.Bavail) * int64(stat.Bsize)
	used := total - avail
	usePct := 0
	if total > 0 {
		usePct = int((float64(used) / float64(total)) * 100)
	}

	inodesTotal := int64(stat.Files)
	inodesAvail := int64(stat.Ffree)
	inodesUsed := inodesTotal - inodesAvail
	inodesPct := 0
	if inodesTotal > 0 {
		inodesPct = int((float64(inodesUsed) / float64(inodesTotal)) * 100)
	}

	entry := FsEntry{
		Device:      device,
		Mount:       mount,
		Type:        fstype,
		SizeBytes:   total,
		SizeHuman:   format.Size(uint64(total)),
		UsedBytes:   used,
		UsedHuman:   format.Size(uint64(used)),
		AvailBytes:  avail,
		AvailHuman:  format.Size(uint64(avail)),
		UsePct:      usePct,
		InodesTotal: inodesTotal,
		InodesUsed:  inodesUsed,
		InodesAvail: inodesAvail,
		InodesPct:   inodesPct,
	}

	return entry, nil
}
