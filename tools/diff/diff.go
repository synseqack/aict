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
}

type Config struct {
	Unified        bool
	LabelOld       string
	LabelNew       string
	Recursive      bool
	IgnoreAllSpace bool
	Quiet          bool
	Context        int
	XML            bool
	JSON           bool
	Plain          bool
	Pretty         bool
}

type DiffResult struct {
	XMLName      xml.Name    `xml:"diff"`
	OldFile      string      `xml:"old_file,attr"`
	NewFile      string      `xml:"new_file,attr"`
	OldLabel     string      `xml:"old_label,attr,omitempty"`
	NewLabel     string      `xml:"new_label,attr,omitempty"`
	AddedLines   int         `xml:"added_lines,attr"`
	RemovedLines int         `xml:"removed_lines,attr"`
	ChangedHunks int         `xml:"changed_hunks,attr"`
	Identical    bool        `xml:"identical,attr"`
	Timestamp    int64       `xml:"timestamp,attr"`
	Hunks        []DiffHunk  `xml:"hunk"`
	Errors       []DiffError `xml:"error,omitempty"`
}

func (*DiffResult) isDiffResult() {}

type DiffHunk struct {
	XMLName  xml.Name   `xml:"hunk"`
	OldStart int        `xml:"old_start,attr"`
	OldCount int        `xml:"old_count,attr"`
	NewStart int        `xml:"new_start,attr"`
	NewCount int        `xml:"new_count,attr"`
	Lines    []DiffLine `xml:"line"`
}

type DiffLine struct {
	XMLName xml.Name `xml:"line"`
	Type    string   `xml:"type,attr"`
	Number  int      `xml:"number,attr"`
	Content string   `xml:"content,attr"`
}

type DiffError struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Msg     string   `xml:"msg,attr"`
	Path    string   `xml:"path,attr"`
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
			line = regexp.MustCompile(`\s+`).ReplaceAllString(line, " ")
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
			current.Lines = append(current.Lines, DiffLine{
				Type:    "removed",
				Number:  oldIdx + 1,
				Content: oldLines[oldIdx],
			})
			oldIdx++
		} else if e.kind == inserted {
			added++
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

func computeLCS(oldLines, newLines []string) []lcsEdit {
	N, M := len(oldLines), len(newLines)

	dp := make([][]int, N+1)
	for i := range dp {
		dp[i] = make([]int, M+1)
	}

	for i := 1; i <= N; i++ {
		for j := 1; j <= M; j++ {
			if oldLines[i-1] == newLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] > dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	var edits []lcsEdit
	i, j := N, M

	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			edits = append(edits, lcsEdit{kind: equal, oldIndex: i - 1, newIndex: j - 1})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			edits = append(edits, lcsEdit{kind: inserted, oldIndex: i - 1, newIndex: j - 1})
			j--
		} else if i > 0 {
			edits = append(edits, lcsEdit{kind: deleted, oldIndex: i - 1, newIndex: j - 1})
			i--
		}
	}

	for l, r := 0, len(edits)-1; l < r; l, r = l+1, r-1 {
		edits[l], edits[r] = edits[r], edits[l]
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
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result, cfg)
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
		oldCount := countDiffLines(hunk.Lines, "removed")
		newCount := countDiffLines(hunk.Lines, "added")
		fmt.Fprintf(w, "@@ -%d,%d +%d,%d @@\n", hunk.OldStart, oldCount, hunk.NewStart, newCount)
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

func countDiffLines(lines []DiffLine, target string) int {
	count := 0
	for _, l := range lines {
		if l.Type == target {
			count++
		}
	}
	return count
}
