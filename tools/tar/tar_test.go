package tar

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/synseqack/aict/internal/testutil"
)

func createTar(t *testing.T, dir, name string, files map[string]string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var tw *tar.Writer
	var gz *gzip.Writer

	if filepath.Ext(name) == ".gz" || filepath.Ext(filepath.Base(name[:len(name)-3])) == ".tar" {
		gz = gzip.NewWriter(f)
		tw = tar.NewWriter(gz)
	} else {
		tw = tar.NewWriter(f)
	}

	for fname, content := range files {
		hdr := &tar.Header{
			Name: fname,
			Mode: 0644,
			Size: int64(len(content)),
		}
		tw.WriteHeader(hdr)
		tw.Write([]byte(content))
	}
	tw.Close()
	if gz != nil {
		gz.Close()
	}
	return path
}

func runTar(t *testing.T, args []string) *TarResult {
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

	var result TarResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}
	return &result
}

func TestTar_ListTar(t *testing.T) {
	dir := t.TempDir()
	archive := createTar(t, dir, "test.tar", map[string]string{
		"hello.txt": "hello world\n",
		"data.txt":  "some data\n",
	})

	result := runTar(t, []string{archive})
	if result.Entries != 2 {
		t.Errorf("expected 2 entries, got %d", result.Entries)
	}
	if len(result.Errors) > 0 {
		t.Errorf("unexpected error: %v", result.Errors[0].Msg)
	}
}

func TestTar_ListTarGz(t *testing.T) {
	dir := t.TempDir()
	archive := createTar(t, dir, "test.tar.gz", map[string]string{
		"hello.txt": "hello\n",
	})

	result := runTar(t, []string{archive})
	if result.Entries != 1 {
		t.Errorf("expected 1 entry, got %d", result.Entries)
	}
	if result.Format != "tar.gz" {
		t.Errorf("expected format=tar.gz, got %q", result.Format)
	}
}

func TestTar_FileTypes(t *testing.T) {
	dir := t.TempDir()
	archive := createTar(t, dir, "test.tar", map[string]string{
		"file.txt": "content\n",
	})

	result := runTar(t, []string{archive})
	if len(result.Files) == 0 {
		t.Fatal("expected at least one file entry")
	}
	for _, f := range result.Files {
		if f.Type == "" {
			t.Errorf("expected Type to be set for entry %q", f.Path)
		}
	}
}

func TestTar_NonExistent(t *testing.T) {
	result := runTar(t, []string{testutil.MissingPath(t, "archive.tar")})
	if len(result.Errors) == 0 {
		t.Error("expected error for non-existent archive")
	}
}

func TestTar_NonTarFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notatar.txt")
	os.WriteFile(path, []byte("this is not a tar file"), 0644)

	result := runTar(t, []string{path})
	if len(result.Errors) == 0 {
		t.Error("expected error for non-tar file")
	}
}

func TestTar_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	archive := createTar(t, dir, "test.tar", map[string]string{
		"a.txt": "aaa",
	})

	result := runTar(t, []string{archive})
	if result.XMLName.Local != "tar" {
		t.Errorf("expected root element 'tar', got %q", result.XMLName.Local)
	}
}
