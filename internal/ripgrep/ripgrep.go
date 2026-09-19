// Package ripgrep locates and drives an external ripgrep binary.
//
// It builds an argument list from a search request, spawns rg with --json,
// and decodes the JSON Lines stream into plain Go values. Callers keep
// responsibility for which files are searched and for their own result
// schema; this package only translates rg's wire format.
package ripgrep

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ErrNotFound is returned when no usable ripgrep binary is on PATH.
var ErrNotFound = errors.New("ripgrep not found")

// maxArgvBytes bounds the size of one rg invocation. ARG_MAX varies by
// platform (macOS is the tightest in practice) and the budget has to cover
// the environment as well, so this stays well below every common limit.
const maxArgvBytes = 128 * 1024

// probeTimeout bounds the availability probe. A hung rg is a reason to fall
// back, not a reason to hang grep.
const probeTimeout = 5 * time.Second

// Args describes a search request that ripgrep can carry out.
type Args struct {
	Pattern         string
	CaseInsensitive bool
	WordMatch       bool
	FixedStrings    bool
	InvertMatch     bool
	MaxCount        int
}

// Match is one matching line.
type Match struct {
	LineNumber     int
	Text           string
	AbsoluteOffset int64
}

// FileMatches collects the matches found in one file.
type FileMatches struct {
	Path    string
	Matches []Match
}

// Result is the merged output of one or more ripgrep invocations.
type Result struct {
	Files []FileMatches
}

// Available reports whether a ripgrep binary is on PATH and actually runs.
// AICT_NORG=1 disables the integration so that grep always uses the
// built-in engine. The probe is bounded: a broken rg on PATH must degrade
// grep to the built-in engine, not hang it.
func Available() bool {
	path, err := Path()
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	return err == nil && bytes.Contains(out, []byte("ripgrep"))
}

// Path returns the resolved rg binary, or ErrNotFound.
func Path() (string, error) {
	if os.Getenv("AICT_NORG") == "1" {
		return "", ErrNotFound
	}
	path, err := exec.LookPath("rg")
	if err != nil {
		return "", ErrNotFound
	}
	return path, nil
}

// Search runs rg over files and returns the decoded matches. Files are
// searched exactly as given; ripgrep's own ignore and hidden-file rules are
// turned off so the set searched is the set the caller passed in. When files
// is empty no process is spawned.
func Search(ctx context.Context, a Args, files []string) (*Result, error) {
	if len(files) == 0 {
		return &Result{}, nil
	}

	bin, err := Path()
	if err != nil {
		return nil, err
	}

	result := &Result{}
	for _, chunk := range chunkFiles(files) {
		if err := searchChunk(ctx, bin, a, chunk, result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func searchChunk(ctx context.Context, bin string, a Args, files []string, result *Result) error {
	cmd := exec.CommandContext(ctx, bin, a.argv(files)...)

	// rg diagnostics are captured rather than surfaced: aict reports failure
	// as a structured error, never by writing to stderr.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	// rg emits one JSON object per line. ReadString grows past its initial
	// buffer for arbitrarily long lines, so a minified 10 MB source file
	// does not truncate the stream.
	reader := bufio.NewReaderSize(stdout, 128*1024)
	dec := newDecoder(result)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimRight(line, "\r\n")
			if len(line) > 0 {
				if derr := dec.add([]byte(line)); derr != nil {
					stdout.Close()
					cmd.Wait()
					return derr
				}
			}
		}
		if err != nil {
			break
		}
	}
	stdout.Close()

	if err := cmd.Wait(); err != nil {
		// rg exits 1 when a search found nothing, which is a normal result
		// here rather than a failure. Anything else is a real error.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil
		}
		if stderr.Len() > 0 {
			return errors.New(string(bytes.TrimSpace(stderr.Bytes())))
		}
		return err
	}
	return nil
}

func (a Args) argv(files []string) []string {
	args := []string{"--json", "--no-ignore", "--hidden"}
	if a.CaseInsensitive {
		args = append(args, "-i")
	}
	if a.WordMatch {
		args = append(args, "-w")
	}
	if a.FixedStrings {
		args = append(args, "-F")
	}
	if a.InvertMatch {
		args = append(args, "-v")
	}
	if a.MaxCount > 0 {
		args = append(args, "-m", strconv.Itoa(a.MaxCount))
	}
	args = append(args, "--", a.Pattern)
	args = append(args, files...)
	return args
}

func chunkFiles(files []string) [][]string {
	var chunks [][]string
	var cur []string
	size := 0
	for _, f := range files {
		cost := len(f) + 1
		if size+cost > maxArgvBytes && len(cur) > 0 {
			chunks = append(chunks, cur)
			cur, size = nil, 0
		}
		cur = append(cur, f)
		size += cost
	}
	if len(cur) > 0 {
		chunks = append(chunks, cur)
	}
	return chunks
}

// decoder turns rg's JSON Lines stream into Result files. rg emits begin,
// match, context and end messages (plus summary when --stats is given);
// every match carries its own path, so grouping needs no other message type.
type decoder struct {
	result *Result
	index  map[string]int
}

func newDecoder(result *Result) *decoder {
	return &decoder{result: result, index: make(map[string]int)}
}

func (d *decoder) add(line []byte) error {
	var msg struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(line, &msg); err != nil {
		return err
	}
	if msg.Type != "match" {
		return nil
	}

	var data struct {
		Path           *arbitrary `json:"path"`
		Lines          *arbitrary `json:"lines"`
		LineNumber     *int       `json:"line_number"`
		AbsoluteOffset int64      `json:"absolute_offset"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return err
	}
	if data.Path == nil || data.Lines == nil {
		return nil
	}

	path := data.Path.value()
	idx, ok := d.index[path]
	if !ok {
		idx = len(d.result.Files)
		d.result.Files = append(d.result.Files, FileMatches{Path: path})
		d.index[path] = idx
	}

	// rg terminates lines.text with the line separator; aict's Text does not
	// carry it, so strip exactly the one "\n" the built-in engine strips.
	text := strings.TrimSuffix(data.Lines.value(), "\n")

	m := Match{Text: text, AbsoluteOffset: data.AbsoluteOffset}
	if data.LineNumber != nil {
		m.LineNumber = *data.LineNumber
	}
	d.result.Files[idx].Matches = append(d.result.Files[idx].Matches, m)
	return nil
}

// arbitrary is rg's "arbitrary data" object: valid UTF-8 arrives as text,
// anything else as base64 bytes.
type arbitrary struct {
	Text  string `json:"text"`
	Bytes string `json:"bytes"`
}

func (a *arbitrary) value() string {
	if a == nil {
		return ""
	}
	if a.Bytes != "" {
		if raw, err := base64.StdEncoding.DecodeString(a.Bytes); err == nil {
			return string(raw)
		}
	}
	return a.Text
}
