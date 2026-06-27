package sort

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/synseqack/aict/internal/meta"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("sort", Run)
	tool.RegisterMeta("sort", tool.GenerateSchema("sort", "Sort lines of text files", Config{}))
}

type Config struct {
	Numeric    bool   `flag:"" desc:"Sort numerically"`
	Reverse    bool   `flag:"" desc:"Sort in reverse order"`
	Key        int    `flag:"" desc:"Sort by field number (1-based)"`
	Delimiter  string `flag:"" desc:"Field delimiter (default: tab)"`
	OutputFile string `flag:"" desc:"Write output to file"`
	Unique     bool   `flag:"" desc:"Remove duplicate lines"`
	XML        bool
	JSON       bool
	Plain      bool
	Pretty     bool
	Compact bool
}

type SortResult struct {
	XMLName   xml.Name    `xml:"sort"`
	Timestamp int64       `xml:"timestamp,attr"`
	LinesIn   int         `xml:"lines_in,attr"`
	LinesOut  int         `xml:"lines_out,attr"`
	KeyField  int         `xml:"key,attr"`
	Order     string      `xml:"order,attr"`
	Content   string      `xml:"content,omitempty"`
	Errors    []SortError `xml:"error,omitempty"`
}

func (*SortResult) isSortResult() {}

type SortError struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Msg     string   `xml:"msg,attr"`
}

func Run(args []string) error {
	cfg, paths := parseFlags(args)

	if len(paths) == 0 {
		return sortStdin(cfg)
	}

	result := &SortResult{
		Timestamp: meta.Now(),
		KeyField:  cfg.Key,
		Order:     "ascending",
	}

	if cfg.Reverse {
		result.Order = "descending"
	}

	var allLines []string

	for _, path := range paths {
		lines, err := readLines(path)
		if err != nil {
			result.Errors = append(result.Errors, SortError{Code: 1, Msg: err.Error()})
			continue
		}
		allLines = append(allLines, lines...)
	}

	result.LinesIn = len(allLines)

	sortLines(&allLines, cfg)

	if cfg.Unique {
		uniqueLines := make([]string, 0, len(allLines))
		var prev string
		for _, line := range allLines {
			if line != prev {
				uniqueLines = append(uniqueLines, line)
				prev = line
			}
		}
		allLines = uniqueLines
	}

	result.LinesOut = len(allLines)
	result.Content = strings.Join(allLines, "\n")
	if len(allLines) > 0 {
		result.Content += "\n"
	}

	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config
	cfg.Key = 1
	cfg.Delimiter = "\t"

	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-n", "--numeric-sort":
			cfg.Numeric = true
		case "-r", "--reverse":
			cfg.Reverse = true
		case "-k", "--key":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					cfg.Key = n
					i++
				}
			}
		case "-t", "--field-separator":
			if i+1 < len(args) {
				cfg.Delimiter = args[i+1]
				i++
			}
		case "-u", "--unique":
			cfg.Unique = true
		case "-o", "--output":
			if i+1 < len(args) {
				cfg.OutputFile = args[i+1]
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
		case "--compact", "-compact":
			cfg.Compact = true
		default:
			positional = append(positional, arg)
		}
	}

	if !cfg.XML && !cfg.JSON && !cfg.Plain {
		cfg.XML = xmlout.IsXMLMode()
	}

	return cfg, positional
}

func readLines(path string) ([]string, error) {
	if path == "-" || path == "/dev/stdin" {
		return readStdin()
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

func readStdin() ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func sortLines(lines *[]string, cfg Config) {
	sorter := &lineSorter{
		lines:     *lines,
		numeric:   cfg.Numeric,
		reverse:   cfg.Reverse,
		key:       cfg.Key,
		delimiter: cfg.Delimiter,
	}
	sort.Sort(sorter)
	*lines = sorter.lines
}

type lineSorter struct {
	lines     []string
	numeric   bool
	reverse   bool
	key       int
	delimiter string
}

func (s *lineSorter) Len() int {
	return len(s.lines)
}

func (s *lineSorter) Swap(i, j int) {
	s.lines[i], s.lines[j] = s.lines[j], s.lines[i]
}

func (s *lineSorter) Less(i, j int) bool {
	a := s.lines[i]
	b := s.lines[j]

	if s.key > 1 {
		a = extractField(a, s.key, s.delimiter)
		b = extractField(b, s.key, s.delimiter)
	}

	if s.numeric {
		ai, _ := strconv.ParseFloat(a, 64)
		bi, _ := strconv.ParseFloat(b, 64)
		if s.reverse {
			return ai > bi
		}
		return ai < bi
	}

	if s.reverse {
		return a > b
	}
	return a < b
}

func extractField(line string, field int, delimiter string) string {
	fields := strings.Split(line, delimiter)
	if field <= 0 || field > len(fields) {
		return line
	}
	return fields[field-1]
}

func sortStdin(cfg Config) error {
	lines, err := readStdin()
	if err != nil {
		return err
	}

	result := &SortResult{
		Timestamp: meta.Now(),
		LinesIn:   len(lines),
		KeyField:  cfg.Key,
		Order:     "ascending",
	}

	if cfg.Reverse {
		result.Order = "descending"
	}

	sortLines(&lines, cfg)
	result.LinesOut = len(lines)
	result.Content = strings.Join(lines, "\n")
	if len(lines) > 0 {
		result.Content += "\n"
	}

	return outputResult(result, cfg)
}

func outputResult(result *SortResult, cfg Config) error {
	if cfg.JSON {
		if cfg.Compact {
		return xmlout.WriteJSONCompact(os.Stdout, result)
	}
	return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *SortResult) error {
	if len(result.Errors) > 0 && result.Content == "" {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "sort: %s\n", e.Msg)
		}
		return nil
	}

	_, err := io.WriteString(w, result.Content)
	return err
}
