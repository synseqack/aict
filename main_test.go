package main

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestHasNoCompact(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{}, false},
		{[]string{"/tmp/x"}, false},
		{[]string{"--no-compact"}, true},
		{[]string{"/tmp/x", "--no-compact"}, true},
		{[]string{"--no-compact", "/tmp/x"}, true},
		{[]string{"-n", "--no-compact", "/tmp/x"}, true},
		{[]string{"--pretty", "--dict"}, false},
	}
	for _, tt := range tests {
		if got := hasNoCompact(tt.args); got != tt.want {
			t.Errorf("hasNoCompact(%v) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

// TestNoCompactFlagPromotedToEnv pins the dispatcher contract: --no-compact
// must reach the env var every tool's output path reads, not just ls.
func TestNoCompactFlagPromotedToEnv(t *testing.T) {
	t.Setenv("AICT_NOCOMPACT", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := captureStdout(t, func() error { return run([]string{"cat", path, "--no-compact"}) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `path="`) {
		t.Errorf("--no-compact must emit long attribute names, got: %s", out)
	}
	if strings.Contains(out, `p="`) {
		t.Errorf("--no-compact must not emit short attribute names, got: %s", out)
	}

	// The promotion must not leak past this invocation.
	if os.Getenv("AICT_NOCOMPACT") != "1" {
		t.Errorf("expected AICT_NOCOMPACT=1 to persist for the process lifetime, got %q", os.Getenv("AICT_NOCOMPACT"))
	}

	t.Setenv("AICT_NOCOMPACT", "")
	compact, err := captureStdout(t, func() error { return run([]string{"cat", path}) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(compact, `p="`) {
		t.Errorf("default output must emit short attribute names, got: %s", compact)
	}
}
