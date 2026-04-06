package df

import (
	"bytes"
	"encoding/xml"
	"os"
	"strings"
	"testing"
)

func runDf(args []string) (string, error) {
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

func TestDf_Basic(t *testing.T) {
	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	output, err := runDf([]string{})
	if err != nil {
		t.Fatal(err)
	}

	if output == "" {
		t.Error("expected output")
	}
}

func TestDf_XMLValidity(t *testing.T) {
	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

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

	var result DfResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.XMLName.Local != "df" {
		t.Errorf("expected root element 'df', got %q", result.XMLName.Local)
	}
}

func TestDf_HumanReadable(t *testing.T) {
	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run([]string{"-h"})

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	if err != nil {
		t.Fatal(err)
	}

	var result DfResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if len(result.Filesystems) == 0 {
		t.Error("expected filesystem entries")
	}
}

func TestDf_PlainOutput(t *testing.T) {
	output, err := runDf([]string{"--plain"})
	if err != nil {
		t.Fatal(err)
	}

	if output == "" {
		t.Error("expected plain output")
	}

	if !bytes.Contains([]byte(output), []byte("Filesystem")) {
		t.Error("expected filesystem header in plain output")
	}
}

func TestDf_HasFilesystems(t *testing.T) {
	result, err := getFilesystems(Config{})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Filesystems) == 0 {
		t.Skip("no filesystems found")
	}

	for _, fs := range result.Filesystems {
		if fs.Device == "" {
			t.Error("expected device to be set")
		}
		if fs.Mount == "" {
			t.Error("expected mount point to be set")
		}
		if fs.SizeBytes == 0 && fs.Type != "sysfs" && fs.Type != "proc" && fs.Type != "devpts" && fs.Type != "cgroup2" && fs.Type != "pstore" && fs.Type != "debugfs" && fs.Type != "tracefs" && fs.Type != "mqueue" && fs.Type != "hugetlbfs" && fs.Type != "fusectl" && fs.Type != "configfs" && fs.Type != "securityfs" && fs.Type != "bpf" && fs.Type != "autofs" && fs.Type != "binfmt_misc" && fs.Type != "nsfs" && !strings.HasPrefix(fs.Device, "/dev/loop") {
			t.Error("expected size to be set")
		}
	}
}
