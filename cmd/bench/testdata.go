package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func setupTestData() (dir string, cleanup func(), err error) {
	dir, err = os.MkdirTemp("", "aict-bench-")
	if err != nil {
		return "", nil, err
	}

	cleanup = func() { os.RemoveAll(dir) }

	// 1000 Go source files
	for i := 0; i < 1000; i++ {
		path := filepath.Join(dir, fmt.Sprintf("file_%d.go", i))
		content := fmt.Sprintf("package test%d\nfunc Foo() {}\n", i)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			cleanup()
			return "", nil, err
		}
	}

	// Deep nested tree for find
	subdir := filepath.Join(dir, "src", "pkg", "util")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		cleanup()
		return "", nil, err
	}
	for i := 0; i < 100; i++ {
		path := filepath.Join(subdir, fmt.Sprintf("deep_%d.txt", i))
		if err := os.WriteFile(path, []byte(strings.Repeat("line\n", 1000)), 0644); err != nil {
			cleanup()
			return "", nil, err
		}
	}

	// 100k-line file for grep/cat/wc
	largeFile := filepath.Join(dir, "large_file.txt")
	f, err := os.Create(largeFile)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	for i := 0; i < 100000; i++ {
		fmt.Fprintf(f, "line %d: some search text here\n", i)
	}
	f.Close()

	// 10k-line CSV for sed/awk
	csvFile := filepath.Join(dir, "data.csv")
	g, err := os.Create(csvFile)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	for i := 0; i < 10000; i++ {
		fmt.Fprintf(g, "user%d,score%d,region%d\n", i%100, i%50, i%10)
	}
	g.Close()

	// Near-identical files for diff
	f1, err := os.Create(filepath.Join(dir, "f1.txt"))
	if err != nil {
		cleanup()
		return "", nil, err
	}
	f2, err := os.Create(filepath.Join(dir, "f2.txt"))
	if err != nil {
		cleanup()
		return "", nil, err
	}
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(f1, "line %d: content\n", i)
		fmt.Fprintf(f2, "line %d: content\n", i)
	}
	fmt.Fprintln(f2, "added line")
	f1.Close()
	f2.Close()

	return dir, cleanup, nil
}
