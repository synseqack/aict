package grep

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/ripgrep"
)

// stubSource is the fake ripgrep the equivalence test installs on PATH.
const stubSource = "testdata/stubrg"

// buildStub compiles the fake ripgrep into a temp dir as "rg" and puts that
// directory first on PATH so ripgrep.Available finds it.
func buildStub(t *testing.T) string {
	t.Helper()

	// An explicit -o name keeps exactly that name, so on Windows the .exe
	// has to be spelled out: exec.LookPath will not resolve a bare "rg"
	// against a file with no extension.
	exe := "rg"
	if runtime.GOOS == "windows" {
		exe = "rg.exe"
	}
	bin := filepath.Join(t.TempDir(), exe)
	out, err := exec.Command("go", "build", "-o", bin, "./"+stubSource).CombinedOutput()
	if err != nil {
		t.Fatalf("build stub rg: %v\n%s", err, out)
	}

	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	return bin
}

// writeTree creates a fixture tree with the mix of files the flag matrix
// needs: nested matches, a word-boundary trap, binary content, a hidden
// file, and a file that shares an extension with an excluded directory.
func writeTree(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	files := map[string]string{
		"one.txt":             "alpha bravo\ncharlie delta\nbravo again\nfinal line",
		"two.go":              "package main\n\nfunc bravo() {}\nvar bravocado = 1\n",
		"nested/three.txt":    "bravo\nalpha\nbravo\n",
		"nested/deep/four.md": "# bravo header\n\nsome text\n",
		"binary.bin":          "\x00\x01\x02bravo\x00\xff",
		".hidden.txt":         "bravo hidden\n",
		"plain":               "no matches here at all\n",
	}
	for name, content := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", full, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
	return root
}

// searchBoth runs the same query through the ripgrep backend and through
// the built-in engine, returning both results.
func searchBoth(t *testing.T, root string, cfg Config) (*GrepResult, *GrepResult) {
	t.Helper()

	resolved, err := pathutil.Resolve(root)
	if err != nil {
		t.Fatalf("resolve %s: %v", root, err)
	}
	info, err := os.Lstat(resolved.Absolute)
	if err != nil {
		t.Fatalf("lstat %s: %v", resolved.Absolute, err)
	}

	t.Setenv("AICT_NORG", "")
	rg := searchPath(resolved.Absolute, resolved.Given, info, cfg)

	t.Setenv("AICT_NORG", "1")
	builtIn := searchPath(resolved.Absolute, resolved.Given, info, cfg)

	return rg, builtIn
}

// normalize drops the timestamp and sorts files so the two backends can be
// compared despite the built-in engine's nondeterministic worker order.
func normalize(r *GrepResult) *GrepResult {
	r.Timestamp = 0
	sort.Slice(r.Matches, func(i, j int) bool {
		return r.Matches[i].Path < r.Matches[j].Path
	})
	return r
}

func TestRipgrep_BackendActuallyRuns(t *testing.T) {
	// A fake rg that is on PATH and identifies as ripgrep must be selected.
	buildStub(t)
	if !ripgrep.Available() {
		t.Fatal("Available() = false with a valid stub on PATH")
	}
	root := writeTree(t)
	cfg := Config{Pattern: "bravo", Recursive: true}

	if !useRipgrep(cfg) {
		t.Fatal("useRipgrep = false, the backend would never be reached")
	}

	t.Setenv("STUB_LIE", "1")
	rg, builtIn := searchBoth(t, root, cfg)
	t.Setenv("STUB_LIE", "")

	// If the rg result carried the fabricated match, the backend really ran
	// rather than falling back.
	var sawLie bool
	for _, f := range rg.Matches {
		for _, l := range f.Lines {
			if l.Number == 999 {
				sawLie = true
			}
		}
	}
	if !sawLie {
		t.Error("fabricated match missing: the ripgrep backend did not run the stub")
	}
	for _, f := range builtIn.Matches {
		for _, l := range f.Lines {
			if l.Number == 999 {
				t.Error("built-in engine reported the fabricated match")
			}
		}
	}
}

func TestRipgrep_EquivalentToBuiltin(t *testing.T) {
	buildStub(t)
	root := writeTree(t)

	cases := []struct {
		name string
		cfg  Config
	}{
		{"plain recursive", Config{Pattern: "bravo", Recursive: true}},
		{"no matches", Config{Pattern: "does-not-exist", Recursive: true}},
		{"case insensitive", Config{Pattern: "BRAVO", Recursive: true, CaseInsensitive: true}},
		{"case sensitive", Config{Pattern: "BRAVO", Recursive: true}},
		{"word match", Config{Pattern: "bravo", Recursive: true, WordMatch: true}},
		{"fixed strings", Config{Pattern: "func bravo() {}", Recursive: true, FixedStrings: true}},
		{"regex metachars", Config{Pattern: `func\s+bravo`, Recursive: true}},
		{"invert match", Config{Pattern: "bravo", Recursive: true, InvertMatch: true}},
		{"max count", Config{Pattern: "bravo", Recursive: true, MaxCount: 1}},
		{"files with matches", Config{Pattern: "bravo", Recursive: true, FilesWithMatches: true}},
		{"count only", Config{Pattern: "bravo", Recursive: true, CountOnly: true}},
		{"include txt", Config{Pattern: "bravo", Recursive: true, Include: "*.txt"}},
		{"include go", Config{Pattern: "bravo", Recursive: true, Include: "*.go"}},
		{"exclude dir", Config{Pattern: "bravo", Recursive: true, ExcludeDir: "nested"}},
		{"exclude dir deep", Config{Pattern: "bravo", Recursive: true, ExcludeDir: "deep"}},
		{"include matches nothing", Config{Pattern: "bravo", Recursive: true, Include: "*.nomatch"}},
		{"non recursive dir", Config{Pattern: "bravo"}},
		{"line numbers flag", Config{Pattern: "bravo", Recursive: true, LineNumbers: true}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rg, builtIn := searchBoth(t, root, tc.cfg)
			if !reflect.DeepEqual(normalize(rg), normalize(builtIn)) {
				t.Errorf("backends disagree on %s\nripgrep:  %s\nbuiltin: %s",
					tc.cfg.patternSummary(), dump(rg), dump(builtIn))
			}
		})
	}
}

// TestMaxCount_StopsAtN pins the documented meaning of -m: "Stop after N
// matches". Both backends must report exactly N, capped by what the file
// actually contains (nested/three.txt has two matching lines).
func TestMaxCount_StopsAtN(t *testing.T) {
	root := writeTree(t)
	buildStub(t)

	for _, n := range []int{1, 2} {
		t.Run("m="+strconv.Itoa(n), func(t *testing.T) {
			rg, builtIn := searchBoth(t, filepath.Join(root, "nested", "three.txt"),
				Config{Pattern: "bravo", MaxCount: n})
			for _, r := range []*GrepResult{rg, builtIn} {
				if r.TotalMatches != n {
					t.Errorf("TotalMatches = %d, want %d", r.TotalMatches, n)
				}
			}
		})
	}
}

func TestRipgrep_EquivalentOnSingleFile(t *testing.T) {
	buildStub(t)
	root := writeTree(t)

	cases := []struct {
		name string
		file string
		cfg  Config
	}{
		{"matching file", "one.txt", Config{Pattern: "bravo"}},
		{"no matches", "plain", Config{Pattern: "bravo"}},
		{"binary file", "binary.bin", Config{Pattern: "bravo"}},
		{"go file word match", "two.go", Config{Pattern: "bravo", WordMatch: true}},
		{"include mismatch", "two.go", Config{Pattern: "bravo", Include: "*.txt"}},
		{"invert", "one.txt", Config{Pattern: "bravo", InvertMatch: true}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := filepath.Join(root, filepath.FromSlash(tc.file))
			rg, builtIn := searchBoth(t, target, tc.cfg)
			if !reflect.DeepEqual(normalize(rg), normalize(builtIn)) {
				t.Errorf("backends disagree on %s\nripgrep:  %s\nbuiltin: %s",
					tc.name, dump(rg), dump(builtIn))
			}
		})
	}
}

// TestRipgrep_FallsBackWhenGone covers the failure modes that must drop back
// to the built-in engine instead of producing an error or partial output.
func TestRipgrep_FallsBackWhenGone(t *testing.T) {
	root := writeTree(t)
	cfg := Config{Pattern: "bravo", Recursive: true}

	t.Run("context flags", func(t *testing.T) {
		buildStub(t)
		if useRipgrep(Config{Pattern: "bravo", Recursive: true, ContextLines: 2}) {
			t.Error("context search must use the built-in engine")
		}
	})

	t.Run("context flags equivalent", func(t *testing.T) {
		// Even though the rg backend declines, output must still be correct.
		buildStub(t)
		t.Setenv("AICT_NORG", "")
		rg := searchWithRipgrepOrBuiltin(t, root, Config{Pattern: "bravo", Recursive: true, BeforeContext: 1})
		t.Setenv("AICT_NORG", "1")
		want := searchWithRipgrepOrBuiltin(t, root, Config{Pattern: "bravo", Recursive: true, BeforeContext: 1})
		if !reflect.DeepEqual(normalize(rg), normalize(want)) {
			t.Errorf("context search differs\nripgrep:  %s\nbuiltin: %s", dump(rg), dump(want))
		}
	})

	t.Run("no rg on path", func(t *testing.T) {
		t.Setenv("AICT_NORG", "1")
		got := searchWithRipgrepOrBuiltin(t, root, cfg)
		if got.SearchedFiles == 0 {
			t.Error("expected the built-in engine to still search the tree")
		}
	})
}

func searchWithRipgrepOrBuiltin(t *testing.T, root string, cfg Config) *GrepResult {
	t.Helper()
	resolved, err := pathutil.Resolve(root)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	info, err := os.Lstat(resolved.Absolute)
	if err != nil {
		t.Fatalf("lstat: %v", err)
	}
	return searchPath(resolved.Absolute, resolved.Given, info, cfg)
}

// dump renders a result for test failure messages.
func dump(r *GrepResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n  searched=%d matched=%d total=%d", r.SearchedFiles, r.MatchedFiles, r.TotalMatches)
	for _, f := range r.Matches {
		fmt.Fprintf(&b, "\n  %s (%s) mif=%d", f.Path, f.Language, f.MatchesInFile)
		for _, l := range f.Lines {
			fmt.Fprintf(&b, "\n    %d:%q @%d", l.Number, l.Text, l.OffsetBytes)
			if l.Before != "" {
				fmt.Fprintf(&b, " before=%q", l.Before)
			}
			if l.After != "" {
				fmt.Fprintf(&b, " after=%q", l.After)
			}
		}
	}
	for _, e := range r.Errors {
		fmt.Fprintf(&b, "\n  ERROR %s: %s", e.Path, e.Msg)
	}
	return b.String()
}

func (c Config) patternSummary() string {
	var flags []string
	if c.Recursive {
		flags = append(flags, "-r")
	}
	if c.CaseInsensitive {
		flags = append(flags, "-i")
	}
	if c.WordMatch {
		flags = append(flags, "-w")
	}
	if c.FixedStrings {
		flags = append(flags, "-F")
	}
	if c.InvertMatch {
		flags = append(flags, "-v")
	}
	if c.MaxCount > 0 {
		flags = append(flags, "-m")
	}
	if c.FilesWithMatches {
		flags = append(flags, "-l")
	}
	if c.Include != "" {
		flags = append(flags, "--include="+c.Include)
	}
	if c.ExcludeDir != "" {
		flags = append(flags, "--exclude-dir="+c.ExcludeDir)
	}
	return c.Pattern + " [" + strings.Join(flags, " ") + "]"
}
