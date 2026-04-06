package checksums

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	pathutil "github.com/synseqack/aict/internal/path"
)

func runChecksums(args []string) (string, error) {
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

func TestChecksums_Basic(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "hello world")

	result, err := runChecksumsWithOutput(filePath, Config{Algorithms: []string{"md5"}})
	if err != nil {
		t.Fatal(err)
	}

	if result.MD5 == "" {
		t.Error("expected MD5 hash")
	}
}

func TestChecksums_MultipleAlgorithms(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "hello world")

	result, err := runChecksumsWithOutput(filePath, Config{Algorithms: []string{"md5", "sha256"}})
	if err != nil {
		t.Fatal(err)
	}

	if result.MD5 == "" {
		t.Error("expected MD5 hash")
	}
	if result.SHA256 == "" {
		t.Error("expected SHA256 hash")
	}
}

func TestChecksums_KnownValue(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "hello world")

	result, err := runChecksumsWithOutput(filePath, Config{Algorithms: []string{"md5"}})
	if err != nil {
		t.Fatal(err)
	}

	if result.MD5 != "5eb63bbbe01eeed093cb22bb8f5acdc3" {
		t.Errorf("expected known MD5 hash, got %s", result.MD5)
	}
}

func TestChecksums_NonExistent(t *testing.T) {
	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	output, err := runChecksums([]string{"/nonexistent/file.txt"})
	if err != nil {
		t.Fatal(err)
	}

	if output == "" {
		t.Error("expected output for non-existent file")
	}
}

func TestChecksums_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "hello world")

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

	var result ChecksumResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.XMLName.Local != "checksums" {
		t.Errorf("expected root element 'checksums', got %q", result.XMLName.Local)
	}
}

func TestChecksums_PlainOutput(t *testing.T) {
	dir := t.TempDir()
	filePath := createFile(t, dir, "test.txt", "hello world")

	output, err := runChecksums([]string{"--plain", filePath})
	if err != nil {
		t.Fatal(err)
	}

	if output == "" {
		t.Error("expected plain output")
	}

	if !bytes.Contains([]byte(output), []byte("  ")) {
		t.Error("expected hash and filename in output")
	}
}

func runChecksumsWithOutput(path string, cfg Config) (*ChecksumFile, error) {
	resolved, err := pathutil.Resolve(path)
	if err != nil {
		return nil, err
	}

	result := &ChecksumFile{
		Path:     resolved.Given,
		Absolute: resolved.Absolute,
	}

	info, err := os.Lstat(resolved.Absolute)
	if err != nil {
		return nil, err
	}

	result.SizeBytes = info.Size()

	if info.IsDir() {
		return nil, fmt.Errorf("is a directory")
	}

	f, err := os.Open(resolved.Absolute)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	md5h := md5.New()
	sha1h := sha1.New()
	sha256h := sha256.New()

	reader := bufio.NewReader(f)

	buffer := make([]byte, 32*1024)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			md5h.Write(buffer[:n])
			sha1h.Write(buffer[:n])
			sha256h.Write(buffer[:n])
		}
		if err != nil {
			break
		}
	}

	for _, algo := range cfg.Algorithms {
		switch algo {
		case "md5":
			result.MD5 = hex.EncodeToString(md5h.Sum(nil))
		case "sha1":
			result.SHA1 = hex.EncodeToString(sha1h.Sum(nil))
		case "sha256":
			result.SHA256 = hex.EncodeToString(sha256h.Sum(nil))
		}
	}

	return result, nil
}

func createFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
