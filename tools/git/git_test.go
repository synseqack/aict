package git

import (
	"bytes"
	"encoding/xml"
	"os"
	"strings"
	"testing"
)

func runGitCmd(t *testing.T, args []string) *GitResult {
	t.Helper()
	os.Setenv("AICT_XML", "1")
	os.Setenv("AICT_NOCOMPACT", "1")
	defer os.Unsetenv("AICT_XML")
	defer os.Unsetenv("AICT_NOCOMPACT")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var outBuf bytes.Buffer
	done := make(chan struct{})
	go func() {
		outBuf.ReadFrom(r)
		close(done)
	}()

	Run(args)
	w.Close()
	os.Stdout = oldStdout
	<-done

	var result GitResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}
	return &result
}

func TestGit_Status(t *testing.T) {
	result := runGitCmd(t, []string{"status"})
	if result.Subcmd != "status" {
		t.Errorf("expected Subcmd=status, got %q", result.Subcmd)
	}
}

func TestGit_Log(t *testing.T) {
	result := runGitCmd(t, []string{"log"})
	if len(result.Log) == 0 {
		t.Error("expected at least one commit in log")
	}
	for _, c := range result.Log {
		if c.Hash == "" {
			t.Error("expected commit Hash to be set")
		}
		if c.Message == "" {
			t.Error("expected commit Message to be set")
		}
		if c.Author == "" {
			t.Error("expected commit Author to be set")
		}
	}
}

func TestGit_LsFiles(t *testing.T) {
	result := runGitCmd(t, []string{"ls-files"})
	if len(result.Files) == 0 {
		t.Error("expected at least one file in ls-files output")
	}

	found := false
	for _, f := range result.Files {
		if strings.Contains(f.Path, "git.go") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected git.go to appear in ls-files output")
	}
}

func TestGit_InvalidSubcmd(t *testing.T) {
	err := Run([]string{"nonexistentsubcmd"})
	if err == nil {
		t.Error("expected error for unknown git subcommand")
	}
}

func TestGit_XMLValidity(t *testing.T) {
	result := runGitCmd(t, []string{"status"})
	if result.XMLName.Local != "git" {
		t.Errorf("expected root element 'git', got %q", result.XMLName.Local)
	}
}
