package doctor

import (
	"bytes"
	"encoding/xml"
	"os"
	"testing"
)

func TestDoctor_Runs(t *testing.T) {
	result := runDiagnostics()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestDoctor_HasChecks(t *testing.T) {
	result := runDiagnostics()
	if len(result.Checks) == 0 {
		t.Error("expected at least one diagnostic check")
	}
}

func TestDoctor_SummarySet(t *testing.T) {
	result := runDiagnostics()
	if result.Summary == "" {
		t.Error("expected Summary to be non-empty")
	}
}

func TestDoctor_ChecksHaveStatus(t *testing.T) {
	result := runDiagnostics()
	for _, c := range result.Checks {
		if c.Name == "" {
			t.Error("check has empty Name")
		}
		switch c.Status {
		case "pass", "warning", "fail":
		default:
			t.Errorf("unexpected check status %q for check %q", c.Status, c.Name)
		}
	}
}

func TestDoctor_XMLValidity(t *testing.T) {
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

	var result DoctorResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}
	if result.XMLName.Local != "doctor" {
		t.Errorf("expected root element 'doctor', got %q", result.XMLName.Local)
	}
}
