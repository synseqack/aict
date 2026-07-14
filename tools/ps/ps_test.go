package ps

import (
	"bytes"
	"encoding/xml"
	"os"
	"runtime"
	"testing"
)

func runPs(args []string) (string, error) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var outBuf bytes.Buffer
	done := make(chan struct{})
	go func() {
		outBuf.ReadFrom(r)
		close(done)
	}()

	err := Run(args)

	w.Close()
	os.Stdout = oldStdout
	<-done

	return outBuf.String(), err
}

func TestPs_Basic(t *testing.T) {
	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	output, err := runPs([]string{})
	if err != nil {
		t.Fatal(err)
	}

	if output == "" {
		t.Error("expected output")
	}
}

func TestPs_XMLValidity(t *testing.T) {
	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var outBuf bytes.Buffer
	done := make(chan struct{})
	go func() {
		outBuf.ReadFrom(r)
		close(done)
	}()

	err := Run([]string{})

	w.Close()
	os.Stdout = oldStdout
	<-done

	if err != nil {
		t.Fatal(err)
	}

	var result PsResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}

	if result.XMLName.Local != "ps" {
		t.Errorf("expected root element 'ps', got %q", result.XMLName.Local)
	}
}

func TestPs_HasProcesses(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ps reads /proc, which is linux-only")
	}
	result, err := getProcesses(Config{})
	if err != nil {
		t.Fatal(err)
	}

	if len(result) == 0 {
		t.Error("expected at least one process")
	}

	hasValidProcess := false
	for _, p := range result {
		if p.PID == 0 {
			t.Error("expected valid PID")
		}
		if p.Command != "" {
			hasValidProcess = true
		}
	}

	if !hasValidProcess {
		t.Error("expected at least one process with command set")
	}
}

func TestPs_PlainOutput(t *testing.T) {
	output, err := runPs([]string{"--plain"})
	if err != nil {
		t.Fatal(err)
	}

	if output == "" {
		t.Error("expected plain output")
	}

	if !bytes.Contains([]byte(output), []byte("USER")) {
		t.Error("expected USER header in plain output")
	}
}

func TestPs_StateDescription(t *testing.T) {
	tests := []struct {
		state string
		desc  string
	}{
		{"R", "running"},
		{"S", "sleeping"},
		{"D", "disk sleep"},
		{"Z", "zombie"},
		{"T", "stopped"},
		{"X", "dead"},
		{"I", "idle"},
		{"i", "idle"},
	}

	for _, tt := range tests {
		result := getStateDescription(tt.state)
		if result != tt.desc {
			t.Errorf("expected %q, got %q", tt.desc, result)
		}
	}
}
