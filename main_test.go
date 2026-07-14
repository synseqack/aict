package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/synseqack/aict/internal/version"
)

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := fn()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func TestVersion(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-V"} {
		output, err := captureStdout(t, func() error { return run([]string{arg}) })
		if err != nil {
			t.Fatalf("%s: %v", arg, err)
		}
		if !strings.HasPrefix(output, "aict "+version.Version) {
			t.Errorf("%s: expected output to start with %q, got %q", arg, "aict "+version.Version, output)
		}
	}
}

func TestUsage_SortedAndDeterministic(t *testing.T) {
	first, err := captureStdout(t, func() error { return run(nil) })
	if err != nil {
		t.Fatal(err)
	}
	second, err := captureStdout(t, func() error { return run(nil) })
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Error("usage output is not deterministic across runs")
	}

	var commands []string
	inCommands := false
	for _, line := range strings.Split(first, "\n") {
		if strings.HasPrefix(line, "Commands:") {
			inCommands = true
			continue
		}
		if inCommands {
			if !strings.HasPrefix(line, "  ") {
				break
			}
			fields := strings.Fields(line)
			if len(fields) > 0 {
				commands = append(commands, fields[0])
			}
		}
	}
	if len(commands) < 30 {
		t.Fatalf("expected at least 30 commands in usage, got %d", len(commands))
	}
	// mcp and version are appended after the sorted tool list
	toolNames := commands[:len(commands)-2]
	for i := 1; i < len(toolNames); i++ {
		if toolNames[i] < toolNames[i-1] {
			t.Errorf("tool list not sorted: %q before %q", toolNames[i-1], toolNames[i])
		}
	}
}

func TestUnknownCommand(t *testing.T) {
	err := run([]string{"definitely-not-a-tool"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}
