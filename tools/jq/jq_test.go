package jq

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
)

func writeJSON(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func runJQ(t *testing.T, args []string) *JQResult {
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

	var result JQResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}
	return &result
}

func TestJQ_FieldAccess(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "data.json", `{"name":"alice","age":30}`)

	result := runJQ(t, []string{".name", path})
	if result.Count != 1 {
		t.Fatalf("expected 1 value, got %d", result.Count)
	}
	if result.Values[0].Type != "string" {
		t.Errorf("expected type string, got %q", result.Values[0].Type)
	}
	if result.Values[0].Raw != `"alice"` {
		t.Errorf("expected raw=\"alice\", got %q", result.Values[0].Raw)
	}
}

func TestJQ_Nested(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "data.json", `{"user":{"name":"bob"}}`)

	result := runJQ(t, []string{".user.name", path})
	if result.Count != 1 {
		t.Fatalf("expected 1 value, got %d", result.Count)
	}
	if result.Values[0].Raw != `"bob"` {
		t.Errorf("expected raw=\"bob\", got %q", result.Values[0].Raw)
	}
}

func TestJQ_ArrayIndex(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "data.json", `["a","b","c"]`)

	result := runJQ(t, []string{".[1]", path})
	if result.Count != 1 {
		t.Fatalf("expected 1 value, got %d", result.Count)
	}
	if result.Values[0].Raw != `"b"` {
		t.Errorf("expected raw=\"b\", got %q", result.Values[0].Raw)
	}
}

func TestJQ_ArrayIterate(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "data.json", `["x","y","z"]`)

	result := runJQ(t, []string{".[]", path})
	if result.Count != 3 {
		t.Errorf("expected 3 values, got %d", result.Count)
	}
}

func TestJQ_FieldFromArray(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "data.json", `[{"id":1},{"id":2}]`)

	result := runJQ(t, []string{".[].id", path})
	if result.Count != 2 {
		t.Errorf("expected 2 values, got %d", result.Count)
	}
}

func TestJQ_MissingField(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "data.json", `{"name":"alice"}`)

	result := runJQ(t, []string{".nonexistent", path})
	if result.Count != 1 {
		t.Fatalf("expected 1 value (null), got %d", result.Count)
	}
	if result.Values[0].Type != "null" {
		t.Errorf("expected type=null for missing field, got %q", result.Values[0].Type)
	}
}

func TestJQ_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "bad.json", `{not valid json}`)

	result := runJQ(t, []string{".key", path})
	if len(result.Errors) == 0 {
		t.Error("expected error for invalid JSON")
	}
}

func TestJQ_Identity(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "data.json", `{"x":1}`)

	result := runJQ(t, []string{".", path})
	if result.Count != 1 {
		t.Fatalf("expected 1 value for identity, got %d", result.Count)
	}
	if result.Values[0].Type != "object" {
		t.Errorf("expected type=object, got %q", result.Values[0].Type)
	}
}

func TestJQ_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "data.json", `{"k":"v"}`)

	result := runJQ(t, []string{".k", path})
	if result.XMLName.Local != "jq" {
		t.Errorf("expected root element 'jq', got %q", result.XMLName.Local)
	}
}
