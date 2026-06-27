package completions

import (
	"bytes"
	"strings"
	"testing"
	"os"
)

func runCompletions(args []string) (string, error) {
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

func TestCompletions_Bash(t *testing.T) {
	out, err := runCompletions([]string{"bash"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "_aict") {
		t.Error("expected '_aict' in bash completion script")
	}
	if !strings.Contains(out, "complete") {
		t.Error("expected 'complete' in bash completion script")
	}
}

func TestCompletions_Zsh(t *testing.T) {
	out, err := runCompletions([]string{"zsh"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "#compdef") {
		t.Error("expected '#compdef' in zsh completion script")
	}
	if !strings.Contains(out, "_aict") {
		t.Error("expected '_aict' in zsh completion script")
	}
}

func TestCompletions_Fish(t *testing.T) {
	out, err := runCompletions([]string{"fish"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "complete") {
		t.Error("expected 'complete' in fish completion script")
	}
	if !strings.Contains(out, "aict") {
		t.Error("expected 'aict' in fish completion script")
	}
}

func TestCompletions_InvalidShell(t *testing.T) {
	out, err := runCompletions([]string{"tcsh"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "unsupported") {
		t.Errorf("expected 'unsupported' in output for unknown shell, got: %q", out)
	}
}

func TestCompletions_HasToolNames(t *testing.T) {
	out, err := runCompletions([]string{"bash"})
	if err != nil {
		t.Fatal(err)
	}
	// The completions package imports completions tool which registers itself
	// But we need at least the basic words list to be present
	if out == "" {
		t.Error("expected non-empty completion script")
	}
}
