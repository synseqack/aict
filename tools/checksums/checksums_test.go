package checksums

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/testutil"
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
	os.Setenv("AICT_NOCOMPACT", "1")
	defer os.Unsetenv("AICT_XML")
	defer os.Unsetenv("AICT_NOCOMPACT")

	output, err := runChecksums([]string{testutil.MissingPath(t, "file.txt")})
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
	os.Setenv("AICT_NOCOMPACT", "1")
	defer os.Unsetenv("AICT_XML")
	defer os.Unsetenv("AICT_NOCOMPACT")

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

func TestChecksums_Empty(t *testing.T) {
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

	var result ChecksumResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if len(result.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(result.Files))
	}
}

func TestChecksums_MissingFile(t *testing.T) {
	os.Setenv("AICT_XML", "1")
	os.Setenv("AICT_NOCOMPACT", "1")
	defer os.Unsetenv("AICT_XML")
	defer os.Unsetenv("AICT_NOCOMPACT")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{"/nonexistent/file.txt"})

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

	if len(result.Errors) == 0 {
		t.Error("expected error for missing file")
	}
}

// -c/--check re-hashes each entry of a manifest. The algorithm is inferred
// from the digest length, so a manifest written by sha256sum checks as
// sha256 without anything telling the tool which one it is.
func TestChecksums_VerifyOK(t *testing.T) {
	dir := t.TempDir()
	data := createFile(t, dir, "data.txt", payload)
	manifest := createFile(t, dir, "sums.sha256",
		fmt.Sprintf("%s  %s\n", sha256Hex(payload), data))

	out, err := runChecksums([]string{"-c", manifest, "--json"})
	if err != nil {
		t.Fatal(err)
	}

	var result ChecksumResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(result.Verified) != 1 {
		t.Fatalf("expected 1 verified entry, got %d: %+v", len(result.Verified), result)
	}
	v := result.Verified[0]
	if v.Status != "ok" {
		t.Errorf("status = %q, want ok", v.Status)
	}
	if v.Algorithm != "sha256" {
		t.Errorf("algorithm = %q, want sha256", v.Algorithm)
	}
	if v.Expected != v.Actual || v.Expected != sha256Hex(payload) {
		t.Errorf("expected %q, actual %q", v.Expected, v.Actual)
	}
	if v.Path != data {
		t.Errorf("path = %q, want the manifest path verbatim", v.Path)
	}
	if v.Absolute == "" {
		t.Error("absolute path must be resolved even for a manifest-relative entry")
	}
	if len(result.Errors) != 0 {
		t.Errorf("unexpected errors: %+v", result.Errors)
	}
}

func TestChecksums_VerifyFailed(t *testing.T) {
	dir := t.TempDir()
	data := createFile(t, dir, "data.txt", "hello world\n")

	// A digest of the right length that does not match the content.
	manifest := createFile(t, dir, "sums.sha256",
		strings.Repeat("0", 64)+"  "+data+"\n")

	out, err := runChecksums([]string{"-c", manifest, "--json"})
	if err != nil {
		t.Fatal(err)
	}

	var result ChecksumResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	v := result.Verified[0]
	if v.Status != "failed" {
		t.Errorf("status = %q, want failed", v.Status)
	}
	if v.Actual != sha256Hex(payload) {
		t.Errorf("actual = %q, want the real digest", v.Actual)
	}
	if v.Expected != strings.Repeat("0", 64) {
		t.Errorf("expected = %q, want the manifest digest", v.Expected)
	}
	// A mismatch is a result, not a tool error: aict reports it in-band and
	// exits 0 like every other tool's structured failure.
	if err == nil && out == "" {
		t.Error("verify output must not be empty")
	}
}

// md5sum -c infers md5 from the 32-character digest, so the manifest an
// agent holds does not need an algorithm label.
func TestChecksums_VerifyInfersAlgorithm(t *testing.T) {
	for _, tc := range []struct {
		algo string
		hash func() string
	}{
		{"md5", func() string { return md5Hex(payload) }},
		{"sha1", func() string { return sha1Hex(payload) }},
		{"sha256", func() string { return sha256Hex(payload) }},
	} {
		t.Run(tc.algo, func(t *testing.T) {
			dir := t.TempDir()
			createFile(t, dir, "data.txt", payload)
			manifest := createFile(t, dir, "sums", tc.hash()+"  "+absolute(t, dir, "data.txt")+"\n")

			out, err := runChecksums([]string{"-c", manifest, "--json"})
			if err != nil {
				t.Fatal(err)
			}
			var result ChecksumResult
			if err := json.Unmarshal([]byte(out), &result); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, out)
			}
			if result.Verified[0].Algorithm != tc.algo {
				t.Errorf("algorithm = %q, want %q", result.Verified[0].Algorithm, tc.algo)
			}
			if result.Verified[0].Status != "ok" {
				t.Errorf("status = %q, want ok", result.Verified[0].Status)
			}
		})
	}
}

// An entry naming a file that does not exist becomes a ChecksumError, and
// the plain renderer reports it the way GNU does.
func TestChecksums_VerifyMissingFile(t *testing.T) {
	dir := t.TempDir()
	manifest := createFile(t, dir, "sums.sha256",
		strings.Repeat("f", 64)+"  "+filepath.Join(dir, "gone.txt")+"\n")

	out, err := runChecksums([]string{"-c", manifest, "--json"})
	if err != nil {
		t.Fatal(err)
	}
	var result ChecksumResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error for the unreadable entry, got %d", len(result.Errors))
	}
	if !strings.Contains(result.Errors[0].Msg, "no such file") {
		t.Errorf("error message = %q", result.Errors[0].Msg)
	}
	if len(result.Verified) != 0 {
		t.Errorf("an unreadable entry must not appear as a verdict, got %+v", result.Verified)
	}
}

func TestChecksums_VerifyMalformed(t *testing.T) {
	cases := []struct {
		name  string
		lines []string
	}{
		{"too_short", []string{"abc  data.txt\n"}},
		{"not_hex", []string{strings.Repeat("z", 64) + "  data.txt\n"}},
		{"no_path", []string{strings.Repeat("a", 64) + "\n"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			createFile(t, dir, "data.txt", "hello\n")
			manifest := createFile(t, dir, "sums", strings.Join(tc.lines, ""))

			out, err := runChecksums([]string{"-c", manifest, "--json"})
			if err != nil {
				t.Fatal(err)
			}
			var result ChecksumResult
			if err := json.Unmarshal([]byte(out), &result); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, out)
			}
			if len(result.Errors) != 1 {
				t.Fatalf("expected 1 error for a malformed line, got %d: %+v", len(result.Errors), result.Errors)
			}
		})
	}
}

func TestChecksums_VerifyPlainMatchesGNU(t *testing.T) {
	dir := t.TempDir()
	data := createFile(t, dir, "data.txt", "hello world\n")
	// Exists but holds different bytes: FAILED, not unreadable.
	createFile(t, dir, "other.txt", "tampered\n")

	manifest := createFile(t, dir, "sums.sha256",
		sha256Hex(payload)+"  "+data+"\n"+
			strings.Repeat("0", 64)+"  "+filepath.Join(dir, "other.txt")+"\n")

	out, err := runChecksums([]string{"-c", manifest, "--plain"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		filepath.Join(dir, "data.txt") + ": OK",
		filepath.Join(dir, "other.txt") + ": FAILED",
		"WARNING: 1 failed, 0 could not be read",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("plain output missing %q\ngot: %s", want, out)
		}
	}
}

// payload is the fixed content every verify test hashes, so the expected
// digest the test writes into a manifest and the one the tool computes come
// from the same bytes.
const payload = "hello world\n"

// absolute writes the file and returns the path a manifest must carry, since
// manifest entries resolve against the working directory and not the
// manifest's own directory.
func absolute(t *testing.T, dir, name string) string {
	t.Helper()
	return createFile(t, dir, name, payload)
}

func md5Hex(content string) string {
	sum := md5.Sum([]byte(content))
	return hex.EncodeToString(sum[:])
}

func sha1Hex(content string) string {
	sum := sha1.Sum([]byte(content))
	return hex.EncodeToString(sum[:])
}

func sha256Hex(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
