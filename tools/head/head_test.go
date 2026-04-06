package head

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
)

func runHead(args []string) (string, error) {
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

func runHeadWithOutput(path string, cfg Config) (*HeadResult, error) {
	return headFile(path, cfg)
}

func TestHead_Basic(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "line1\nline2\nline3\nline4\nline5")

	result, err := runHeadWithOutput(filePath, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.Path == "" {
		t.Error("expected path to be set")
	}
	if result.LinesRequested == 0 {
		t.Error("expected lines_requested to be set")
	}
}

func TestHead_DefaultLines(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12")

	result, err := runHeadWithOutput(filePath, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.LinesReturned != 10 {
		t.Errorf("expected 10 lines returned, got %d", result.LinesReturned)
	}
	if result.Truncated != "true" {
		t.Errorf("expected truncated=true, got %s", result.Truncated)
	}
}

func TestHead_NLines(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "line1\nline2\nline3\nline4\nline5")

	result, err := runHeadWithOutput(filePath, Config{LinesFlag: true, Lines: 3})
	if err != nil {
		t.Fatal(err)
	}

	if result.LinesReturned != 3 {
		t.Errorf("expected 3 lines returned, got %d", result.LinesReturned)
	}
	if result.Truncated != "true" {
		t.Errorf("expected truncated=true, got %s", result.Truncated)
	}
}

func TestHead_MoreLinesThanExist(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "line1\nline2\nline3")

	result, err := runHeadWithOutput(filePath, Config{LinesFlag: true, Lines: 10})
	if err != nil {
		t.Fatal(err)
	}

	if result.LinesReturned != 3 {
		t.Errorf("expected 3 lines returned, got %d", result.LinesReturned)
	}
	if result.Truncated != "false" {
		t.Errorf("expected truncated=false, got %s", result.Truncated)
	}
}

func TestHead_Bytes(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "abcdefghij")

	result, err := runHeadWithOutput(filePath, Config{BytesFlag: true, Bytes: 5})
	if err != nil {
		t.Fatal(err)
	}

	if result.BytesReturned != 5 {
		t.Errorf("expected 5 bytes returned, got %d", result.BytesReturned)
	}
}

func TestHead_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "empty.txt", "")

	result, err := runHeadWithOutput(filePath, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.LinesReturned != 0 {
		t.Errorf("expected 0 lines returned, got %d", result.LinesReturned)
	}
}

func TestHead_NonExistent(t *testing.T) {
	result, err := runHeadWithOutput("/nonexistent/file.txt", Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Errors) == 0 {
		t.Fatal("expected error for non-existent file")
	}
}

func TestHead_Directory(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)

	result, err := runHeadWithOutput(dir, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Errors) == 0 {
		t.Fatal("expected error for directory")
	}
}

func TestHead_Binary(t *testing.T) {
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

	result, err := runHeadWithOutput(binPath, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Errors) == 0 {
		t.Fatal("expected error for binary file")
	}
}

func TestHead_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	file1 := createFile(t, dir, "a.txt", "aaa\nbbb")
	file2 := createFile(t, dir, "b.txt", "111\n222")

	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	_, err := runHead([]string{file1, file2})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHead_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "line1\nline2\nline3")

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

	var result HeadResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.XMLName.Local != "head" {
		t.Errorf("expected root element 'head', got %q", result.XMLName.Local)
	}
}

func TestHead_LanguageEnrichment(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "main.go", "package main\nfunc main() {}")

	result, err := runHeadWithOutput(filePath, Config{XML: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.Language != "go" {
		t.Errorf("expected language 'go', got %q", result.Language)
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
