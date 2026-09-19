package diff

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/synseqack/aict/internal/testutil"
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
	os.Setenv("AICT_NOCOMPACT", "1")
	os.Setenv("AICT_NOCOMPACT", "1")
	defer os.Unsetenv("AICT_XML")
	defer os.Unsetenv("AICT_NOCOMPACT")
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

	result := runDiff(t, []string{existing, testutil.MissingPath(t, "missing.txt")})
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

// Context lines flank each change and merge hunks whose windows touch, as in
// GNU diff. Without -u or -U the hunk stays to changed lines only, which is
// what every other diff test in this package expects.
func TestDiff_Context(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "c1.txt", "a\nb\nc\nd\ne\nf\ng\nh\ni\nj\n")
	p2 := createFile(t, dir, "c2.txt", "a\nb\nc\nd\ne\nf\ng\nh\nI\nj\n")

	t.Run("no flags keeps changed lines only", func(t *testing.T) {
		result := runDiff(t, []string{p1, p2})
		if len(result.Hunks) != 1 {
			t.Fatalf("expected 1 hunk, got %d", len(result.Hunks))
		}
		h := result.Hunks[0]
		if h.OldStart != 9 || h.OldCount != 1 || h.NewStart != 9 || h.NewCount != 1 {
			t.Errorf("hunk header = -%d,%d +%d,%d; want -9,1 +9,1", h.OldStart, h.OldCount, h.NewStart, h.NewCount)
		}
		for _, l := range h.Lines {
			if l.Type == "context" {
				t.Errorf("no context was requested but %q appeared as context", l.Content)
			}
		}
	})

	t.Run("U1 gives one line of context", func(t *testing.T) {
		result := runDiff(t, []string{p1, p2, "-U", "1"})
		if len(result.Hunks) != 1 {
			t.Fatalf("expected 1 hunk, got %d", len(result.Hunks))
		}
		h := result.Hunks[0]
		if h.OldStart != 8 || h.OldCount != 3 || h.NewStart != 8 || h.NewCount != 3 {
			t.Errorf("hunk header = -%d,%d +%d,%d; want -8,3 +8,3", h.OldStart, h.OldCount, h.NewStart, h.NewCount)
		}
		var kinds []string
		for _, l := range h.Lines {
			kinds = append(kinds, l.Type)
		}
		want := []string{"context", "removed", "added", "context"}
		if len(kinds) != len(want) {
			t.Fatalf("line kinds = %v, want %v", kinds, want)
		}
		for i := range want {
			if kinds[i] != want[i] {
				t.Errorf("line kind %d = %q, want %q", i, kinds[i], want[i])
			}
		}
	})

	t.Run("U0 matches the no-context output", func(t *testing.T) {
		plain := runDiff(t, []string{p1, p2})
		explicit := runDiff(t, []string{p1, p2, "-U", "0"})
		if len(explicit.Hunks) != len(plain.Hunks) {
			t.Fatalf("-U 0 gave %d hunks, plain gave %d", len(explicit.Hunks), len(plain.Hunks))
		}
		for i := range plain.Hunks {
			a, b := plain.Hunks[i], explicit.Hunks[i]
			if a.OldStart != b.OldStart || a.OldCount != b.OldCount ||
				a.NewStart != b.NewStart || a.NewCount != b.NewCount ||
				len(a.Lines) != len(b.Lines) {
				t.Errorf("hunk %d differs: plain=%+v explicit=%+v", i, a, b)
			}
		}
	})

	t.Run("u defaults to three lines of context", func(t *testing.T) {
		result := runDiff(t, []string{p1, p2, "-u"})
		if len(result.Hunks) != 1 {
			t.Fatalf("expected 1 hunk, got %d", len(result.Hunks))
		}
		h := result.Hunks[0]
		// The change is on line 9 of a 10-line file: three lines of context
		// on each side is a window of 6..12, clamped to 6..10. GNU diff -u
		// reports the same header.
		if h.OldStart != 6 || h.OldCount != 5 || h.NewStart != 6 || h.NewCount != 5 {
			t.Errorf("hunk header = -%d,%d +%d,%d; want -6,5 +6,5", h.OldStart, h.OldCount, h.NewStart, h.NewCount)
		}
	})

	t.Run("close changes merge into one hunk", func(t *testing.T) {
		q1 := createFile(t, dir, "m1.txt", "a\nb\nc\nd\ne\n")
		q2 := createFile(t, dir, "m2.txt", "a\nB\nc\nD\ne\n")

		result := runDiff(t, []string{q1, q2, "-U", "1"})
		if len(result.Hunks) != 1 {
			t.Fatalf("two changes one line apart must merge into 1 hunk, got %d: %+v", len(result.Hunks), result.Hunks)
		}
		if result.ChangedHunks != 1 {
			t.Errorf("ChangedHunks = %d, want 1", result.ChangedHunks)
		}
	})

	t.Run("distant changes stay separate", func(t *testing.T) {
		r1 := createFile(t, dir, "s1.txt", "a\nb\nc\nd\ne\nf\ng\n")
		r2 := createFile(t, dir, "s2.txt", "A\nb\nc\nd\ne\nf\nG\n")

		result := runDiff(t, []string{r1, r2, "-U", "1"})
		if len(result.Hunks) != 2 {
			t.Fatalf("changes six lines apart must stay 2 hunks, got %d", len(result.Hunks))
		}
	})

	t.Run("counts cover only real changes", func(t *testing.T) {
		result := runDiff(t, []string{p1, p2, "-u"})
		if result.AddedLines != 1 || result.RemovedLines != 1 {
			t.Errorf("added=%d removed=%d; context must not inflate either", result.AddedLines, result.RemovedLines)
		}
	})

	t.Run("insertion at start of file", func(t *testing.T) {
		s1 := createFile(t, dir, "i1.txt", "x\ny\n")
		s2 := createFile(t, dir, "i2.txt", "new\nx\ny\n")

		result := runDiff(t, []string{s1, s2, "-U", "2"})
		if len(result.Hunks) != 1 {
			t.Fatalf("expected 1 hunk, got %d", len(result.Hunks))
		}
		h := result.Hunks[0]
		if h.OldStart != 1 || h.NewStart != 1 {
			t.Errorf("hunk starts = -%d +%d; want -1 +1", h.OldStart, h.NewStart)
		}
		if h.OldCount != 2 || h.NewCount != 3 {
			t.Errorf("hunk counts = -%d,%d; want -2,3", h.OldCount, h.NewCount)
		}
	})
}

// The plain renderer must show context lines with a leading space, the way
// GNU's unified output does; before context existed it dropped them.
func TestDiff_ContextPlain(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "p1.txt", "a\nb\nc\n")
	p2 := createFile(t, dir, "p2.txt", "a\nB\nc\n")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	Run([]string{p1, p2, "-U", "1", "--plain"})
	w.Close()
	os.Stdout = oldStdout

	var out bytes.Buffer
	out.ReadFrom(r)

	want := "@@ -1,3 +1,3 @@\n a\n-b\n+B\n c\n"
	if out.String() != want {
		t.Errorf("plain output:\ngot:  %q\nwant: %q", out.String(), want)
	}
}

// GNU also accepts the count glued onto the flag; aict parses both spellings
// rather than silently treating -U3 as a path.
func TestDiff_GluedContextFlag(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "g1.txt", "a\nb\nc\nd\ne\nf\ng\nh\ni\nj\n")
	p2 := createFile(t, dir, "g2.txt", "a\nb\nc\nd\ne\nf\ng\nh\nI\nj\n")

	glued := runDiff(t, []string{p1, p2, "-U1"})
	separate := runDiff(t, []string{p1, p2, "-U", "1"})

	if len(glued.Hunks) != 1 || len(separate.Hunks) != 1 {
		t.Fatalf("glued gave %d hunks, separate gave %d", len(glued.Hunks), len(separate.Hunks))
	}
	g, s := glued.Hunks[0], separate.Hunks[0]
	if g.OldStart != s.OldStart || g.OldCount != s.OldCount ||
		g.NewStart != s.NewStart || g.NewCount != s.NewCount {
		t.Errorf("glued -U1 (%d,%d +%d,%d) differs from -U 1 (%d,%d +%d,%d)",
			g.OldStart, g.OldCount, g.NewStart, g.NewCount,
			s.OldStart, s.OldCount, s.NewStart, s.NewCount)
	}
	if g.OldStart != 8 {
		t.Errorf("expected the window to start at line 8, got %d", g.OldStart)
	}
}
