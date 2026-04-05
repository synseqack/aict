package format

import (
	"fmt"
	"math"
)

var units = []string{"K", "M", "G", "T", "P", "E"}

func Size(bytes uint64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d", bytes)
	}

	exp := int(math.Log(float64(bytes)) / math.Log(1024))
	if exp > len(units) {
		exp = len(units)
	}

	val := float64(bytes) / math.Pow(1024, float64(exp))

	if val >= 10 {
		return fmt.Sprintf("%.0f%s", val, units[exp-1])
	}
	return fmt.Sprintf("%.1f%s", val, units[exp-1])
}

func SizeWithUnit(bytes uint64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}

	exp := int(math.Log(float64(bytes)) / math.Log(1024))
	if exp > len(units) {
		exp = len(units)
	}

	val := float64(bytes) / math.Pow(1024, float64(exp))

	if val >= 10 {
		return fmt.Sprintf("%.0f%s", val, units[exp-1])
	}
	return fmt.Sprintf("%.1f%s", val, units[exp-1])
}
