package pathutil

import (
	"io/fs"
	"os"
	"path/filepath"
)

type Resolved struct {
	Given    string
	Absolute string
}

func Resolve(input string) (Resolved, error) {
	cleaned := filepath.Clean(input)
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return Resolved{}, err
	}
	return Resolved{
		Given:    input,
		Absolute: abs,
	}, nil
}

func ResolveSymlink(path string) (target string, targetAbs string, exists bool, err error) {
	target, err = os.Readlink(path)
	if err != nil {
		return "", "", false, err
	}

	if !filepath.IsAbs(target) {
		dir := filepath.Dir(path)
		targetAbs = filepath.Join(dir, target)
	} else {
		targetAbs = target
	}

	targetAbs = filepath.Clean(targetAbs)

	_, err = os.Lstat(targetAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return target, targetAbs, false, nil
		}
		return "", "", false, err
	}

	return target, targetAbs, true, nil
}

func Exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func Type(path string) string {
	info, err := os.Lstat(path)
	if err != nil {
		return "unknown"
	}

	mode := info.Mode()

	if mode&fs.ModeSymlink != 0 {
		return "symlink"
	}
	if mode.IsDir() {
		return "directory"
	}
	if mode.IsRegular() {
		return "file"
	}

	return "unknown"
}
