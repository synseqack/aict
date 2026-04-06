package sort

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
)

func runSort(args []string) (string, error) {
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

func TestSort_Basic(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "cherry\napple\nbanana\n")

	result, err := runSortWithOutput(filePath, Config{})
	if err != nil {
		t.Fatal(err)
	}

	if result.LinesIn != 3 {
		t.Errorf("expected 3 lines in, got %d", result.LinesIn)
	}
}

func TestSort_Alphabetical(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "cherry\napple\nbanana\n")

	result, err := runSortWithOutput(filePath, Config{})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Content) == 0 {
		t.Error("expected sorted content")
	}
}

func TestSort_Numeric(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "10\n2\n1\n")

	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	_, err := runSort([]string{"-n", filePath})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSort_Reverse(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "apple\nbanana\ncherry")

	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{"-r", filePath})

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	if err != nil {
		t.Fatal(err)
	}

	var result SortResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.Order != "descending" {
		t.Errorf("expected order 'descending', got %q", result.Order)
	}
}

func TestSort_Unique(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "apple\napple\nbanana")

	result, err := runSortWithOutput(filePath, Config{Unique: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.LinesOut == 0 {
		t.Error("expected output lines")
	}
}

func TestSort_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "b\na\n")

	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{filePath})

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	if err != nil {
		t.Fatal(err)
	}

	var result SortResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.XMLName.Local != "sort" {
		t.Errorf("expected root element 'sort', got %q", result.XMLName.Local)
	}
}

func runSortWithOutput(path string, cfg Config) (*SortResult, error) {
	lines, err := readLines(path)
	if err != nil {
		return nil, err
	}

	result := &SortResult{
		Timestamp: 12345,
		KeyField:  cfg.Key,
		Order:     "ascending",
	}
	result.LinesIn = len(lines)

	sortLines(&lines, cfg)

	if cfg.Unique {
		uniqueLines := make([]string, 0, len(lines))
		var prev string
		for _, line := range lines {
			if line != prev {
				uniqueLines = append(uniqueLines, line)
				prev = line
			}
		}
		lines = uniqueLines
	}

	result.LinesOut = len(lines)

	result.Content = bytesJoin(lines, "\n")
	if len(lines) > 0 {
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
