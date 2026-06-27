package cat

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var (
	catBenchDir   string
	catBenchLarge string
)

func TestMain(m *testing.M) {
	setupCatBenchData()
	code := m.Run()
	os.RemoveAll(catBenchDir)
	os.Exit(code)
}

func setupCatBenchData() {
	var err error
	catBenchDir, err = os.MkdirTemp("", "aict-cat-bench-")
	if err != nil {
		panic(err)
	}

	// 100k-line text file
	catBenchLarge = filepath.Join(catBenchDir, "large.txt")
	f, err := os.Create(catBenchLarge)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 100000; i++ {
		fmt.Fprintf(f, "line %d: some content here\n", i)
	}
	f.Close()
}

func BenchmarkCat_Stream_100k(b *testing.B) {
b.ResetTimer()
	for b.Loop() {
		_, _, _, err := readFileContent(catBenchLarge)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCat_BinaryDetect(b *testing.B) {
// Benchmark just the binary-detection scan on a 512-byte header.
	header := make([]byte, 512)
	b.ResetTimer()
	for b.Loop() {
		isBinaryContent(header)
	}
}

func BenchmarkCat_SmallFile(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "small.go")
	os.WriteFile(path, []byte("package main\n\nfunc main() {}\n"), 0644)
	b.ResetTimer()
	for b.Loop() {
		_, _, _, err := readFileContent(path)
		if err != nil {
			b.Fatal(err)
		}
	}
}
