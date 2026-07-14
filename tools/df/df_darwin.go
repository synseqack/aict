//go:build darwin

package df

import "syscall"

const mntNoWait = 2 // MNT_NOWAIT

func getMounts() ([]mountInfo, error) {
	n, err := syscall.Getfsstat(nil, mntNoWait)
	if err != nil {
		return nil, err
	}

	buf := make([]syscall.Statfs_t, n)
	n, err = syscall.Getfsstat(buf, mntNoWait)
	if err != nil {
		return nil, err
	}

	mounts := make([]mountInfo, 0, n)
	for _, s := range buf[:n] {
		mounts = append(mounts, mountInfo{
			Device:     cString(s.Mntfromname[:]),
			Mountpoint: cString(s.Mntonname[:]),
			Fstype:     cString(s.Fstypename[:]),
		})
	}
	return mounts, nil
}

func cString(b []int8) string {
	out := make([]byte, 0, len(b))
	for _, c := range b {
		if c == 0 {
			break
		}
		out = append(out, byte(c))
	}
	return string(out)
}
