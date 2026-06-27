package grep

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	benchDir     string
	benchLarge   string
	bench1kFiles string
)

func TestMain(m *testing.M) {
	setupBenchData()
	code := m.Run()
	os.RemoveAll(benchDir)
	os.Exit(code)
}

func setupBenchData() {
	var err error
	benchDir, err = os.MkdirTemp("", "aict-grep-bench-")
	if err != nil {
		panic(err)
	}

	// 100k-line file
	benchLarge = filepath.Join(benchDir, "large.txt")
	f, err := os.Create(benchLarge)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 100000; i++ {
		fmt.Fprintf(f, "line %d: some search text here\n", i)
	}
	f.Close()

	// 1000 small Go files for recursive search
	bench1kFiles = filepath.Join(benchDir, "src")
	os.MkdirAll(bench1kFiles, 0755)
	for i := 0; i < 1000; i++ {
		path := filepath.Join(bench1kFiles, fmt.Sprintf("file_%d.go", i))
		content := fmt.Sprintf("package test\n// line %d\nfunc Foo%d() { }\n", i, i)
		os.WriteFile(path, []byte(content), 0644)
	}
}

func BenchmarkGrep_Regex_100k(b *testing.B) {
	re, err := buildRegexp("search", Config{})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		findMatches(benchLarge, re, Config{})
	}
}

func BenchmarkGrep_Literal_100k(b *testing.B) {
	// Fixed-string path: QuoteMeta makes regexp treat pattern as literal.
	// This shows how close we can get without a true strings.Contains fast path.
	re, err := buildRegexp("search", Config{FixedStrings: true})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		findMatches(benchLarge, re, Config{FixedStrings: true})
	}
}

func BenchmarkGrep_StringsContains_100k(b *testing.B) {
	// Baseline: what a raw strings.Contains scan would cost (no regexp overhead).
	data, err := os.ReadFile(benchLarge)
	if err != nil {
		b.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	b.ResetTimer()
	for b.Loop() {
		var count int
		for _, line := range lines {
			if strings.Contains(line, "search") {
				count++
			}
		}
		_ = count
	}
}

func BenchmarkGrep_Recursive_1000(b *testing.B) {
	cfg := Config{
		Pattern:   "Foo",
		Recursive: true,
	}
	b.ResetTimer()
	for b.Loop() {
		searchDirectory(bench1kFiles, bench1kFiles, cfg)
	}
}
