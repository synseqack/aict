//go:build !(linux && amd64)

package stat

func getBirthSec(_ string) int64 { return 0 }
