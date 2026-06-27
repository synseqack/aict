package wc

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

func runWC(paths []string, cfg Config) (*WCResult, error) {
	result := &WCResult{}
	for _, p := range paths {
		wc, err := countFile(p, cfg)
		if err != nil {
			return nil, err
		}
		result.Files = append(result.Files, *wc)
		if wc.Errors == nil {
			result.TotalLines += wc.Lines
			result.TotalWords += wc.Words
			result.TotalBytes += wc.Bytes
		}
	}
	return result, nil
}

func TestWC_Default(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "hello world\nfoo bar baz\n")

	result, err := runWC([]string{path}, Config{Lines: true, Words: true, Bytes: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result.Files))
	}
	f := result.Files[0]
	if f.Lines != 2 {
		t.Errorf("expected 2 lines, got %d", f.Lines)
	}
	if f.Words != 5 {
		t.Errorf("expected 5 words, got %d", f.Words)
	}
	if f.Bytes != 24 {
		t.Errorf("expected 24 bytes, got %d", f.Bytes)
	}
}

func TestWC_Lines(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "a\nb\nc\n")

	result, err := runWC([]string{path}, Config{Lines: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Files[0].Lines != 3 {
		t.Errorf("expected 3 lines, got %d", result.Files[0].Lines)
	}
}

func TestWC_Words(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "one two three\n")

	result, err := runWC([]string{path}, Config{Words: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Files[0].Words != 3 {
		t.Errorf("expected 3 words, got %d", result.Files[0].Words)
	}
}

func TestWC_Bytes(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "hello\n")

	result, err := runWC([]string{path}, Config{Bytes: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Files[0].Bytes != 6 {
		t.Errorf("expected 6 bytes, got %d", result.Files[0].Bytes)
	}
}

func TestWC_MultiFile(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.txt", "one\ntwo\n")
	p2 := createFile(t, dir, "b.txt", "three\n")

	result, err := runWC([]string{p1, p2}, Config{Lines: true, Words: true, Bytes: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalLines != 3 {
		t.Errorf("expected TotalLines=3, got %d", result.TotalLines)
	}
	if result.TotalWords != 3 {
		t.Errorf("expected TotalWords=3, got %d", result.TotalWords)
	}
}

func TestWC_Empty(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "empty.txt", "")

	result, err := runWC([]string{path}, Config{Lines: true, Words: true, Bytes: true})
	if err != nil {
		t.Fatal(err)
	}
	f := result.Files[0]
	if f.Lines != 0 || f.Words != 0 || f.Bytes != 0 {
		t.Errorf("expected all zeros for empty file, got lines=%d words=%d bytes=%d", f.Lines, f.Words, f.Bytes)
	}
}

func TestWC_Directory(t *testing.T) {
	dir := t.TempDir()

	result, err := runWC([]string{dir}, Config{Lines: true, Words: true, Bytes: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files[0].Errors) == 0 {
		t.Error("expected error for directory input")
	}
}

func TestWC_Missing(t *testing.T) {
	result, err := runWC([]string{"/nonexistent/file.txt"}, Config{Lines: true, Words: true, Bytes: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files[0].Errors) == 0 {
		t.Error("expected error for non-existent file")
	}
}

func TestWC_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "hello world\n")

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

	err := Run([]string{path})
	w.Close()
	os.Stdout = oldStdout
	<-done

	if err != nil {
		t.Fatal(err)
	}

	var result WCResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}
	if result.XMLName.Local != "wc" {
		t.Errorf("expected root element 'wc', got %q", result.XMLName.Local)
	}
}
