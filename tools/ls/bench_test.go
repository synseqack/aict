package ls

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var lsBenchDir string

func TestMain(m *testing.M) {
	setupLSBenchData()
	code := m.Run()
	os.RemoveAll(lsBenchDir)
	os.Exit(code)
}

func setupLSBenchData() {
	var err error
	lsBenchDir, err = os.MkdirTemp("", "aict-ls-bench-")
	if err != nil {
		panic(err)
	}

	for i := 0; i < 1000; i++ {
		path := filepath.Join(lsBenchDir, fmt.Sprintf("file_%d.go", i))
		content := fmt.Sprintf("package test%d\nfunc Foo() {}\n", i)
		os.WriteFile(path, []byte(content), 0644)
	}
}

// BenchmarkLS_WithDetection_1000 measures the full ls path including MIME and
// language detection — the primary overhead vs GNU ls.
func BenchmarkLS_WithDetection_1000(b *testing.B) {
	cfg := Config{}
	b.ResetTimer()
	for b.Loop() {
		_, err := listDir(lsBenchDir, cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLS_Plain_1000 measures ls in plain mode which skips
// MIME/language detection, isolating the filesystem overhead.
func BenchmarkLS_Plain_1000(b *testing.B) {
	cfg := Config{Plain: true}
	b.ResetTimer()
	for b.Loop() {
		_, err := listDir(lsBenchDir, cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}
