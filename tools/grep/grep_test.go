package grep

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

func runGrep(t *testing.T, args []string) *GrepResult {
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

	var result GrepResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}
	return &result
}

func TestGrep_Literal(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "hello world\nfoo bar\nhello again\n")

	result := runGrep(t, []string{"hello", path})
	if result.TotalMatches != 2 {
		t.Errorf("expected 2 matches, got %d", result.TotalMatches)
	}
}

func TestGrep_Regex(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "foo123\nbar456\nfoo789\n")

	result := runGrep(t, []string{"foo[0-9]+", path})
	if result.TotalMatches != 2 {
		t.Errorf("expected 2 regex matches, got %d", result.TotalMatches)
	}
}

func TestGrep_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "Hello World\nhello world\nHELLO WORLD\n")

	result := runGrep(t, []string{"-i", "hello", path})
	if result.TotalMatches != 3 {
		t.Errorf("expected 3 case-insensitive matches, got %d", result.TotalMatches)
	}
}

func TestGrep_InvertMatch(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "keep this\nremove this\nkeep too\n")

	result := runGrep(t, []string{"-v", "remove", path})
	if result.TotalMatches != 2 {
		t.Errorf("expected 2 inverted matches, got %d", result.TotalMatches)
	}
}

func TestGrep_WordMatch(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "foo\nfoobar\nfoo bar\n")

	result := runGrep(t, []string{"-w", "foo", path})
	if result.TotalMatches != 2 {
		t.Errorf("expected 2 word matches (not substring), got %d", result.TotalMatches)
	}
}

func TestGrep_LineNumbers(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "aaa\nbbb\nccc\n")

	result := runGrep(t, []string{"-n", "bbb", path})
	if result.TotalMatches != 1 {
		t.Errorf("expected 1 match with -n, got %d", result.TotalMatches)
	}
}

func TestGrep_CountOnly(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "a\na\na\nb\n")

	result := runGrep(t, []string{"-c", "a", path})
	if result.TotalMatches != 3 {
		t.Errorf("expected count=3, got %d", result.TotalMatches)
	}
}

func TestGrep_FilesWithMatches(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "match.txt", "hello\n")
	createFile(t, dir, "nomatch.txt", "world\n")

	result := runGrep(t, []string{"-r", "-l", "hello", dir})
	if result.MatchedFiles < 1 {
		t.Errorf("expected at least 1 matched file, got %d", result.MatchedFiles)
	}
	if result.TotalMatches < 1 {
		t.Errorf("expected at least 1 total match, got %d", result.TotalMatches)
	}
}

func TestGrep_Recursive(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "top.txt", "needle here\n")
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0755)
	createFile(t, sub, "deep.txt", "needle there\n")

	result := runGrep(t, []string{"-r", "needle", dir})
	if result.TotalMatches < 2 {
		t.Errorf("expected at least 2 recursive matches, got %d", result.TotalMatches)
	}
}

func TestGrep_NoMatch(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "no match here\n")

	result := runGrep(t, []string{"ZZZNOMATCH", path})
	if result.TotalMatches != 0 {
		t.Errorf("expected 0 matches, got %d", result.TotalMatches)
	}
	if len(result.Errors) > 0 {
		t.Errorf("expected no errors for no-match, got: %v", result.Errors[0].Msg)
	}
}

func TestGrep_BinarySkip(t *testing.T) {
	dir := t.TempDir()
	data := append([]byte("hello\x00"), []byte("world\n")...)
	path := filepath.Join(dir, "binary.bin")
	os.WriteFile(path, data, 0644)

	result := runGrep(t, []string{"hello", path})
	if result.TotalMatches != 0 {
		t.Errorf("expected binary file to be skipped (0 matches), got %d", result.TotalMatches)
	}
}

func TestGrep_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "hello\n")

	result := runGrep(t, []string{"hello", path})
	if result.XMLName.Local != "grep" {
		t.Errorf("expected root element 'grep', got %q", result.XMLName.Local)
	}
}
