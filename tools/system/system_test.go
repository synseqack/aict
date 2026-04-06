package system

import (
	"bytes"
	"encoding/xml"
	"os"
	"testing"
)

func runSystem(args []string) (string, error) {
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

func TestSystem_Basic(t *testing.T) {
	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	output, err := runSystem([]string{})
	if err != nil {
		t.Fatal(err)
	}

	if output == "" {
		t.Error("expected output")
	}
}

func TestSystem_XMLValidity(t *testing.T) {
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

	var result SystemResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.XMLName.Local != "system" {
		t.Errorf("expected root element 'system', got %q", result.XMLName.Local)
	}
}

func TestSystem_UserInfo(t *testing.T) {
	result := getUserInfo()

	if result.Username == "" {
		t.Error("expected username to be set")
	}
	if result.UID == "" {
		t.Error("expected uid to be set")
	}
}

func TestSystem_OSInfo(t *testing.T) {
	result := getOSInfo()

	if result.GOOS == "" {
		t.Error("expected GOOS to be set")
	}
	if result.GOARCH == "" {
		t.Error("expected GOARCH to be set")
	}
}

func TestSystem_RuntimeInfo(t *testing.T) {
	result := getRuntimeInfo()

	if result.Version == "" {
		t.Error("expected runtime version to be set")
	}
	if result.NumCPU == 0 {
		t.Error("expected num cpu to be set")
	}
}

func TestSystem_PlainOutput(t *testing.T) {
	output, err := runSystem([]string{"--plain"})
	if err != nil {
		t.Fatal(err)
	}

	if output == "" {
		t.Error("expected plain output")
	}

	if !bytes.Contains([]byte(output), []byte("User:")) {
		t.Error("expected User: in plain output")
	}
}
