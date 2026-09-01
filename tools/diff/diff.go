package diff

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("diff", Run)
	tool.RegisterMeta("diff", tool.GenerateSchema("diff", "Compare two files or directories and show differences", Config{}))

	dict := map[string]string{
		"of": "old_file",
		"nf": "new_file",
		"ol": "old_label",
		"nl": "new_label",
		"al": "added_lines",
		"rl": "removed_lines",
		"ch": "changed_hunks",
		"id": "identical",
		"t":  "timestamp",
		"h":  "hunk",
		"os": "old_start",
		"oc": "old_count",
		"ns": "new_start",
		"nc": "new_count",
		"l":  "line",
		"ty": "type",
		"num":"number",
		"ct": "content",
		"e":  "error",
		"c":  "code",
		"msg":"msg",
	}
	xmlout.RegisterDict("diff", dict)
}

type Config struct {
	Unified        bool   `flag:"" desc:"Use unified diff format"`
	LabelOld       string `flag:"" desc:"Label for old file in diff"`
	LabelNew       string `flag:"" desc:"Label for new file in diff"`
	Recursive      bool   `flag:"" desc:"Compare directories recursively"`
	IgnoreAllSpace bool   `flag:"" desc:"Ignore all whitespace changes"`
	Quiet          bool   `flag:"" desc:"Output only whether files differ"`
	Context        int    `flag:"" desc:"Number of context lines"`
	XML            bool
	JSON           bool
	Plain          bool
	Pretty         bool
	NoCompact     bool
	Dict           bool
}

type DiffResult struct {
	XMLName      xml.Name    `xml:"diff" json:"-"`
	OldFile      string      `xml:"old_file,attr" json:"of"`
	NewFile      string      `xml:"new_file,attr" json:"nf"`
	OldLabel     string      `xml:"old_label,attr,omitempty" json:"ol"`
	NewLabel     string      `xml:"new_label,attr,omitempty" json:"nl"`
	AddedLines   int         `xml:"added_lines,attr" json:"al"`
	RemovedLines int         `xml:"removed_lines,attr" json:"rl"`
	ChangedHunks int         `xml:"changed_hunks,attr" json:"ch"`
	Identical    bool        `xml:"identical,attr" json:"id"`
	Timestamp    int64       `xml:"timestamp,attr" json:"t"`
	Hunks        []DiffHunk  `xml:"hunk" json:"h"`
	Errors       []DiffError `xml:"error,omitempty" json:"e"`
}

func (*DiffResult) isDiffResult() {}

var wsRe = regexp.MustCompile(`\s+`)

type DiffHunk struct {
	XMLName  xml.Name   `xml:"hunk" json:"-"`
	OldStart int        `xml:"old_start,attr" json:"os"`
	OldCount int        `xml:"old_count,attr" json:"oc"`
	NewStart int        `xml:"new_start,attr" json:"ns"`
	NewCount int        `xml:"new_count,attr" json:"nc"`
	Lines    []DiffLine `xml:"line" json:"l"`
}

type DiffLine struct {
	XMLName xml.Name `xml:"line" json:"-"`
	Type    string   `xml:"type,attr" json:"ty"`
	Number  int      `xml:"number,attr" json:"num"`
	Content string   `xml:"content,attr" json:"ct"`
}

type DiffError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"msg"`
	Path    string   `xml:"path,attr" json:"p"`
}

func Run(args []string) error {
	cfg, paths := parseFlags(args)

	if len(paths) < 2 {
		return outputResult(&DiffResult{
			Timestamp: meta.Now(),
			Errors:    []DiffError{{Code: 1, Msg: "missing file operands", Path: ""}},
		}, cfg)
	}

	oldPath := paths[0]
	newPath := paths[1]

	oldResolved, err := pathutil.Resolve(oldPath)
	if err != nil {
		return outputResult(&DiffResult{
			OldFile:   oldPath,
			NewFile:   newPath,
			Timestamp: meta.Now(),
			Errors:    []DiffError{{Code: 1, Msg: err.Error(), Path: oldPath}},
		}, cfg)
	}

	newResolved, err := pathutil.Resolve(newPath)
	if err != nil {
		return outputResult(&DiffResult{
			OldFile:   oldPath,
			NewFile:   newPath,
			Timestamp: meta.Now(),
			Errors:    []DiffError{{Code: 1, Msg: err.Error(), Path: newPath}},
		}, cfg)
	}

	oldInfo, oldErr := os.Lstat(oldResolved.Absolute)
	newInfo, newErr := os.Lstat(newResolved.Absolute)

	if oldErr != nil || newErr != nil {
		result := &DiffResult{
			OldFile:   oldPath,
			NewFile:   newPath,
			Timestamp: meta.Now(),
		}
		if oldErr != nil {
			code := 2
			if os.IsNotExist(oldErr) {
				code = 2
			}
			result.Errors = append(result.Errors, DiffError{Code: code, Msg: "no such file or directory", Path: oldResolved.Absolute})
		}
		if newErr != nil {
			code := 2
			if os.IsNotExist(newErr) {
				code = 2
			}
			result.Errors = append(result.Errors, DiffError{Code: code, Msg: "no such file or directory", Path: newResolved.Absolute})
		}
		return outputResult(result, cfg)
	}

	if oldInfo.IsDir() && newInfo.IsDir() && cfg.Recursive {
		return diffDirectories(oldResolved.Absolute, newResolved.Absolute, cfg)
	}

	if oldInfo.IsDir() || newInfo.IsDir() {
		return outputResult(&DiffResult{
			OldFile:   oldPath,
			NewFile:   newPath,
			Timestamp: meta.Now(),
			Errors:    []DiffError{{Code: 1, Msg: "different file types", Path: ""}},
		}, cfg)
	}

	return diffFiles(oldResolved.Absolute, newResolved.Absolute, oldPath, newPath, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-u", "--unified":
			cfg.Unified = true
		case "-r", "--recursive":
			cfg.Recursive = true
		case "-w", "--ignore-all-space":
			cfg.IgnoreAllSpace = true
		case "-q", "--brief":
			cfg.Quiet = true
		case "--label":
			if i+1 < len(args) {
				if cfg.LabelOld == "" {
					cfg.LabelOld = args[i+1]
				} else if cfg.LabelNew == "" {
					cfg.LabelNew = args[i+1]
				}
				i++
			}
		case "-U", "--context":
			if i+1 < len(args) {
				n, _ := strconv.Atoi(args[i+1])
				cfg.Context = n
				i++
			}
		case "--xml", "-xml":
			cfg.XML = true
		case "--json", "-json":
			cfg.JSON = true
		case "--plain", "-plain":
			cfg.Plain = true
		case "--pretty", "-pretty":
			cfg.Pretty = true
		case "--no-compact":
			cfg.NoCompact = true
		case "--dict":
			cfg.Dict = true
		default:
			if !strings.HasPrefix(arg, "-") {
				positional = append(positional, arg)
			}
		}
	}

	if !cfg.XML && !cfg.JSON && !cfg.Plain {
		cfg.XML = xmlout.IsXMLMode()
	}

	if cfg.LabelOld == "" {
		cfg.LabelOld = "a"
	}
	if cfg.LabelNew == "" {
		cfg.LabelNew = "b"
	}

	return cfg, positional
}

func diffFiles(oldPath, newPath, oldName, newName string, cfg Config) error {
	oldLines, err := readLines(oldPath, cfg.IgnoreAllSpace)
	if err != nil {
		return outputResult(&DiffResult{
			OldFile:   oldName,
			NewFile:   newName,
			OldLabel:  cfg.LabelOld,
			NewLabel:  cfg.LabelNew,
			Timestamp: meta.Now(),
			Errors:    []DiffError{{Code: 1, Msg: err.Error(), Path: oldPath}},
		}, cfg)
	}

	newLines, err := readLines(newPath, cfg.IgnoreAllSpace)
	if err != nil {
		return outputResult(&DiffResult{
			OldFile:   oldName,
			NewFile:   newName,
			OldLabel:  cfg.LabelOld,
			NewLabel:  cfg.LabelNew,
			Timestamp: meta.Now(),
			Errors:    []DiffError{{Code: 1, Msg: err.Error(), Path: newPath}},
		}, cfg)
	}

	result := computeDiff(oldLines, newLines, oldName, newName, cfg)
	return outputResult(result, cfg)
}

func readLines(path string, ignoreSpace bool) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if ignoreSpace {
			line = wsRe.ReplaceAllString(line, " ")
		}
		lines = append(lines, line)
	}
	return lines, scanner.Err()
}

func computeDiff(oldLines, newLines []string, oldName, newName string, cfg Config) *DiffResult {
	result := &DiffResult{
		OldFile:   oldName,
		NewFile:   newName,
		OldLabel:  cfg.LabelOld,
		NewLabel:  cfg.LabelNew,
		Timestamp: meta.Now(),
	}

	if slicesEqual(oldLines, newLines) {
		result.Identical = true
		return result
	}

	N, M := len(oldLines), len(newLines)
	if N == 0 && M == 0 {
		result.Identical = true
		return result
	}

	edits := computeLCS(oldLines, newLines)

	var hunks []DiffHunk
	var current *DiffHunk
	added := 0
	removed := 0

	oldIdx := 0
	newIdx := 0

	for _, e := range edits {
		if e.kind == equal {
			if current != nil {
				hunks = append(hunks, *current)
				current = nil
			}
			oldIdx++
			newIdx++
			continue
		}

		if current == nil {
			current = &DiffHunk{
				OldStart: oldIdx + 1,
				NewStart: newIdx + 1,
			}
		}

		if e.kind == deleted {
			removed++
			current.OldCount++
			current.Lines = append(current.Lines, DiffLine{
				Type:    "removed",
				Number:  oldIdx + 1,
				Content: oldLines[oldIdx],
			})
			oldIdx++
		} else if e.kind == inserted {
			added++
			current.NewCount++
			current.Lines = append(current.Lines, DiffLine{
				Type:    "added",
				Number:  newIdx + 1,
				Content: newLines[newIdx],
			})
			newIdx++
		}
	}

	if current != nil {
		hunks = append(hunks, *current)
	}

	result.Hunks = hunks
	result.AddedLines = added
	result.RemovedLines = removed
	result.ChangedHunks = len(hunks)
	result.Identical = added == 0 && removed == 0

	return result
}

type editKind int

const (
	deleted editKind = iota
	inserted
	equal
)

type lcsEdit struct {
	kind     editKind
	oldIndex int
	newIndex int
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// computeLCS implements Myers O(ND) diff algorithm. It returns the same
// []lcsEdit sequence (deleted/inserted/equal) as before; callers are unchanged.
func computeLCS(a, b []string) []lcsEdit {
	N, M := len(a), len(b)
	if N == 0 && M == 0 {
		return nil
	}

	max := N + M
	// v maps diagonal k (offset by max) to the furthest x reached on that diagonal.
	v := make([]int, 2*max+1)
	// trace stores v snapshots per edit distance d for backtracking.
	trace := make([][]int, 0, max+1)

	found := false
	var endD int

outer:
	for d := 0; d <= max; d++ {
		snap := make([]int, len(v))
		copy(snap, v)
		trace = append(trace, snap)

		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[k-1+max] < v[k+1+max]) {
				x = v[k+1+max]
			} else {
				x = v[k-1+max] + 1
			}
			y := x - k
			for x < N && y < M && a[x] == b[y] {
				x++
				y++
			}
			v[k+max] = x
			if x >= N && y >= M {
				endD = d
				found = true
				break outer
			}
		}
	}

	if !found {
		// Fallback: all deleted then all inserted.
		var edits []lcsEdit
		for i := range a {
			edits = append(edits, lcsEdit{kind: deleted, oldIndex: i})
		}
		for j := range b {
			edits = append(edits, lcsEdit{kind: inserted, newIndex: j})
		}
		return edits
	}

	// Backtrack from (N, M) to (0, 0). trace[d] holds the v-state from
	// before round d, i.e. the frontier the round-d move originated from.
	var rev []lcsEdit
	x, y := N, M
	for d := endD; d >= 0 && (x > 0 || y > 0); d-- {
		V := trace[d]
		k := x - y
		var pk int
		if k == -d || (k != d && V[k-1+max] < V[k+1+max]) {
			pk = k + 1
		} else {
			pk = k - 1
		}
		px := V[pk+max]
		py := px - pk

		// Walk the snake (equal run) backwards to the move endpoint.
		for x > px && y > py {
			rev = append(rev, lcsEdit{kind: equal, oldIndex: x - 1, newIndex: y - 1})
			x--
			y--
		}
		if d > 0 {
			if x == px {
				// Downward move: b[y-1] was inserted.
				rev = append(rev, lcsEdit{kind: inserted, oldIndex: x, newIndex: y - 1})
				y--
			} else {
				// Rightward move: a[x-1] was deleted.
				rev = append(rev, lcsEdit{kind: deleted, oldIndex: x - 1, newIndex: y})
				x--
			}
		}
	}

	edits := make([]lcsEdit, len(rev))
	for i, e := range rev {
		edits[len(rev)-1-i] = e
	}
	return edits
}

func diffDirectories(oldDir, newDir string, cfg Config) error {
	result := &DiffResult{
		OldFile:   oldDir,
		NewFile:   newDir,
		OldLabel:  cfg.LabelOld,
		NewLabel:  cfg.LabelNew,
		Timestamp: meta.Now(),
	}

	files := make(map[string]bool)

	filepath.WalkDir(oldDir, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(oldDir, path)
		files[rel] = true
		return nil
	})

	filepath.WalkDir(newDir, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(newDir, path)
		files[rel] = true
		return nil
	})

	for rel := range files {
		oldPath := filepath.Join(oldDir, rel)
		newPath := filepath.Join(newDir, rel)

		oldInfo, _ := os.Lstat(oldPath)
		newInfo, _ := os.Lstat(newPath)

		if oldInfo == nil {
			result.AddedLines++
			result.Hunks = append(result.Hunks, DiffHunk{
				NewStart: 1,
				NewCount: 1,
				Lines:    []DiffLine{{Type: "added", Number: 1, Content: "new file: " + rel}},
			})
		} else if newInfo == nil {
			result.RemovedLines++
			result.Hunks = append(result.Hunks, DiffHunk{
				OldStart: 1,
				OldCount: 1,
				Lines:    []DiffLine{{Type: "removed", Number: 1, Content: "deleted: " + rel}},
			})
		}
	}

	result.ChangedHunks = len(result.Hunks)
	result.Identical = result.AddedLines == 0 && result.RemovedLines == 0

	return outputResult(result, cfg)
}

func outputResult(result *DiffResult, cfg Config) error {
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("diff")
		if dict != nil {
			var keys []string
			for short := range dict {
				keys = append(keys, short)
			}
			for i := 0; i < len(keys); i++ {
				for j := i + 1; j < len(keys); j++ {
					if keys[i] > keys[j] {
						keys[i], keys[j] = keys[j], keys[i]
					}
				}
			}
			fmt.Print("<dict>")
			for _, short := range keys {
				fmt.Printf("<%s>%s</%s>", short, dict[short], short)
			}
			fmt.Println("</dict>")
		}
		return nil
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result, cfg)
	}
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *DiffResult, cfg Config) error {
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "diff: %s: %s\n", e.Path, e.Msg)
		}
		return nil
	}

	if cfg.Quiet && result.Identical {
		return nil
	}

	if cfg.Quiet {
		fmt.Fprintf(w, "Files %s and %s differ\n", result.OldFile, result.NewFile)
		return nil
	}

	if result.Identical {
		fmt.Fprintf(w, "No differences found\n")
		return nil
	}

	for _, hunk := range result.Hunks {
		fmt.Fprintf(w, "@@ -%d,%d +%d,%d @@\n", hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount)
		for _, line := range hunk.Lines {
			switch line.Type {
			case "removed":
				fmt.Fprintf(w, "-%s\n", line.Content)
			case "added":
				fmt.Fprintf(w, "+%s\n", line.Content)
			}
		}
	}

	return nil
}
