package tr

import (
	"bytes"
	"encoding/xml"
	"os"
	"testing"
)

func TestTr_Basic(t *testing.T) {
	_, err := parseFlags([]string{})
	if err == nil {
		t.Error("expected error for empty args")
	}
}

func TestTr_Translate(t *testing.T) {
	cfg, err := parseFlags([]string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Set1 != "a" || cfg.Set2 != "b" {
		t.Errorf("expected sets a->b, got %q->%q", cfg.Set1, cfg.Set2)
	}
}

func TestTr_Delete(t *testing.T) {
	cfg, err := parseFlags([]string{"-d", "abc"})
	if err != nil {
		t.Fatal(err)
	}

	if !cfg.Delete {
		t.Error("expected delete flag")
	}
	if cfg.Set1 != "abc" {
		t.Errorf("expected set1 'abc', got %q", cfg.Set1)
	}
}

func TestTr_Squeeze(t *testing.T) {
	cfg, err := parseFlags([]string{"-s", "a"})
	if err != nil {
		t.Fatal(err)
	}

	if !cfg.Squeeze {
		t.Error("expected squeeze flag")
	}
}

func TestTr_MissingOperand(t *testing.T) {
	_, err := parseFlags([]string{})
	if err == nil {
		t.Error("expected error for missing operand")
	}
}

func TestTr_XMLValidity(t *testing.T) {
	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.WriteString("hello\n")
		w.Close()
	}()

	oldStdout := os.Stdout
	outR, outW, _ := os.Pipe()
	os.Stdout = outW

	err := Run([]string{})

	outW.Close()
	os.Stdout = oldStdout

	os.Stdin = oldStdin

	if err == nil {
		var outBuf bytes.Buffer
		outBuf.ReadFrom(outR)

		var result TrResult
		if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
			t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
		}

		if result.XMLName.Local != "tr" {
			t.Errorf("expected root element 'tr', got %q", result.XMLName.Local)
		}
	}
}

func TestTr_ExpandSet(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a-z", "abcdefghijklmnopqrstuvwxyz"},
		{"A-Z", "ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{"0-9", "0123456789"},
		{"\\n", "\n"},
		{"\\t", "\t"},
	}

	for _, tt := range tests {
		result := expandSet(tt.input)
		if result != tt.expected {
			t.Errorf("expandSet(%q): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}
