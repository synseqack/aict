package ripgrep

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// sherlock is the shape of real rg --json output, modelled on the example
// in the grep_printer::JSON documentation. The JSON line-terminator escapes
// inside lines.text are spelled "\\n" because a raw string cannot hold a
// literal backslash-n without collapsing it.
const beginLine = `{"type":"begin","data":{"path":{"text":"sherlock"}}}`

const matchLine = `{"type":"match","data":{"path":{"text":"sherlock"},"lines":{"text":"For the Doctor Watsons of this world, as opposed to the Sherlock` + "\\n" + `"},"line_number":1,"absolute_offset":0,"submatches":[{"match":{"text":"Sherlock"},"start":37,"end":44}]}}`

const contextLine = `{"type":"context","data":{"path":{"text":"sherlock"},"lines":{"text":"well that he was not himself a qualified surveyor.` + "\\n" + `"},"line_number":2,"absolute_offset":92,"submatches":[]}}`

const endLine = `{"type":"end","data":{"path":{"text":"sherlock"},"binary_offset":null,"stats":{"elapsed":{"secs":0,"nanos":56965,"human":"0.000056s"},"searches":1,"searches_with_match":1,"bytes_searched":1053,"bytes_printed":168,"matched_lines":1,"matches":1}}}`

var sherlock = strings.Join([]string{beginLine, matchLine, contextLine, endLine}, "\n")

func TestDecoder_ReadsMatch(t *testing.T) {
	var result Result
	dec := newDecoder(&result)
	for _, line := range strings.Split(sherlock, "\n") {
		if line == "" {
			continue
		}
		if err := dec.add([]byte(line)); err != nil {
			t.Fatalf("add(%q): %v", line, err)
		}
	}

	want := &Result{Files: []FileMatches{{
		Path: "sherlock",
		Matches: []Match{{
			LineNumber:     1,
			Text:           "For the Doctor Watsons of this world, as opposed to the Sherlock",
			AbsoluteOffset: 0,
		}},
	}}}
	if !reflect.DeepEqual(result, *want) {
		t.Errorf("got %+v, want %+v", result, *want)
	}
}

func TestDecoder_DecodesBase64Lines(t *testing.T) {
	// "\xff\xfeSherlock" is invalid UTF-8, so rg sends it as base64 bytes.
	want := "\xff\xfeSherlock"
	line := `{"type":"match","data":{"path":{"text":"bin"},"lines":{"bytes":"` +
		base64.StdEncoding.EncodeToString([]byte(want)) +
		`"},"line_number":3,"absolute_offset":40,"submatches":[]}}`

	var result Result
	if err := newDecoder(&result).add([]byte(line)); err != nil {
		t.Fatalf("add: %v", err)
	}

	if got := result.Files[0].Matches[0].Text; got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
}

func TestDecoder_IgnoresOtherMessageTypes(t *testing.T) {
	lines := []string{
		`{"type":"summary","data":{"elapsed_total":{"secs":0,"nanos":1000,"human":"0.000001s"},"stats":{"searches":1,"searches_with_match":1,"bytes_searched":1053,"bytes_printed":168,"matched_lines":1,"matches":1}}}`,
		`{"type":"context","data":{"path":{"text":"sherlock"},"lines":{"text":"context line\n"},"line_number":2,"absolute_offset":92,"submatches":[]}}`,
		`{"type":"end","data":{"path":{"text":"sherlock"},"binary_offset":null,"stats":{}}}`,
	}

	var result Result
	dec := newDecoder(&result)
	for _, line := range lines {
		if err := dec.add([]byte(line)); err != nil {
			t.Fatalf("add(%q): %v", line, err)
		}
	}
	if len(result.Files) != 0 {
		t.Errorf("expected no files, got %d", len(result.Files))
	}
}

func TestDecoder_GroupsByPath(t *testing.T) {
	lines := []string{
		`{"type":"match","data":{"path":{"text":"a"},"lines":{"text":"one\n"},"line_number":1,"absolute_offset":0,"submatches":[]}}`,
		`{"type":"match","data":{"path":{"text":"b"},"lines":{"text":"two\n"},"line_number":5,"absolute_offset":9,"submatches":[]}}`,
		`{"type":"match","data":{"path":{"text":"a"},"lines":{"text":"three\n"},"line_number":2,"absolute_offset":4,"submatches":[]}}`,
	}

	var result Result
	dec := newDecoder(&result)
	for _, line := range lines {
		if err := dec.add([]byte(line)); err != nil {
			t.Fatalf("add(%q): %v", line, err)
		}
	}

	if len(result.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result.Files))
	}
	if got := len(result.Files[0].Matches); got != 2 {
		t.Errorf("file a: %d matches, want 2", got)
	}
	if got := result.Files[1].Path; got != "b" {
		t.Errorf("second file = %q, want b", got)
	}
}

func TestArgs_Argv(t *testing.T) {
	got := Args{
		Pattern:         "foo|bar",
		CaseInsensitive: true,
		WordMatch:       true,
		FixedStrings:    true,
		InvertMatch:     true,
		MaxCount:        10,
	}.argv([]string{"/a", "/b"})

	want := []string{
		"--json", "--no-ignore", "--hidden",
		"-i", "-w", "-F", "-v", "-m", "10",
		"--", "foo|bar", "/a", "/b",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestChunkFiles(t *testing.T) {
	// Each path costs its length plus a separator; force a split by making
	// the second file exceed the budget on its own is not possible, so use a
	// tiny budget via the real ceiling by passing many long paths.
	long := strings.Repeat("x", maxArgvBytes-8)
	files := []string{"/short", long, "/after"}

	chunks := chunkFiles(files)
	if len(chunks) < 2 {
		t.Fatalf("expected chunking, got %d chunk(s)", len(chunks))
	}

	// Round trip: every input file appears exactly once, in order.
	var seen []string
	for _, c := range chunks {
		seen = append(seen, c...)
	}
	if !reflect.DeepEqual(seen, files) {
		t.Errorf("chunking lost or reordered files: %v vs %v", seen, files)
	}
}

func TestSearch_NoFilesSpawnsNothing(t *testing.T) {
	res, err := Search(context.Background(), Args{Pattern: "x"}, nil)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if res == nil || len(res.Files) != 0 {
		t.Errorf("expected empty result, got %+v", res)
	}
}

// writeStub installs an executable named rg into a temp dir and puts it on
// PATH. body is printed for a search; the stub reports a ripgrep version so
// Available accepts it. The body is handed to the stub through the process
// environment so no quoting is needed.
func writeStub(t *testing.T, body string, exitCode int) {
	t.Helper()

	dir := t.TempDir()
	bodyFile := filepath.Join(dir, "body.json")
	if err := os.WriteFile(bodyFile, []byte(body), 0o644); err != nil {
		t.Fatalf("write body: %v", err)
	}

	script := "#!/bin/sh\n" +
		`if [ "$1" = "--version" ]; then echo "ripgrep 14.1.1"; exit 0; fi` + "\n" +
		`cat "$STUB_BODY"` + "\n" +
		"exit " + strconv.Itoa(exitCode) + "\n"
	stub := filepath.Join(dir, "rg")
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}

	t.Setenv("STUB_BODY", bodyFile)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestSearch_StubOnPath(t *testing.T) {
	writeStub(t, sherlock, 0)

	res, err := Search(context.Background(), Args{Pattern: "Sherlock"}, []string{"sherlock"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Files) != 1 || res.Files[0].Path != "sherlock" {
		t.Fatalf("got %+v", res.Files)
	}
	if got := res.Files[0].Matches[0].LineNumber; got != 1 {
		t.Errorf("line number = %d, want 1", got)
	}
}

func TestSearch_ExitCodeOneIsNoMatches(t *testing.T) {
	writeStub(t, "", 1)

	res, err := Search(context.Background(), Args{Pattern: "missing"}, []string{"sherlock"})
	if err != nil {
		t.Fatalf("rg exit 1 must be treated as success, got: %v", err)
	}
	if len(res.Files) != 0 {
		t.Errorf("expected no files, got %d", len(res.Files))
	}
}

func TestSearch_ExitCodeTwoIsAnError(t *testing.T) {
	writeStub(t, "", 2)

	if _, err := Search(context.Background(), Args{Pattern: "x"}, []string{"sherlock"}); err == nil {
		t.Error("expected an error for rg exit 2")
	}
}

func TestSearch_AICTNORGDisables(t *testing.T) {
	writeStub(t, sherlock, 0)
	t.Setenv("AICT_NORG", "1")

	if _, err := Search(context.Background(), Args{Pattern: "x"}, []string{"sherlock"}); err != ErrNotFound {
		t.Errorf("got %v, want ErrNotFound", err)
	}
	if Available() {
		t.Error("Available() = true, want false under AICT_NORG=1")
	}
}

func TestSearch_StubMissingFromPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	if Available() {
		t.Error("Available() = true with no rg on PATH")
	}
	if _, err := Search(context.Background(), Args{Pattern: "x"}, []string{"sherlock"}); err != ErrNotFound {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}
