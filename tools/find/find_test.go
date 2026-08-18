package find

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

func runFind(t *testing.T, args []string) *FindResult {
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

	var result FindResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}
	return &result
}

func TestFind_All(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "a.txt", "aaa")
	createFile(t, dir, "b.go", "bbb")

	result := runFind(t, []string{dir})
	if result.TotalMatches < 2 {
		t.Errorf("expected at least 2 matches, got %d", result.TotalMatches)
	}
}

func TestFind_ByName(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "hello.txt", "")
	createFile(t, dir, "hello.go", "")

	result := runFind(t, []string{dir, "-name", "*.txt"})
	for _, m := range result.Matches {
		ext := filepath.Ext(m.Path)
		if ext != ".txt" {
			t.Errorf("expected only .txt files, got %q", m.Path)
		}
	}
	if result.TotalMatches == 0 {
		t.Error("expected at least one .txt match")
	}
}

func TestFind_TypeFile(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "test.txt", "data")
	subdir := filepath.Join(dir, "subdir")
	os.Mkdir(subdir, 0755)

	result := runFind(t, []string{dir, "-type", "f"})
	for _, m := range result.Matches {
		if m.Type != "file" {
			t.Errorf("expected type=file, got %q for %q", m.Type, m.Path)
		}
	}
}

func TestFind_TypeDir(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "test.txt", "data")
	subdir := filepath.Join(dir, "subdir")
	os.Mkdir(subdir, 0755)

	result := runFind(t, []string{dir, "-type", "d"})
	for _, m := range result.Matches {
		if m.Type != "directory" {
			t.Errorf("expected type=directory, got %q for %q", m.Type, m.Path)
		}
	}
	if result.TotalMatches == 0 {
		t.Error("expected at least one directory match (root dir)")
	}
}

func TestFind_MaxDepth(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "top.txt", "")
	subdir := filepath.Join(dir, "sub")
	os.Mkdir(subdir, 0755)
	createFile(t, subdir, "deep.txt", "")

	result := runFind(t, []string{dir, "-maxdepth", "1"})
	for _, m := range result.Matches {
		if m.Depth > 1 {
			t.Errorf("expected depth <= 1, got depth=%d for %q", m.Depth, m.Path)
		}
	}
}

func TestFind_Size(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "big.txt", "hello world\n")
	createFile(t, dir, "empty.txt", "")

	result := runFind(t, []string{dir, "-size", "5"})
	for _, m := range result.Matches {
		if m.SizeBytes < 5 {
			t.Errorf("expected SizeBytes >= 5, got %d for %q", m.SizeBytes, m.Path)
		}
	}
}

func TestFind_Invert(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "test.txt", "")
	createFile(t, dir, "other.go", "")

	result := runFind(t, []string{dir, "-not", "-name", "*.txt", "-type", "f"})
	for _, m := range result.Matches {
		ext := filepath.Ext(m.Path)
		if ext == ".txt" {
			t.Errorf("expected .txt files to be excluded, but got %q", m.Path)
		}
	}
}

func TestFind_Missing(t *testing.T) {
	result := runFind(t, []string{testutil.MissingPath(t, "directory")})
	if len(result.Errors) == 0 {
		t.Error("expected error for non-existent root")
	}
}

func TestFind_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "test.txt", "hello")

	result := runFind(t, []string{dir})
	if result.XMLName.Local != "find" {
		t.Errorf("expected root element 'find', got %q", result.XMLName.Local)
	}
}

func TestFind_NotBindsToNextPredicateOnly(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "a.go", "")
	createFile(t, dir, "b.txt", "")
	createFile(t, dir, "c.go", "")

	// -not negates only -name "a.go"; -name "*.go" must still apply (GNU
	// semantics). Expect exactly c.go — issue #30's repro returned the
	// directory and b.txt instead.
	result := runFind(t, []string{dir, "-not", "-name", "a.go", "-name", "*.go", "-type", "f"})

	var names []string
	for _, m := range result.Matches {
		names = append(names, filepath.Base(m.Path))
	}
	if len(names) != 1 || names[0] != "c.go" {
		t.Errorf("expected exactly [c.go], got %v", names)
	}

	// The negation must be visible in the echoed conditions.
	foundNegated := false
	for _, c := range result.Conditions {
		if c.Type == "name" && c.Value == "a.go" && c.Negated == "true" {
			foundNegated = true
		}
		if c.Type == "name" && c.Value == "*.go" && c.Negated == "true" {
			t.Errorf("-not leaked onto the second -name predicate")
		}
	}
	if !foundNegated {
		t.Error("expected negated=\"true\" on the -not'd condition")
	}
}

func TestFind_NotEachPredicate(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "a.go", "")
	createFile(t, dir, "b.txt", "")

	// Two independent negations: NOT dir AND NOT *.txt → only a.go.
	result := runFind(t, []string{dir, "-not", "-type", "d", "-not", "-name", "*.txt"})

	var names []string
	for _, m := range result.Matches {
		names = append(names, filepath.Base(m.Path))
	}
	if len(names) != 1 || names[0] != "a.go" {
		t.Errorf("expected exactly [a.go], got %v", names)
	}
}
