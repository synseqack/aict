// Command stubrg is a stand-in for ripgrep used by the grep equivalence
// test. It implements only the flag subset aict translates and emits the
// JSON Lines format documented for rg --json, using Go's own regular
// expressions to decide what matches. That keeps the test honest about what
// it proves: the built-in engine and the ripgrep path see the same matches,
// and the conversion layer reproduces aict's schema from rg's wire format.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	var (
		pattern                 string
		files                   []string
		caseInsensitive         bool
		wordMatch, fixedStrings bool
		invert                  bool
		maxCount                int
	)

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--version":
			// aict probes for this before trusting the binary.
			fmt.Println("ripgrep 14.1.1 (stub)")
			os.Exit(0)
		case "--json", "--no-ignore", "--hidden":
		case "-i":
			caseInsensitive = true
		case "-w":
			wordMatch = true
		case "-F":
			fixedStrings = true
		case "-v":
			invert = true
		case "-m":
			i++
			if i < len(args) {
				maxCount, _ = strconv.Atoi(args[i])
			}
		case "--":
			i++
			if i < len(args) {
				pattern = args[i]
				files = args[i+1:]
			}
			i = len(args)
		default:
			fmt.Fprintf(os.Stderr, "stubrg: unrecognized flag %q\n", a)
			os.Exit(2)
		}
	}

	re, err := compile(pattern, caseInsensitive, wordMatch, fixedStrings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stubrg: %v\n", err)
		os.Exit(2)
	}

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	matchedAny := false
	for _, file := range files {
		lines, err := readLines(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "stubrg: %v\n", err)
			os.Exit(2)
		}

		write(out, "begin", map[string]any{"path": arbitrary(file)})

		var offset int64
		count := 0
		for ln, text := range lines {
			if re.MatchString(text) != invert {
				write(out, "match", matchData{
					Path:           arbitrary(file),
					Lines:          arbitrary(text + "\n"),
					LineNumber:     ln + 1,
					AbsoluteOffset: offset,
					Submatches:     submatches(re, text, invert),
				})
				matchedAny = true
				count++
				if maxCount > 0 && count >= maxCount {
					break
				}
			}
			offset += int64(len(text) + 1)
		}

		fabricate(out, file)

		write(out, "end", map[string]any{
			"path":          arbitrary(file),
			"binary_offset": nil,
			"stats":         map[string]any{},
		})
	}

	if !matchedAny {
		os.Exit(1)
	}
}

// fabricate emits a match that does not exist in the file. Tests set
// STUB_LIE=1 to prove the ripgrep backend is really running: if aict had
// silently fallen back to the built-in engine, the fabricated match would
// never appear.
func fabricate(out *bufio.Writer, file string) {
	if os.Getenv("STUB_LIE") != "1" {
		return
	}
	write(out, "match", matchData{
		Path:           arbitrary(file),
		Lines:          arbitrary("fabricated by stub\n"),
		LineNumber:     999,
		AbsoluteOffset: 0,
		Submatches:     []map[string]any{{"match": map[string]string{"text": "fabricated"}, "start": 0, "end": 10}},
	})
}

func compile(pattern string, caseInsensitive, wordMatch, fixedStrings bool) (*regexp.Regexp, error) {
	if fixedStrings {
		pattern = regexp.QuoteMeta(pattern)
	}
	if wordMatch {
		pattern = `\b` + pattern + `\b`
	}
	if caseInsensitive {
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

// readLines mirrors aict's own reader: a line keeps a trailing \r and a
// final line without a terminator is still searched intact.
func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	reader := bufio.NewReaderSize(f, 128*1024)
	for {
		line, err := reader.ReadString('\n')
		if len(line) == 0 {
			break
		}
		lines = append(lines, strings.TrimSuffix(line, "\n"))
		if err != nil {
			break
		}
	}
	return lines, nil
}

type arbitrary string

func (a arbitrary) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"text": string(a)})
}

type matchData struct {
	Path           arbitrary        `json:"path"`
	Lines          arbitrary        `json:"lines"`
	LineNumber     int              `json:"line_number"`
	AbsoluteOffset int64            `json:"absolute_offset"`
	Submatches     []map[string]any `json:"submatches"`
}

func submatches(re *regexp.Regexp, text string, invert bool) []map[string]any {
	if invert {
		return []map[string]any{}
	}
	idx := re.FindStringSubmatchIndex(text)
	if idx == nil {
		return []map[string]any{}
	}
	return []map[string]any{{
		"match": map[string]string{"text": text[idx[0]:idx[1]]},
		"start": idx[0],
		"end":   idx[1],
	}}
}

func write(out *bufio.Writer, typ string, data any) {
	b, err := json.Marshal(map[string]any{"type": typ, "data": data})
	if err != nil {
		return
	}
	out.Write(b)
	out.WriteByte('\n')
}
