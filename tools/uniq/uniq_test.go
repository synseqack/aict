package uniq

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
)

func runUniq(args []string) (string, error) {
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

func TestUniq_Basic(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "a\nb\na\nc\nb\n")

	result, err := runUniqWithOutput(filePath, Config{})
	if err != nil {
		t.Fatal(err)
	}

	if result.LinesIn != 5 {
		t.Errorf("expected 5 lines in, got %d", result.LinesIn)
	}
}

func TestUniq_Deduplication(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "a\na\nb\nb\nc")

	result, err := runUniqWithOutput(filePath, Config{})
	if err != nil {
		t.Fatal(err)
	}

	if result.DuplicatesRemoved == 0 {
		t.Error("expected duplicates removed")
	}
}

func TestUniq_Count(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "a\na\nb")

	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{"-c", filePath})

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	if err != nil {
		t.Fatal(err)
	}

	var result UniqResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if len(result.Duplicates) == 0 {
		t.Error("expected duplicate counts")
	}
}

func TestUniq_Unique(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "a\na\nb\nb\nc")

	result, err := runUniqWithOutput(filePath, Config{Unique: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.LinesOut != 1 {
		t.Errorf("expected 1 unique line, got %d", result.LinesOut)
	}
}

func TestUniq_Duplicates(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "a\na\nb\nb\nc")

	result, err := runUniqWithOutput(filePath, Config{Duplicates: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.LinesOut != 2 {
		t.Errorf("expected 2 duplicate lines, got %d", result.LinesOut)
	}
}

func TestUniq_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "a\nb\na\n")

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

	var result UniqResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.XMLName.Local != "uniq" {
		t.Errorf("expected root element 'uniq', got %q", result.XMLName.Local)
	}
}

func runUniqWithOutput(path string, cfg Config) (*UniqResult, error) {
	lines, err := readLines(path)
	if err != nil {
		return nil, err
	}

	result := &UniqResult{Timestamp: 12345}
	result.LinesIn = len(lines)

	outputLines, dups := processUniq(lines, cfg)
	result.LinesOut = len(outputLines)
	result.DuplicatesRemoved = result.LinesIn - result.LinesOut

	if cfg.Count {
		result.Duplicates = dups
	}

	return result, nil
}

func createFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
