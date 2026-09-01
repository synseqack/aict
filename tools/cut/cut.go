package cut

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/synseqack/aict/internal/meta"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("cut", Run)
	tool.RegisterMeta("cut", tool.GenerateSchema("cut", "Cut out sections of each line from files", Config{}))

	dict := map[string]string{
		"t":  "timestamp",
		"del":"delimiter",
		"flds":"fields",
		"lp": "lines_processed",
		"ct": "content",
		"e":  "error",
		"c":  "code",
		"msg":"msg",
	}
	xmlout.RegisterDict("cut", dict)
}

type Config struct {
	Fields     string `flag:"" desc:"Select fields (e.g., 1,3-5)"`
	Delimiter  string `flag:"" desc:"Field delimiter (default: tab)"`
	Characters string `flag:"" desc:"Select characters (e.g., 1-10)"`
	OnlyDelim  bool   `flag:"" desc:"Only print lines with delimiter"`
	XML        bool
	JSON       bool
	Plain      bool
	Pretty     bool
	NoCompact bool
	Dict       bool
}

type CutResult struct {
	XMLName        xml.Name   `xml:"cut" json:"-"`
	Timestamp      int64      `xml:"timestamp,attr" json:"t"`
	Delimiter      string     `xml:"delimiter,attr" json:"del"`
	Fields         string     `xml:"fields,attr" json:"flds"`
	LinesProcessed int        `xml:"lines_processed,attr" json:"lp"`
	Content        string     `xml:"content,omitempty" json:"ct"`
	Errors         []CutError `xml:"error,omitempty" json:"e"`
}

func (*CutResult) isCutResult() {}

type CutError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"msg"`
}

func Run(args []string) error {
	cfg, paths := parseFlags(args)

	if len(paths) == 0 {
		return cutStdin(cfg)
	}

	result := &CutResult{
		Timestamp: meta.Now(),
		Delimiter: cfg.Delimiter,
		Fields:    cfg.Fields,
	}

	for _, path := range paths {
		lines, err := readLines(path)
		if err != nil {
			result.Errors = append(result.Errors, CutError{Code: 1, Msg: err.Error()})
			continue
		}
		result.LinesProcessed += len(lines)

		cutLines := processCut(lines, cfg)
		result.Content += strings.Join(cutLines, "\n")
		if path != paths[len(paths)-1] || len(cutLines) > 0 {
			result.Content += "\n"
		}
	}

	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config
	cfg.Delimiter = "\t"

	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-d", "--delimiter":
			if i+1 < len(args) {
				cfg.Delimiter = args[i+1]
				i++
			}
		case "-f", "--fields":
			if i+1 < len(args) {
				cfg.Fields = args[i+1]
				i++
			}
		case "-c", "--characters":
			if i+1 < len(args) {
				cfg.Characters = args[i+1]
				i++
			}
		case "--only-delimited", "-s":
			cfg.OnlyDelim = true
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

func processCut(lines []string, cfg Config) []string {
	var output []string

	if cfg.Fields != "" {
		fieldIndices := parseFieldList(cfg.Fields)
		for _, line := range lines {
			fields := strings.Split(line, cfg.Delimiter)
			var selected []string
			for _, idx := range fieldIndices {
				if idx > 0 && idx <= len(fields) {
					selected = append(selected, fields[idx-1])
				} else if idx < 0 && len(fields) >= -idx {
					selected = append(selected, fields[len(fields)+idx])
				}
			}
			if len(selected) > 0 || !cfg.OnlyDelim {
				output = append(output, strings.Join(selected, cfg.Delimiter))
			}
		}
	} else if cfg.Characters != "" {
		charRanges := parseCharList(cfg.Characters)
		for _, line := range lines {
			output = append(output, extractChars(line, charRanges))
		}
	}

	return output
}

func parseFieldList(s string) []int {
	var indices []int
	parts := strings.Split(s, ",")
	for _, p := range parts {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			indices = append(indices, n)
		}
	}
	return indices
}

func parseCharList(s string) [][2]int {
	var ranges [][2]int
	parts := strings.Split(s, ",")
	for _, p := range parts {
		if strings.Contains(p, "-") {
			rangeParts := strings.Split(p, "-")
			start, _ := strconv.Atoi(rangeParts[0])
			end, _ := strconv.Atoi(rangeParts[1])
			ranges = append(ranges, [2]int{start - 1, end})
		} else {
			n, _ := strconv.Atoi(p)
			ranges = append(ranges, [2]int{n - 1, n})
		}
	}
	return ranges
}

func extractChars(s string, ranges [][2]int) string {
	var result strings.Builder
	chars := []rune(s)
	for _, r := range ranges {
		start := r[0]
		end := r[1]
		if start < 0 {
			start = 0
		}
		if end > len(chars) {
			end = len(chars)
		}
		if start < end {
			result.WriteString(string(chars[start:end]))
		}
	}
	return result.String()
}

func cutStdin(cfg Config) error {
	lines, err := readStdin()
	if err != nil {
		return err
	}

	result := &CutResult{
		Timestamp:      meta.Now(),
		Delimiter:      cfg.Delimiter,
		Fields:         cfg.Fields,
		LinesProcessed: len(lines),
	}

	cutLines := processCut(lines, cfg)
	result.Content = strings.Join(cutLines, "\n")
	if len(cutLines) > 0 {
		result.Content += "\n"
	}

	return outputResult(result, cfg)
}

func outputResult(result *CutResult, cfg Config) error {
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("cut")
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
		return writePlain(os.Stdout, result)
	}
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *CutResult) error {
	if len(result.Errors) > 0 && result.Content == "" {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "cut: %s\n", e.Msg)
		}
		return nil
	}

	_, err := io.WriteString(w, result.Content)
	return err
}
