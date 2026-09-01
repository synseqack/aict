package uniq

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/synseqack/aict/internal/meta"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("uniq", Run)
	tool.RegisterMeta("uniq", tool.GenerateSchema("uniq", "Report or filter out repeated lines", Config{}))

	dict := map[string]string{
		"t":  "timestamp",
		"lin":"lines_in",
		"lout":"lines_out",
		"dr": "duplicates_removed",
		"ct": "content",
		"d":  "duplicate",
		"l":  "line",
		"cnt":"count",
		"e":  "error",
		"c":  "code",
		"msg":"msg",
	}
	xmlout.RegisterDict("uniq", dict)
}

type Config struct {
	Count      bool `flag:"" desc:"Prefix lines by number of occurrences"`
	Duplicates bool `flag:"" desc:"Only show duplicate lines"`
	Unique     bool `flag:"" desc:"Only show unique lines"`
	IgnoreCase bool `flag:"" desc:"Case insensitive comparison"`
	XML        bool
	JSON       bool
	Plain      bool
	Pretty     bool
	NoCompact bool
	Dict       bool
}

type UniqResult struct {
	XMLName           xml.Name    `xml:"uniq" json:"-"`
	Timestamp         int64       `xml:"timestamp,attr" json:"t"`
	LinesIn           int         `xml:"lines_in,attr" json:"lin"`
	LinesOut          int         `xml:"lines_out,attr" json:"lout"`
	DuplicatesRemoved int         `xml:"duplicates_removed,attr" json:"dr"`
	Content           string      `xml:"content,omitempty" json:"ct"`
	Duplicates        []UniqDup   `xml:"duplicate,omitempty" json:"d"`
	Errors            []UniqError `xml:"error,omitempty" json:"e"`
}

func (*UniqResult) isUniqResult() {}

type UniqDup struct {
	XMLName xml.Name `xml:"duplicate" json:"-"`
	Line    string   `xml:"line,attr" json:"l"`
	Count   int      `xml:"count,attr" json:"cnt"`
}

type UniqError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"msg"`
}

func Run(args []string) error {
	cfg, paths := parseFlags(args)

	if len(paths) == 0 {
		return uniqStdin(cfg)
	}

	result := &UniqResult{
		Timestamp: meta.Now(),
	}

	var allLines []string

	for _, path := range paths {
		lines, err := readLines(path)
		if err != nil {
			result.Errors = append(result.Errors, UniqError{Code: 1, Msg: err.Error()})
			continue
		}
		allLines = append(allLines, lines...)
	}

	result.LinesIn = len(allLines)

	outputLines, dups := processUniq(allLines, cfg)

	result.LinesOut = len(outputLines)
	result.DuplicatesRemoved = result.LinesIn - result.LinesOut

	result.Content = strings.Join(outputLines, "\n")
	if len(outputLines) > 0 {
		result.Content += "\n"
	}

	if cfg.Count {
		result.Duplicates = dups
	}

	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config

	var positional []string

	for _, arg := range args {
		switch arg {
		case "-c", "--count":
			cfg.Count = true
		case "-d", "--repeated":
			cfg.Duplicates = true
		case "-u", "--unique":
			cfg.Unique = true
		case "-i", "--ignore-case":
			cfg.IgnoreCase = true
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

func processUniq(lines []string, cfg Config) ([]string, []UniqDup) {
	if len(lines) == 0 {
		return nil, nil
	}

	var output []string
	var duplicates []UniqDup
	prev := ""
	count := 0

	for _, line := range lines {
		current := line
		if cfg.IgnoreCase {
			current = strings.ToLower(line)
		}

		if current == prev {
			count++
		} else {
			if prev != "" {
				shouldAdd := true
				if cfg.Duplicates && count == 1 {
					shouldAdd = false
				}
				if cfg.Unique && count > 1 {
					shouldAdd = false
				}

				if shouldAdd {
					output = append(output, prev)
					if cfg.Count {
						duplicates = append(duplicates, UniqDup{Line: prev, Count: count})
					}
				}
			}
			prev = line
			count = 1
		}
	}

	if prev != "" {
		shouldAdd := true
		if cfg.Duplicates && count == 1 {
			shouldAdd = false
		}
		if cfg.Unique && count > 1 {
			shouldAdd = false
		}

		if shouldAdd {
			output = append(output, prev)
			if cfg.Count {
				duplicates = append(duplicates, UniqDup{Line: prev, Count: count})
			}
		}
	}

	return output, duplicates
}

func uniqStdin(cfg Config) error {
	lines, err := readStdin()
	if err != nil {
		return err
	}

	result := &UniqResult{
		Timestamp: meta.Now(),
		LinesIn:   len(lines),
	}

	outputLines, dups := processUniq(lines, cfg)

	result.LinesOut = len(outputLines)
	result.DuplicatesRemoved = result.LinesIn - result.LinesOut

	result.Content = strings.Join(outputLines, "\n")
	if len(outputLines) > 0 {
		result.Content += "\n"
	}

	if cfg.Count {
		result.Duplicates = dups
	}

	return outputResult(result, cfg)
}

func outputResult(result *UniqResult, cfg Config) error {
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("uniq")
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

func writePlain(w io.Writer, result *UniqResult, cfg Config) error {
	if len(result.Errors) > 0 && result.Content == "" {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "uniq: %s\n", e.Msg)
		}
		return nil
	}

	if cfg.Count {
		for _, d := range result.Duplicates {
			fmt.Fprintf(w, "%d %s\n", d.Count, d.Line)
		}
	} else {
		_, err := io.WriteString(w, result.Content)
		return err
	}

	return nil
}
