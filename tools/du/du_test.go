package du

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
)

func runDu(args []string) (string, error) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run(args)

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)
	return outBuf.String(), err
}

func runDuWithOutput(path string, cfg Config) (*DuResult, error) {
	cfg.XML = true
	entries, total, err := calculateDu(path, cfg)
	if err != nil {
		return nil, err
	}
	return &DuResult{
		Timestamp:  12345,
		TotalBytes: total,
		Paths:      entries,
	}, nil
}

func TestDu_Basic(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "test.txt", "hello world")

	result, err := runDuWithOutput(dir, Config{})
	if err != nil {
		t.Fatal(err)
	}

	if result.TotalBytes == 0 {
		t.Error("expected non-zero total bytes")
	}
}

func TestDu_SingleFile(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "hello world")

	result, err := runDuWithOutput(filePath, Config{})
	if err != nil {
		t.Fatal(err)
	}

	if result.TotalBytes == 0 {
		t.Error("expected non-zero total bytes")
	}
}

func TestDu_Directory(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	createFile(t, dir, "file1.txt", "content1")
	createFile(t, filepath.Join(dir, "subdir"), "file2.txt", "content2")

	result, err := runDuWithOutput(dir, Config{})
	if err != nil {
		t.Fatal(err)
	}

	if result.TotalBytes == 0 {
		t.Error("expected non-zero total bytes")
	}
}

func TestDu_Summarize(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "test.txt", "hello world")

	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{"-s", dir})

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	if err != nil {
		t.Fatal(err)
	}

	var result DuResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if len(result.Paths) != 1 {
		t.Errorf("expected 1 entry with -s, got %d", len(result.Paths))
	}
}

func TestDu_AllFiles(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "file1.txt", "hello")
	createFile(t, dir, "file2.txt", "world")

	result, err := runDuWithOutput(dir, Config{All: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Paths) == 0 {
		t.Error("expected entries with -a flag")
	}
}

func TestDu_MaxDepth(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	os.MkdirAll(subdir, 0755)
	createFile(t, dir, "root.txt", "root")
	createFile(t, subdir, "child.txt", "child")

	result, err := runDuWithOutput(dir, Config{All: true, MaxDepth: 1})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Paths) == 0 {
		t.Error("expected entries within max depth")
	}
}

func TestDu_NonExistent(t *testing.T) {
	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{"/nonexistent/path"})

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	if err == nil {
		var result DuResult
		if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
			t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
		}
		if len(result.Errors) == 0 {
			t.Error("expected error for non-existent path")
		}
	}
}

func TestDu_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "test.txt", "hello")

	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{dir})

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	if err != nil {
		t.Fatal(err)
	}

	var result DuResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.XMLName.Local != "du" {
		t.Errorf("expected root element 'du', got %q", result.XMLName.Local)
	}
}

func TestDu_HumanReadable(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "test.txt", "hello world")

	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{"-h", dir})

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	if err != nil {
		t.Fatal(err)
	}

	var result DuResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.TotalHuman == "" {
		t.Error("expected human readable size")
	}
}

func createFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
