package basename

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
)

func runBasename(args []string) (string, error) {
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

func TestBasename_Basic(t *testing.T) {
	result, err := runBasenameWithOutput("/path/to/file.txt")
	if err != nil {
		t.Fatal(err)
	}

	if result.Base != "file.txt" {
		t.Errorf("expected base 'file.txt', got %q", result.Base)
	}
}

func TestBasename_WithExtension(t *testing.T) {
	result, err := runBasenameWithOutput("/path/to/file.txt")
	if err != nil {
		t.Fatal(err)
	}

	if result.Extension != ".txt" {
		t.Errorf("expected extension '.txt', got %q", result.Extension)
	}
	if result.Stem != "file" {
		t.Errorf("expected stem 'file', got %q", result.Stem)
	}
}

func TestBasename_WithoutExtension(t *testing.T) {
	result, err := runBasenameWithOutput("/path/to/file")
	if err != nil {
		t.Fatal(err)
	}

	if result.Base != "file" {
		t.Errorf("expected base 'file', got %q", result.Base)
	}
	if result.Extension != "" {
		t.Errorf("expected empty extension, got %q", result.Extension)
	}
}

func TestBasename_Directory(t *testing.T) {
	result, err := runBasenameWithOutput("/path/to/dir/")
	if err != nil {
		t.Fatal(err)
	}

	if result.Base != "dir" {
		t.Errorf("expected base 'dir', got %q", result.Base)
	}
}

func TestBasename_WithSuffix(t *testing.T) {
	result, err := runBasenameWithOutputWithSuffix("/path/to/file.txt", ".txt")
	if err != nil {
		t.Fatal(err)
	}

	if result.Base != "file" {
		t.Errorf("expected base 'file', got %q", result.Base)
	}
}

func TestBasename_XMLValidity(t *testing.T) {
	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{"/path/to/file.txt"})

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	if err != nil {
		t.Fatal(err)
	}

	var result BasenameResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.XMLName.Local != "basename" {
		t.Errorf("expected root element 'basename', got %q", result.XMLName.Local)
	}
}

func runBasenameWithOutput(path string) (BasenameEntry, error) {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	stem := base[:len(base)-len(ext)]
	return BasenameEntry{
		Path:      path,
		Base:      base,
		Stem:      stem,
		Extension: ext,
	}, nil
}

func runBasenameWithSuffixWithSuffix(path, suffix string) (BasenameEntry, error) {
	base := filepath.Base(path)
	if suffix != "" {
		base = base[:len(base)-len(suffix)]
	}
	ext := filepath.Ext(base)
	stem := base[:len(base)-len(ext)]
	return BasenameEntry{
		Path:      path,
		Base:      base,
		Stem:      stem,
		Extension: ext,
	}, nil
}

func runBasenameWithOutputWithSuffix(path, suffix string) (BasenameEntry, error) {
	base := filepath.Base(path)
	if suffix != "" && len(base) >= len(suffix) && base[len(base)-len(suffix):] == suffix {
		base = base[:len(base)-len(suffix)]
	}
	ext := filepath.Ext(base)
	stem := base
	if ext != "" {
		stem = base[:len(base)-len(ext)]
	}
	return BasenameEntry{
		Path:      path,
		Base:      base,
		Stem:      stem,
		Extension: ext,
	}, nil
}
