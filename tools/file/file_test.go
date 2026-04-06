package file

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
)

func runFile(args []string) (string, error) {
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

func runFileWithOutput(path string, cfg Config) (*FileResult, error) {
	return identifyFile(path, cfg)
}

func TestFile_Basic(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "hello.txt", "hello world")

	result, err := runFileWithOutput(filepath.Join(dir, "hello.txt"), Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.Path == "" {
		t.Error("expected path to be set")
	}
	if result.MIME == "" {
		t.Error("expected MIME to be set")
	}
	if result.Category == "" {
		t.Error("expected category to be set")
	}
}

func TestFile_TextFile(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "hello world")

	result, err := runFileWithOutput(filePath, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.Category != "text" {
		t.Errorf("expected category 'text', got %q", result.Category)
	}
	if result.Language != "text" {
		t.Errorf("expected language 'text', got %q", result.Language)
	}
}

func TestFile_GoSource(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "main.go", "package main\nfunc main() {}")

	result, err := runFileWithOutput(filePath, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.Language != "go" {
		t.Errorf("expected language 'go', got %q", result.Language)
	}
}

func TestFile_Directory(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	os.MkdirAll(subdir, 0755)

	result, err := runFileWithOutput(subdir, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.Type != "directory" {
		t.Errorf("expected type 'directory', got %q", result.Type)
	}
	if result.Category != "directory" {
		t.Errorf("expected category 'directory', got %q", result.Category)
	}
}

func TestFile_Symlink(t *testing.T) {
	dir := t.TempDir()
	target := createFile(t, dir, "target.txt", "target content")
	linkPath := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Skip("symlinks not supported")
	}

	result, err := runFileWithOutput(linkPath, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.Type != "symlink" {
		t.Errorf("expected type 'symlink', got %q", result.Type)
	}
}

func TestFile_Binary(t *testing.T) {
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

	result, err := runFileWithOutput(binPath, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.Category != "binary" && result.Category != "application" {
		t.Errorf("expected binary category, got %q", result.Category)
	}
}

func TestFile_Executable(t *testing.T) {
	dir := t.TempDir()
	execPath := createFile(t, dir, "script.sh", "#!/bin/bash")
	os.Chmod(execPath, 0755)

	result, err := runFileWithOutput(execPath, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.Executable != "true" {
		t.Error("expected script.sh to be executable")
	}
}

func TestFile_NonExistent(t *testing.T) {
	result, err := runFileWithOutput("/nonexistent/path/file.txt", Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Errors) == 0 {
		t.Fatal("expected error for non-existent file")
	}

	if result.Errors[0].Code == 0 {
		t.Error("expected non-zero error code")
	}
}

func TestFile_BriefFlag(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "test.txt", "hello")

	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	output, err := runFile([]string{"-b", filepath.Join(dir, "test.txt")})
	if err != nil {
		t.Fatal(err)
	}

	if output == "" {
		t.Error("expected brief output")
	}
}

func TestFile_MIMEFlag(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "test.txt", "hello")

	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	output, err := runFile([]string{"-i", filepath.Join(dir, "test.txt")})
	if err != nil {
		t.Fatal(err)
	}

	if output == "" {
		t.Error("expected MIME output")
	}
}

func TestFile_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "test.txt", "hello")

	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{filepath.Join(dir, "test.txt")})

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	if err != nil {
		t.Fatal(err)
	}

	var result FileResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.XMLName.Local != "file" {
		t.Errorf("expected root element 'file', got %q", result.XMLName.Local)
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
