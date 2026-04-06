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
	defer os.Unsetenv("AICT_XML")

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
