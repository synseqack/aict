package pwd

import (
	"bytes"
	"encoding/xml"
	"os"
	"testing"
)

func runPwd(args []string) (string, error) {
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

func TestPwd_Basic(t *testing.T) {
	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	output, err := runPwd([]string{})
	if err != nil {
		t.Fatal(err)
	}

	if output == "" {
		t.Error("expected output")
	}
}

func TestPwd_XMLValidity(t *testing.T) {
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

	var result PwdResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.XMLName.Local != "pwd" {
		t.Errorf("expected root element 'pwd', got %q", result.XMLName.Local)
	}
}

func TestPwd_PlainOutput(t *testing.T) {
	output, err := runPwd([]string{"--plain"})
	if err != nil {
		t.Fatal(err)
	}

	if output == "" {
		t.Error("expected output")
	}
}
