package cut

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
)

func runCut(args []string) (string, error) {
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

func TestCut_Basic(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "a\tb\tc\n")

	result, err := runCutWithOutput(filePath, Config{Fields: "1"})
	if err != nil {
		t.Fatal(err)
	}

	if result.LinesProcessed == 0 {
		t.Error("expected lines processed")
	}
}

func TestCut_FieldExtraction(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "a\tb\tc\nd\te\tf\n")

	result, err := runCutWithOutput(filePath, Config{Fields: "1,3", Delimiter: "\t"})
	if err != nil {
		t.Fatal(err)
	}

	if result.Content == "" {
		t.Error("expected content after cut")
	}
}

func TestCut_Delimiter(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "a|b|c")

	result, err := runCutWithOutput(filePath, Config{Fields: "1", Delimiter: "|"})
	if err != nil {
		t.Fatal(err)
	}

	if result.Delimiter != "|" {
		t.Errorf("expected delimiter '|', got %q", result.Delimiter)
	}
}

func TestCut_Characters(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "abcdef")

	result, err := runCutWithOutput(filePath, Config{Characters: "1-3"})
	if err != nil {
		t.Fatal(err)
	}

	if result.Content == "" {
		t.Error("expected content after character cut")
	}
}

func TestCut_OnlyDelimited(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "a\tb\nno delimiter here")

	result, err := runCutWithOutput(filePath, Config{Fields: "1", OnlyDelim: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.Content == "" {
		t.Error("expected content with only-delimited")
	}
}

func TestCut_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "a\tb\tc\n")

	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{"-f", "1", filePath})

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	if err != nil {
		t.Fatal(err)
	}

	var result CutResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.XMLName.Local != "cut" {
		t.Errorf("expected root element 'cut', got %q", result.XMLName.Local)
	}
}

func runCutWithOutput(path string, cfg Config) (*CutResult, error) {
	lines, err := readLines(path)
	if err != nil {
		return nil, err
	}

	result := &CutResult{
		Timestamp:      12345,
		Delimiter:      cfg.Delimiter,
		Fields:         cfg.Fields,
		LinesProcessed: len(lines),
	}

	cutLines := processCut(lines, cfg)
	result.Content = bytesJoin(cutLines, "\n")
	if len(cutLines) > 0 {
		result.Content += "\n"
	}

	return result, nil
}

func bytesJoin(s []string, sep string) string {
	result := ""
	for i, s := range s {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func createFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
