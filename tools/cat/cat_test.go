package cat

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/synseqack/aict/internal/testutil"
)

func createFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCat_Text(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "hello.txt", "line one\nline two\n")

	result, err := catFile(path, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected error: %v", result.Errors[0].Msg)
	}
	if result.Lines != 2 {
		t.Errorf("expected 2 lines, got %d", result.Lines)
	}
	if result.Binary != "false" {
		t.Errorf("expected Binary=false, got %q", result.Binary)
	}
	if result.Content != "line one\nline two\n" {
		t.Errorf("unexpected content: %q", result.Content)
	}
}

func TestCat_Binary(t *testing.T) {
	dir := t.TempDir()
	data := []byte{0x00, 0x01, 0x02, 0x03}
	path := filepath.Join(dir, "binary.bin")
	os.WriteFile(path, data, 0644)

	result, err := catFile(path, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Binary != "true" {
		t.Errorf("expected Binary=true for null-byte file, got %q", result.Binary)
	}
	if result.Content != "" {
		t.Errorf("expected empty Content for binary file, got %q", result.Content)
	}
}

func TestCat_UTFBOM(t *testing.T) {
	dir := t.TempDir()
	bom := []byte{0xEF, 0xBB, 0xBF}
	data := append(bom, []byte("hello\n")...)
	path := filepath.Join(dir, "bom.txt")
	os.WriteFile(path, data, 0644)

	result, err := catFile(path, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Encoding != "utf-8-bom" {
		t.Errorf("expected Encoding=utf-8-bom, got %q", result.Encoding)
	}
}

func TestCat_MultiFile(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.txt", "aaa\n")
	p2 := createFile(t, dir, "b.txt", "bbb\n")

	mainResult := &CatResult{}
	for _, p := range []string{p1, p2} {
		cr, err := catFile(p, Config{})
		if err != nil {
			t.Fatal(err)
		}
		mainResult.Files = append(mainResult.Files, *cr)
		if cr.Errors == nil {
			mainResult.Lines += cr.Lines
			mainResult.SizeBytes += cr.SizeBytes
		}
	}

	if len(mainResult.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(mainResult.Files))
	}
}

func TestCat_Directory(t *testing.T) {
	dir := t.TempDir()

	result, err := catFile(dir, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) == 0 {
		t.Error("expected error for directory input")
	}
}

func TestCat_Missing(t *testing.T) {
	result, err := catFile(testutil.MissingPath(t, "missing.txt"), Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) == 0 {
		t.Error("expected error for non-existent file")
	}
}

func TestCat_XMLSpecialContent(t *testing.T) {
	dir := t.TempDir()
	content := "cdata terminator ]]> here\nmarkup <tag attr=\"v\"> & entities &amp;\n"
	path := createFile(t, dir, "special.txt", content)

	os.Setenv("AICT_XML", "1")
	os.Setenv("AICT_NOCOMPACT", "1")
	defer os.Unsetenv("AICT_XML")
	defer os.Unsetenv("AICT_NOCOMPACT")

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

	var result CatResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("XML with ]]> content must stay well-formed: %v\n%s", err, outBuf.String())
	}
	if result.Content != content {
		t.Errorf("content did not round-trip: got %q, want %q", result.Content, content)
	}
}

func TestCat_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "hello\n")

	os.Setenv("AICT_XML", "1")
	os.Setenv("AICT_NOCOMPACT", "1")
	defer os.Unsetenv("AICT_XML")
	defer os.Unsetenv("AICT_NOCOMPACT")

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

	var result CatResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}
	if result.XMLName.Local != "cat" {
		t.Errorf("expected root element 'cat', got %q", result.XMLName.Local)
	}
}
