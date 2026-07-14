//go:build linux

package df

import "os"

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
