package ls

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func runLS(args []string) (string, error) {
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

func runLSWithOutput(args []string, cfg Config) (*LSResult, error) {
	if len(args) == 0 {
		args = []string{"."}
	}
	return listDir(args[0], cfg)
}

func TestLS_Basic(t *testing.T) {
	dir := t.TempDir()

	createFile(t, dir, "hello.txt", "hello world")
	createFile(t, dir, "main.go", "package main")

	result, err := runLSWithOutput([]string{dir}, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.Absolute != dir {
		t.Errorf("expected absolute path %q, got %q", dir, result.Absolute)
	}
	if result.TotalEntries != 2 {
		t.Errorf("expected 2 entries, got %d", result.TotalEntries)
	}
	if result.Timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}

	var foundTxt, foundGo bool
	for _, e := range result.Entries {
		switch entry := e.(type) {
		case FileEntry:
			if entry.Name == "hello.txt" {
				foundTxt = true
				if entry.Language != "text" {
					t.Errorf("hello.txt language: expected 'text', got %q", entry.Language)
				}
				if entry.Binary != "false" {
					t.Errorf("hello.txt should not be binary")
				}
			}
			if entry.Name == "main.go" {
				foundGo = true
				if entry.Language != "go" {
					t.Errorf("main.go language: expected 'go', got %q", entry.Language)
				}
			}
		}
	}

	if !foundTxt {
		t.Error("expected hello.txt in output")
	}
	if !foundGo {
		t.Error("expected main.go in output")
	}
}

func TestLS_Directories(t *testing.T) {
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	createFile(t, dir, "file.txt", "content")

	result, err := runLSWithOutput([]string{dir}, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	var foundDir, foundFile bool
	for _, e := range result.Entries {
		switch e.(type) {
		case DirEntry:
			foundDir = true
		case FileEntry:
			foundFile = true
		}
	}

	if !foundDir {
		t.Error("expected directory entry")
	}
	if !foundFile {
		t.Error("expected file entry")
	}
}

func TestLS_HiddenFiles(t *testing.T) {
	dir := t.TempDir()

	createFile(t, dir, "visible.txt", "visible")
	createFile(t, dir, ".hidden", "hidden")

	result, err := runLSWithOutput([]string{dir}, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.TotalEntries != 1 {
		t.Errorf("expected 1 entry (no hidden), got %d", result.TotalEntries)
	}

	resultA, err := runLSWithOutput([]string{dir}, Config{XML: true, All: true})
	if err != nil {
		t.Fatal(err)
	}

	if resultA.TotalEntries < 2 {
		t.Errorf("expected >=2 entries with -a, got %d", resultA.TotalEntries)
	}
}

func TestLS_Symlinks(t *testing.T) {
	dir := t.TempDir()

	target := createFile(t, dir, "target.txt", "target content")
	linkPath := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Skip("symlinks not supported")
	}

	result, err := runLSWithOutput([]string{dir}, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	var foundSymlink bool
	for _, e := range result.Entries {
		if entry, ok := e.(SymlinkEntry); ok {
			foundSymlink = true
			if entry.Name != "link.txt" {
				t.Errorf("expected symlink name 'link.txt', got %q", entry.Name)
			}
			if entry.TargetExists != "true" {
				t.Error("expected symlink target to exist")
			}
		}
	}

	if !foundSymlink {
		t.Error("expected symlink entry")
	}
}

func TestLS_Error_NonExistent(t *testing.T) {
	result, err := runLSWithOutput([]string{"/nonexistent/path/that/does/not/exist"}, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Errors) == 0 {
		t.Fatal("expected error in result")
	}

	if result.Errors[0].Code == 0 {
		t.Error("expected non-zero error code")
	}
	if result.Errors[0].Msg == "" {
		t.Error("expected error message")
	}
}

func TestLS_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	result, err := runLSWithOutput([]string{dir}, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.TotalEntries != 0 {
		t.Errorf("expected 0 entries in empty dir, got %d", result.TotalEntries)
	}
	if len(result.Entries) != 0 {
		t.Errorf("expected empty entries slice")
	}
}

func TestLS_SingleFile(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "single.go", "package main")

	result, err := runLSWithOutput([]string{filePath}, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.TotalEntries != 1 {
		t.Errorf("expected 1 entry, got %d", result.TotalEntries)
	}

	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}

	entry, ok := result.Entries[0].(FileEntry)
	if !ok {
		t.Fatal("expected FileEntry")
	}
	if entry.Name != "single.go" {
		t.Errorf("expected name 'single.go', got %q", entry.Name)
	}
	if entry.Language != "go" {
		t.Errorf("expected language 'go', got %q", entry.Language)
	}
}

func TestLS_SortByTime(t *testing.T) {
	dir := t.TempDir()

	createFile(t, dir, "old.txt", "old")
	createFile(t, dir, "new.txt", "new")

	oldPath := filepath.Join(dir, "old.txt")
	os.Chtimes(oldPath, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour))

	result, err := runLSWithOutput([]string{dir}, Config{XML: true, SortTime: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Entries) < 2 {
		t.Fatal("expected at least 2 entries")
	}

	first := result.Entries[0].(FileEntry)
	second := result.Entries[1].(FileEntry)

	if first.Modified < second.Modified {
		t.Error("expected newest file first with -t flag")
	}
}

func TestLS_Recursive(t *testing.T) {
	dir := t.TempDir()

	subdir := filepath.Join(dir, "sub")
	os.MkdirAll(subdir, 0755)
	createFile(t, dir, "root.txt", "root")
	createFile(t, subdir, "child.txt", "child")

	result, err := runLSWithOutput([]string{dir}, Config{XML: true, Recursive: true})
	if err != nil {
		t.Fatal(err)
	}

	var foundNested bool
	for _, e := range result.Entries {
		if nested, ok := e.(*LSResult); ok {
			if nested.TotalEntries > 0 {
				foundNested = true
			}
		}
	}

	if !foundNested {
		t.Error("expected nested LSResult in recursive mode")
	}
}

func TestLS_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "test.xml", "<root/>")

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

	var result LSResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.XMLName.Local != "ls" {
		t.Errorf("expected root element 'ls', got %q", result.XMLName.Local)
	}
}

func TestLS_Permissions(t *testing.T) {
	dir := t.TempDir()

	execPath := createFile(t, dir, "script.sh", "#!/bin/bash")
	os.Chmod(execPath, 0755)

	result, err := runLSWithOutput([]string{dir}, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range result.Entries {
		if entry, ok := e.(FileEntry); ok {
			if entry.Name == "script.sh" {
				if entry.Executable != "true" {
					t.Error("expected script.sh to be executable")
				}
				if !strings.HasPrefix(entry.Permissions, "-rwx") {
					t.Errorf("expected -rwx permissions, got %q", entry.Permissions)
				}
			}
		}
	}
}

func TestLS_BinaryFile(t *testing.T) {
	dir := t.TempDir()

	binPath := filepath.Join(dir, "binary.bin")
	data := make([]byte, 512)
	for i := range data {
		data[i] = byte(i % 256)
	}
	data[0] = 0x00
	data[1] = 0x01
	data[2] = 0x02
	os.WriteFile(binPath, data, 0644)

	result, err := runLSWithOutput([]string{dir}, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range result.Entries {
		if entry, ok := e.(FileEntry); ok {
			if entry.Name == "binary.bin" {
				if entry.Binary != "true" {
					t.Error("expected binary.bin to be marked as binary")
				}
			}
		}
	}
}

func TestLS_MultiplePaths(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	createFile(t, dir1, "a.txt", "a")
	createFile(t, dir2, "b.txt", "b")

	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{dir1, dir2})

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	if err != nil {
		t.Fatal(err)
	}

	output := outBuf.String()
	if !strings.Contains(output, dir1) {
		t.Error("expected output to contain first path")
	}
	if !strings.Contains(output, dir2) {
		t.Error("expected output to contain second path")
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
