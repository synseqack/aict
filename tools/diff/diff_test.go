package diff

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
)

func createFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func runDiff(t *testing.T, args []string) *DiffResult {
	t.Helper()
	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var outBuf bytes.Buffer
	done := make(chan struct{})
	go func() {
		outBuf.ReadFrom(r)
		close(done)
	}()

	Run(args)
	w.Close()
	os.Stdout = oldStdout
	<-done

	var result DiffResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}
	return &result
}

func TestDiff_Identical(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.txt", "hello\nworld\n")
	p2 := createFile(t, dir, "b.txt", "hello\nworld\n")

	result := runDiff(t, []string{p1, p2})
	if !result.Identical {
		t.Error("expected Identical=true for same content")
	}
	if result.AddedLines != 0 || result.RemovedLines != 0 {
		t.Errorf("expected 0 added/removed, got added=%d removed=%d", result.AddedLines, result.RemovedLines)
	}
}

func TestDiff_Added(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.txt", "hello\n")
	p2 := createFile(t, dir, "b.txt", "hello\nextra line\n")

	result := runDiff(t, []string{p1, p2})
	if result.Identical {
		t.Error("expected files to differ")
	}
	if result.AddedLines == 0 {
		t.Error("expected AddedLines > 0")
	}
}

func TestDiff_Removed(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.txt", "hello\nworld\n")
	p2 := createFile(t, dir, "b.txt", "hello\n")

	result := runDiff(t, []string{p1, p2})
	if result.Identical {
		t.Error("expected files to differ")
	}
	if result.RemovedLines == 0 {
		t.Error("expected RemovedLines > 0")
	}
}

func TestDiff_Changed(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.txt", "hello\nworld\n")
	p2 := createFile(t, dir, "b.txt", "hello\nearth\n")

	result := runDiff(t, []string{p1, p2})
	if result.Identical {
		t.Error("expected files to differ")
	}
	if result.AddedLines == 0 || result.RemovedLines == 0 {
		t.Errorf("expected both added and removed lines, got added=%d removed=%d", result.AddedLines, result.RemovedLines)
	}
}

func TestDiff_IgnoreSpace(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.txt", "hello world\n")
	p2 := createFile(t, dir, "b.txt", "hello   world\n")

	result := runDiff(t, []string{"-w", p1, p2})
	if !result.Identical {
		t.Error("expected Identical=true with -w (ignore whitespace)")
	}
}

func TestDiff_Quiet(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.txt", "aaa\n")
	p2 := createFile(t, dir, "b.txt", "bbb\n")

	result := runDiff(t, []string{"-q", p1, p2})
	if result.Identical {
		t.Error("expected files to differ with -q flag")
	}
}

func TestDiff_MissingFile(t *testing.T) {
	dir := t.TempDir()
	existing := createFile(t, dir, "a.txt", "hello\n")

	result := runDiff(t, []string{existing, "/nonexistent/missing.txt"})
	if len(result.Errors) == 0 {
		t.Error("expected error for missing file")
	}
}

func TestDiff_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.txt", "hello\n")
	p2 := createFile(t, dir, "b.txt", "world\n")

	result := runDiff(t, []string{p1, p2})
	if result.XMLName.Local != "diff" {
		t.Errorf("expected root element 'diff', got %q", result.XMLName.Local)
	}
}
