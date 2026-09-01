package dirname

import (
	"bytes"
	"encoding/xml"
	"os"
	"testing"
)

func runDirname(args []string) (string, error) {
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

func TestDirname_Basic(t *testing.T) {
	os.Setenv("AICT_XML", "1")
	os.Setenv("AICT_NOCOMPACT", "1")
	defer os.Unsetenv("AICT_XML")
	defer os.Unsetenv("AICT_NOCOMPACT")

	result, err := runDirname([]string{"/path/to/file.txt"})
	if err != nil {
		t.Fatal(err)
	}

	if result == "" {
		t.Error("expected output")
	}
}

func TestDirname_XMLValidity(t *testing.T) {
	os.Setenv("AICT_XML", "1")
	os.Setenv("AICT_NOCOMPACT", "1")
	defer os.Unsetenv("AICT_XML")
	defer os.Unsetenv("AICT_NOCOMPACT")

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

	var result DirnameResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.XMLName.Local != "dirname" {
		t.Errorf("expected root element 'dirname', got %q", result.XMLName.Local)
	}
}

func TestDirname_Path(t *testing.T) {
	result, err := runDirname([]string{"/path/to/file.txt"})
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains([]byte(result), []byte("/path/to")) {
		t.Error("expected /path/to in output")
	}
}

func TestDirname_Directory(t *testing.T) {
	result, err := runDirname([]string{"/path/to/dir/"})
	if err != nil {
		t.Fatal(err)
	}

	if result == "" {
		t.Error("expected output")
	}
}

func TestDirname_PlainOutput(t *testing.T) {
	result, err := runDirname([]string{"--plain", "/path/to/file.txt"})
	if err != nil {
		t.Fatal(err)
	}

	if result == "" {
		t.Error("expected output")
	}
}

func TestDirname_Empty(t *testing.T) {
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

	var result DirnameResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if len(result.Paths) != 0 {
		t.Errorf("expected 0 paths, got %d", len(result.Paths))
	}
}

func TestDirname_Multiple(t *testing.T) {
	os.Setenv("AICT_XML", "1")
	os.Setenv("AICT_NOCOMPACT", "1")
	defer os.Unsetenv("AICT_XML")
	defer os.Unsetenv("AICT_NOCOMPACT")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{"/a/b/c", "/x/y/z"})

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	if err != nil {
		t.Fatal(err)
	}

	var result DirnameResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if len(result.Paths) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result.Paths))
	}
}
