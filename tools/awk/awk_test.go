package awk

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

func runAwk(t *testing.T, args []string) *AwkResult {
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

	var result AwkResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}
	return &result
}

func TestAwk_FieldExtract(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "alice 30\nbob 25\n")

	result := runAwk(t, []string{"{print $1}", path})
	if len(result.Files) == 0 {
		t.Fatal("expected file result")
	}
	lines := result.Files[0].Lines
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d", len(lines))
	}
	if lines[0].Output != "alice" {
		t.Errorf("expected 'alice', got %q", lines[0].Output)
	}
	if lines[1].Output != "bob" {
		t.Errorf("expected 'bob', got %q", lines[1].Output)
	}
}

func TestAwk_MultiField(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "a b c\nd e f\n")

	result := runAwk(t, []string{"{print $1, $3}", path})
	if len(result.Files) == 0 {
		t.Fatal("expected file result")
	}
	lines := result.Files[0].Lines
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d", len(lines))
	}
	if lines[0].Output != "a c" {
		t.Errorf("expected 'a c', got %q", lines[0].Output)
	}
}

func TestAwk_CustomFS(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.csv", "alice,30,engineer\nbob,25,manager\n")

	result := runAwk(t, []string{"-F", ",", "{print $2}", path})
	if len(result.Files) == 0 {
		t.Fatal("expected file result")
	}
	lines := result.Files[0].Lines
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d", len(lines))
	}
	if lines[0].Output != "30" {
		t.Errorf("expected '30', got %q", lines[0].Output)
	}
}

func TestAwk_PatternFilter(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "apple\nbanana\napricot\n")

	result := runAwk(t, []string{"/^a/ {print $0}", path})
	if len(result.Files) == 0 {
		t.Fatal("expected file result")
	}
	lines := result.Files[0].Lines
	if len(lines) != 2 {
		t.Errorf("expected 2 lines matching /^a/, got %d", len(lines))
	}
}

func TestAwk_NR(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "one\ntwo\nthree\n")

	result := runAwk(t, []string{"{print NR}", path})
	if len(result.Files) == 0 {
		t.Fatal("expected file result")
	}
	lines := result.Files[0].Lines
	if len(lines) != 3 {
		t.Fatalf("expected 3 output lines, got %d", len(lines))
	}
	if lines[0].Output != "1" {
		t.Errorf("expected NR=1, got %q", lines[0].Output)
	}
}

func TestAwk_MissingFile(t *testing.T) {
	result := runAwk(t, []string{"{print $1}", "/nonexistent/file.txt"})
	if len(result.Errors) == 0 {
		t.Error("expected error for non-existent file")
	}
}

func TestAwk_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "hello world\n")

	result := runAwk(t, []string{"{print $1}", path})
	if result.XMLName.Local != "awk" {
		t.Errorf("expected root element 'awk', got %q", result.XMLName.Local)
	}
}
