package sed

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

func runSed(t *testing.T, args []string) *SedResult {
	t.Helper()
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

	Run(args)
	w.Close()
	os.Stdout = oldStdout
	<-done

	var result SedResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}
	return &result
}

func lineContents(sf SedFile) []string {
	var out []string
	for _, l := range sf.Lines {
		out = append(out, l.Content)
	}
	return out
}

func TestSed_Substitution(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "hello world\n")

	result := runSed(t, []string{"-e", "s/hello/goodbye/", path})
	if len(result.Files) == 0 {
		t.Fatal("expected file result")
	}
	lines := lineContents(result.Files[0])
	if len(lines) == 0 {
		t.Fatal("expected output lines")
	}
	found := false
	for _, l := range lines {
		if l == "goodbye world" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'goodbye world' in output, got: %v", lines)
	}
}

func TestSed_GlobalFlag(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "a a a\n")

	result := runSed(t, []string{"-e", "s/a/b/g", path})
	if len(result.Files) == 0 {
		t.Fatal("expected file result")
	}
	lines := lineContents(result.Files[0])
	found := false
	for _, l := range lines {
		if l == "b b b" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'b b b' in output, got: %v", lines)
	}
}

func TestSed_Delete(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "keep\ndelete me\nkeep too\n")

	result := runSed(t, []string{"-e", "/delete/d", path})
	if len(result.Files) == 0 {
		t.Fatal("expected file result")
	}
	lines := lineContents(result.Files[0])
	for _, l := range lines {
		if l == "delete me" {
			t.Error("deleted line should not appear in output")
		}
	}
}

func TestSed_PrintWithSuppress(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "aaa\nbbb\nccc\n")

	result := runSed(t, []string{"-n", "-e", "/bbb/p", path})
	if len(result.Files) == 0 {
		t.Fatal("expected file result")
	}
	lines := lineContents(result.Files[0])
	if len(lines) == 0 {
		t.Error("expected 'bbb' to be printed with -n -e /bbb/p")
	}
}

func TestSed_LineAddress(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "first\nsecond\nthird\n")

	result := runSed(t, []string{"-e", "2s/second/2nd/", path})
	if len(result.Files) == 0 {
		t.Fatal("expected file result")
	}
	lines := lineContents(result.Files[0])
	found := false
	for _, l := range lines {
		if l == "2nd" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected '2nd' in output, got: %v", lines)
	}
}

func TestSed_RegexAddress(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "foo bar\nbaz qux\nfoo baz\n")

	result := runSed(t, []string{"-e", "/foo/s/foo/FOO/", path})
	if len(result.Files) == 0 {
		t.Fatal("expected file result")
	}
	lines := lineContents(result.Files[0])
	fooCount := 0
	for _, l := range lines {
		if len(l) > 0 && l[:3] == "FOO" {
			fooCount++
		}
	}
	if fooCount == 0 {
		t.Errorf("expected FOO-prefixed lines, got: %v", lines)
	}
}

func TestSed_MissingFile(t *testing.T) {
	result := runSed(t, []string{"-e", "s/a/b/", testutil.MissingPath(t, "file.txt")})
	if len(result.Errors) == 0 {
		t.Error("expected error for non-existent file")
	}
}

func TestSed_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "hello\n")

	result := runSed(t, []string{"-e", "s/hello/world/", path})
	if result.XMLName.Local != "sed" {
		t.Errorf("expected root element 'sed', got %q", result.XMLName.Local)
	}
}

func TestSed_Empty(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "empty.txt", "")

	os.Setenv("AICT_XML", "1")
	os.Setenv("AICT_NOCOMPACT", "1")
	defer os.Unsetenv("AICT_XML")
	defer os.Unsetenv("AICT_NOCOMPACT")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{"-e", "s/a/b/g", filePath})

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	if err != nil {
		t.Fatal(err)
	}

	var result SedResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if len(result.Files) > 0 && result.Files[0].LinesRead != 0 {
		t.Errorf("expected LinesRead=0, got %d", result.Files[0].LinesRead)
	}
}

func TestSed_NoScript(t *testing.T) {
	os.Setenv("AICT_XML", "1")
	os.Setenv("AICT_NOCOMPACT", "1")
	defer os.Unsetenv("AICT_XML")
	defer os.Unsetenv("AICT_NOCOMPACT")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{})

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	if err != nil {
		t.Fatal(err)
	}

	var result SedResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if len(result.Errors) == 0 {
		t.Error("expected error for missing script")
	}
}
