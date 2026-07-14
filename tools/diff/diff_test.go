package diff

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
)

func createFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func runDiff(t *testing.T, args []string) *DiffResult {
	t.Helper()
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

	Run(args)
	w.Close()
	os.Stdout = oldStdout
	<-done

	var result DiffResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, outBuf.String())
	}
	return &result
}

func TestDiff_Identical(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.txt", "hello\nworld\n")
	p2 := createFile(t, dir, "b.txt", "hello\nworld\n")

	result := runDiff(t, []string{p1, p2})
	if !result.Identical {
		t.Error("expected Identical=true for same content")
	}
	if result.AddedLines != 0 || result.RemovedLines != 0 {
		t.Errorf("expected 0 added/removed, got added=%d removed=%d", result.AddedLines, result.RemovedLines)
	}
}

func TestDiff_Added(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.txt", "hello\n")
	p2 := createFile(t, dir, "b.txt", "hello\nextra line\n")

	result := runDiff(t, []string{p1, p2})
	if result.Identical {
		t.Error("expected files to differ")
	}
	if result.AddedLines == 0 {
		t.Error("expected AddedLines > 0")
	}
}

func TestDiff_Removed(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.txt", "hello\nworld\n")
	p2 := createFile(t, dir, "b.txt", "hello\n")

	result := runDiff(t, []string{p1, p2})
	if result.Identical {
		t.Error("expected files to differ")
	}
	if result.RemovedLines == 0 {
		t.Error("expected RemovedLines > 0")
	}
}

func TestDiff_Changed(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.txt", "hello\nworld\n")
	p2 := createFile(t, dir, "b.txt", "hello\nearth\n")

	result := runDiff(t, []string{p1, p2})
	if result.Identical {
		t.Error("expected files to differ")
	}
	if result.AddedLines == 0 || result.RemovedLines == 0 {
		t.Errorf("expected both added and removed lines, got added=%d removed=%d", result.AddedLines, result.RemovedLines)
	}
}

func TestDiff_IgnoreSpace(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.txt", "hello world\n")
	p2 := createFile(t, dir, "b.txt", "hello   world\n")

	result := runDiff(t, []string{"-w", p1, p2})
	if !result.Identical {
		t.Error("expected Identical=true with -w (ignore whitespace)")
	}
}

func TestDiff_Quiet(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.txt", "aaa\n")
	p2 := createFile(t, dir, "b.txt", "bbb\n")

	result := runDiff(t, []string{"-q", p1, p2})
	if result.Identical {
		t.Error("expected files to differ with -q flag")
	}
}

func TestDiff_MissingFile(t *testing.T) {
	dir := t.TempDir()
	existing := createFile(t, dir, "a.txt", "hello\n")

	result := runDiff(t, []string{existing, "/nonexistent/missing.txt"})
	if len(result.Errors) == 0 {
		t.Error("expected error for missing file")
	}
}

func TestDiff_XMLValidity(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.txt", "hello\n")
	p2 := createFile(t, dir, "b.txt", "world\n")

	result := runDiff(t, []string{p1, p2})
	if result.XMLName.Local != "diff" {
		t.Errorf("expected root element 'diff', got %q", result.XMLName.Local)
	}
}

// applyEdits reconstructs the new file from oldLines and the edit script.
// Any corruption in the Myers backtracking shows up as a mismatch here.
func applyEdits(t *testing.T, oldLines, newLines []string) {
	t.Helper()
	edits := computeLCS(oldLines, newLines)
	var got []string
	oldIdx := 0
	for _, e := range edits {
		switch e.kind {
		case equal:
			if oldIdx >= len(oldLines) {
				t.Fatalf("equal edit past end of old input (oldIdx=%d)", oldIdx)
			}
			got = append(got, oldLines[oldIdx])
			oldIdx++
		case deleted:
			oldIdx++
		case inserted:
			if e.newIndex >= len(newLines) {
				t.Fatalf("insert edit past end of new input (newIndex=%d)", e.newIndex)
			}
			got = append(got, newLines[e.newIndex])
		}
	}
	if oldIdx != len(oldLines) {
		t.Errorf("edit script consumed %d of %d old lines", oldIdx, len(oldLines))
	}
	if !slicesEqual(got, newLines) {
		t.Errorf("edit script does not reconstruct new file:\n got: %q\nwant: %q", got, newLines)
	}
}

func TestDiff_EditScriptReconstruction(t *testing.T) {
	cases := []struct {
		name     string
		old, new []string
	}{
		{"issue32_repro", []string{"a", "b", "c", "d"}, []string{"a", "X", "c", "d", "e"}},
		{"change_middle", []string{"1", "2", "3"}, []string{"1", "two", "3"}},
		{"prepend", []string{"x", "y"}, []string{"new", "x", "y"}},
		{"append", []string{"x", "y"}, []string{"x", "y", "z"}},
		{"delete_all", []string{"a", "b"}, nil},
		{"insert_all", nil, []string{"a", "b"}},
		{"disjoint", []string{"a", "b", "c"}, []string{"x", "y"}},
		{"interleaved", []string{"a", "b", "c", "d", "e"}, []string{"b", "x", "d", "y", "e"}},
		{"repeated_lines", []string{"a", "a", "b", "a"}, []string{"a", "b", "a", "a"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			applyEdits(t, tc.old, tc.new)
		})
	}
}

func TestDiff_HunkCounts(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "d1.txt", "a\nb\nc\nd\n")
	p2 := createFile(t, dir, "d2.txt", "a\nX\nc\nd\ne\n")

	result := runDiff(t, []string{p1, p2})

	if len(result.Hunks) != 2 {
		t.Fatalf("expected 2 hunks (b→X, +e), got %d: %+v", len(result.Hunks), result.Hunks)
	}

	first := result.Hunks[0]
	if first.OldStart != 2 || first.OldCount != 1 || first.NewStart != 2 || first.NewCount != 1 {
		t.Errorf("first hunk header = -%d,%d +%d,%d; want -2,1 +2,1",
			first.OldStart, first.OldCount, first.NewStart, first.NewCount)
	}

	second := result.Hunks[1]
	if second.OldCount != 0 || second.NewCount != 1 {
		t.Errorf("second hunk header = -%d,%d +%d,%d; want old_count=0 new_count=1",
			second.OldStart, second.OldCount, second.NewStart, second.NewCount)
	}

	// The unchanged line "a" must not appear in any hunk.
	for _, h := range result.Hunks {
		for _, l := range h.Lines {
			if l.Content == "a" {
				t.Errorf("unchanged line %q leaked into a hunk as %q", l.Content, l.Type)
			}
		}
	}
}
