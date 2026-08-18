package stat

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

func TestStat_File(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "hello world\n")

	result, err := statPath(path, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected error: %v", result.Errors[0].Msg)
	}
	if result.Type != "file" {
		t.Errorf("expected Type=file, got %q", result.Type)
	}
	if result.Permissions == "" {
		t.Error("expected Permissions to be set")
	}
	if result.ModeOctal == "" {
		t.Error("expected ModeOctal to be set")
	}
	if result.Mtime == 0 {
		t.Error("expected non-zero Mtime")
	}
}

func TestStat_Directory(t *testing.T) {
	dir := t.TempDir()

	result, err := statPath(dir, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected error: %v", result.Errors[0].Msg)
	}
	if result.Type != "directory" {
		t.Errorf("expected Type=directory, got %q", result.Type)
	}
}

func TestStat_Symlink(t *testing.T) {
	dir := t.TempDir()
	target := createFile(t, dir, "target.txt", "data\n")
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks not supported:", err)
	}

	result, err := statPath(link, Config{FollowSymlinks: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected error: %v", result.Errors[0].Msg)
	}
	if result.Type != "symlink" {
		t.Errorf("expected Type=symlink, got %q", result.Type)
	}
}

func TestStat_FollowSymlink(t *testing.T) {
	dir := t.TempDir()
	target := createFile(t, dir, "target.txt", "data\n")
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks not supported:", err)
	}

	result, err := statPath(link, Config{FollowSymlinks: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected error: %v", result.Errors[0].Msg)
	}
	if result.Type != "file" {
		t.Errorf("with FollowSymlinks, expected Type=file, got %q", result.Type)
	}
}

func TestStat_Missing(t *testing.T) {
	result, err := statPath(testutil.MissingPath(t, "missing.txt"), Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) == 0 {
		t.Error("expected error for non-existent path")
	}
}

func TestStat_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "hello\n")

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

	var result StatResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}
	if result.XMLName.Local != "stat" {
		t.Errorf("expected root element 'stat', got %q", result.XMLName.Local)
	}
}
