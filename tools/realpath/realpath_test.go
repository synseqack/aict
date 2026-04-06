package realpath

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
)

func runRealpath(args []string) (string, error) {
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

func runRealpathWithOutput(path string) (RealpathEntry, error) {
	return resolvePath(path), nil
}

func TestRealpath_Basic(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "hello")

	result, err := runRealpathWithOutput(filePath)
	if err != nil {
		t.Fatal(err)
	}

	if result.Absolute == "" {
		t.Error("expected absolute path to be set")
	}
	if result.Exists != "true" {
		t.Errorf("expected exists=true, got %s", result.Exists)
	}
	if result.Type != "file" {
		t.Errorf("expected type 'file', got %s", result.Type)
	}
}

func TestRealpath_Directory(t *testing.T) {
	dir := t.TempDir()

	result, err := runRealpathWithOutput(dir)
	if err != nil {
		t.Fatal(err)
	}

	if result.Absolute == "" {
		t.Error("expected absolute path to be set")
	}
	if result.Type != "directory" {
		t.Errorf("expected type 'directory', got %s", result.Type)
	}
}

func TestRealpath_NonExistent(t *testing.T) {
	result, err := runRealpathWithOutput("/nonexistent/path/file.txt")
	if err != nil {
		t.Fatal(err)
	}

	if result.Exists != "false" {
		t.Errorf("expected exists=false, got %s", result.Exists)
	}
}

func TestRealpath_Symlink(t *testing.T) {
	dir := t.TempDir()
	target := createFile(t, dir, "target.txt", "target content")
	linkPath := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Skip("symlinks not supported")
	}

	result, err := runRealpathWithOutput(linkPath)
	if err != nil {
		t.Fatal(err)
	}

	if result.Type == "" {
		t.Error("expected type to be set")
	}
}

func TestRealpath_RelativePath(t *testing.T) {
	result, err := runRealpathWithOutput(".")
	if err != nil {
		t.Fatal(err)
	}

	if result.Absolute == "" {
		t.Error("expected absolute path")
	}
}

func TestRealpath_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "hello")

	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{filePath})

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	if err != nil {
		t.Fatal(err)
	}

	var result RealpathResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.XMLName.Local != "realpath" {
		t.Errorf("expected root element 'realpath', got %q", result.XMLName.Local)
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
