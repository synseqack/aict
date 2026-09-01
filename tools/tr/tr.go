package tr

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
	tool.Register("tr", Run)
	tool.RegisterMeta("tr", tool.GenerateSchema("tr", "Translate, squeeze, or delete characters from stdin", Config{}))

	dict := map[string]string{
		"t":  "timestamp",
		"fr": "set1",
		"to": "set2",
		"lin":"lines_in",
		"lout":"lines_out",
		"ct": "content",
		"e":  "error",
		"c":  "code",
		"msg":"msg",
	}
	xmlout.RegisterDict("tr", dict)
}

type Config struct {
	Delete  bool `flag:"" desc:"Delete characters in set1"`
	Squeeze bool `flag:"" desc:"Squeeze repeated characters"`
	Set1    string
	Set2    string
	XML     bool
	JSON    bool
	Plain   bool
	Pretty  bool
	NoCompact bool
	Dict    bool
}

type TrResult struct {
	XMLName   xml.Name  `xml:"tr" json:"-"`
	Timestamp int64     `xml:"timestamp,attr" json:"t"`
	Set1      string    `xml:"set1,attr" json:"fr"`
	Set2      string    `xml:"set2,attr" json:"to"`
	LinesIn   int       `xml:"lines_in,attr" json:"lin"`
	LinesOut  int       `xml:"lines_out,attr" json:"lout"`
	Content   string    `xml:"content,omitempty" json:"ct"`
	Errors    []TrError `xml:"error,omitempty" json:"e"`
}

func (*TrResult) isTrResult() {}

type TrError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"msg"`
}

func Run(args []string) error {
	cfg, err := parseFlags(args)
	if err != nil {
		return outputResult(&TrResult{
			Timestamp: meta.Now(),
			Errors:    []TrError{{Code: 1, Msg: err.Error()}},
		}, cfg)
	}

	return trStdin(cfg)
}

func parseFlags(args []string) (Config, error) {
	var cfg Config

	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-d", "--delete":
			cfg.Delete = true
		case "-s", "--squeeze-repeats":
			cfg.Squeeze = true
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

	if len(positional) > 0 {
		cfg.Set1 = positional[0]
	}
	if len(positional) > 1 {
		cfg.Set2 = positional[1]
	}

	if cfg.Set2 == "" && !cfg.Delete && !cfg.Squeeze {
		return cfg, fmt.Errorf("missing operand")
	}

	return cfg, nil
}

func trStdin(cfg Config) error {
	lines, err := readStdin()
	if err != nil {
		return err
	}

	result := &TrResult{
		Timestamp: meta.Now(),
		Set1:      cfg.Set1,
		Set2:      cfg.Set2,
		LinesIn:   len(lines),
	}

	outputLines := processTr(lines, cfg)
	result.LinesOut = len(outputLines)
	result.Content = strings.Join(outputLines, "\n")
	if len(outputLines) > 0 {
		result.Content += "\n"
	}

	return outputResult(result, cfg)
}

func readStdin() ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func processTr(lines []string, cfg Config) []string {
	set1 := expandSet(cfg.Set1)
	set2 := expandSet(cfg.Set2)

	var output []string

	for _, line := range lines {
		if cfg.Delete {
			output = append(output, deleteChars(line, set1))
		} else if cfg.Set2 != "" {
			output = append(output, translateChars(line, set1, set2))
		} else if cfg.Squeeze {
			output = append(output, squeezeChars(line, set1))
		}
	}

	return output
}

func expandSet(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	escape := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if escape {
			switch c {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case '\\':
				result.WriteByte('\\')
			default:
				result.WriteByte(c)
			}
			escape = false
			continue
		}

		if c == '\\' {
			escape = true
			continue
		}

		if i+2 < len(s) && s[i+1] == '-' {
			start := c
			end := s[i+2]
			if start <= end {
				for j := start; j <= end; j++ {
					result.WriteByte(j)
				}
			} else {
				for j := start; j >= end; j-- {
					result.WriteByte(j)
				}
			}
			i += 2
			continue
		}

		result.WriteByte(c)
	}

	return result.String()
}

func translateChars(s, set1, set2 string) string {
	if set1 == "" || set2 == "" {
		return s
	}

	trans := make(map[rune]rune)
	for i := 0; i < len(set1); i++ {
		var r1, r2 rune
		if i < len(set2) {
			r1 = rune(set1[i])
			r2 = rune(set2[i])
		} else {
			r1 = rune(set1[i])
			r2 = rune(set2[len(set2)-1])
		}
		trans[r1] = r2
	}

	var result strings.Builder
	for _, c := range s {
		if r, ok := trans[c]; ok {
			result.WriteRune(r)
		} else {
			result.WriteRune(c)
		}
	}

	return result.String()
}

func deleteChars(s, set string) string {
	del := make(map[rune]bool)
	for _, c := range set {
		del[c] = true
	}

	var result strings.Builder
	for _, c := range s {
		if !del[c] {
			result.WriteRune(c)
		}
	}

	return result.String()
}

func squeezeChars(s, set string) string {
	if set == "" {
		var prev rune
		var result strings.Builder
		first := true
		for _, c := range s {
			if first || c != prev {
				result.WriteRune(c)
				prev = c
				first = false
			}
		}
		return result.String()
	}

	squeeze := make(map[rune]bool)
	for _, c := range set {
		squeeze[c] = true
	}

	var prev rune
	var result strings.Builder
	first := true
	for _, c := range s {
		if !squeeze[c] || first || c != prev {
			result.WriteRune(c)
			prev = c
			first = false
		}
	}

	return result.String()
}

func outputResult(result *TrResult, cfg Config) error {
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("tr")
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

func writePlain(w io.Writer, result *TrResult) error {
	if len(result.Errors) > 0 && result.Content == "" {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "tr: %s\n", e.Msg)
		}
		return nil
	}

	_, err := io.WriteString(w, result.Content)
	return err
}
