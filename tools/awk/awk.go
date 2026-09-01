package awk

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("awk", Run)
	tool.RegisterMeta("awk", tool.GenerateSchema("awk", "Extract fields and apply pattern-action rules to file contents", Config{}))
	xmlout.RegisterDict("awk", map[string]string{
		"p": "program",
		"fs": "field_sep",
		"tl": "total_lines",
		"n": "number",
		"o": "output",
	})
}

type Config struct {
	FieldSep  string `flag:"" desc:"Field separator (-F)"`
	Program   string `flag:"" desc:"Awk program"`
	XML       bool
	JSON      bool
	Plain     bool
	Pretty    bool
	Dict      bool
	NoCompact bool
}

type AwkResult struct {
	XMLName    xml.Name   `xml:"awk" json:"-"`
	Program    string     `xml:"program,attr" json:"p"`
	FieldSep   string     `xml:"field_sep,attr" json:"fs"`
	TotalLines int        `xml:"total_lines,attr" json:"tl"`
	Timestamp  int64      `xml:"timestamp,attr" json:"t"`
	Files      []AwkFile  `xml:"file,omitempty" json:"files,omitempty"`
	Errors     []AwkError `xml:"error,omitempty" json:"errors,omitempty"`
}

func (*AwkResult) isAwkResult() {}

type AwkFile struct {
	XMLName xml.Name  `xml:"file" json:"-"`
	Path    string    `xml:"path,attr" json:"p"`
	Lines   []AwkLine `xml:"line,omitempty" json:"lines,omitempty"`
}

type AwkLine struct {
	XMLName xml.Name `xml:"line" json:"-"`
	Number  int      `xml:"number,attr" json:"n"`
	Output  string   `xml:"output,attr" json:"o"`
}

type AwkError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"m"`
	Path    string   `xml:"path,attr,omitempty" json:"p,omitempty"`
}

type rule struct {
	pattern string // "" = match all, "/re/" = regex, "BEGIN", "END"
	patRe   *regexp.Regexp
	action  string
}

func Run(args []string) error {
	cfg, paths := parseFlags(args)

	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("awk")
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

	result := &AwkResult{
		Program:   cfg.Program,
		FieldSep:  cfg.FieldSep,
		Timestamp: meta.Now(),
	}

	if cfg.Program == "" {
		result.Errors = append(result.Errors, AwkError{Code: 1, Msg: "no program specified"})
		return outputResult(result, cfg)
	}

	rules, err := parseProgram(cfg.Program)
	if err != nil {
		result.Errors = append(result.Errors, AwkError{Code: 1, Msg: "parse error: " + err.Error()})
		return outputResult(result, cfg)
	}

	fs := cfg.FieldSep
	if fs == "" {
		fs = " "
	}

	// Check for BEGIN FS override
	for _, r := range rules {
		if r.pattern == "BEGIN" && strings.Contains(r.action, "FS=") {
			// Extract FS value from BEGIN block
			if idx := strings.Index(r.action, "FS="); idx >= 0 {
				rest := r.action[idx+3:]
				rest = strings.TrimSpace(rest)
				if len(rest) > 0 && (rest[0] == '"' || rest[0] == '\'') {
					end := strings.IndexByte(rest[1:], rest[0])
					if end >= 0 {
						fs = rest[1 : end+1]
					}
				}
			}
		}
	}

	// BEGIN actions
	var beginOutput []string
	nr := 0
	for _, r := range rules {
		if r.pattern == "BEGIN" {
			out := executeAction(r.action, "", nil, nr, 0, fs)
			if out != "" {
				beginOutput = append(beginOutput, out)
			}
		}
	}

	for _, p := range paths {
		resolved, err := pathutil.Resolve(p)
		if err != nil {
			result.Errors = append(result.Errors, AwkError{Code: 1, Msg: err.Error(), Path: p})
			continue
		}

		f, err := os.Open(resolved.Absolute)
		if err != nil {
			result.Errors = append(result.Errors, AwkError{Code: 1, Msg: err.Error(), Path: p})
			continue
		}

		af, fileNR, err := processFile(f, resolved.Given, rules, fs, nr)
		f.Close()
		if err != nil {
			result.Errors = append(result.Errors, AwkError{Code: 1, Msg: err.Error(), Path: p})
			continue
		}
		nr = fileNR
		result.Files = append(result.Files, *af)
	}

	result.TotalLines = nr

	// END actions
	for _, r := range rules {
		if r.pattern == "END" {
			out := executeAction(r.action, "", nil, nr, 0, fs)
			if out != "" {
				// Append to last file or create a synthetic entry
				line := AwkLine{Number: 0, Output: out}
				if len(result.Files) > 0 {
					result.Files[len(result.Files)-1].Lines = append(result.Files[len(result.Files)-1].Lines, line)
				}
			}
		}
	}

	_ = beginOutput
	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-F":
			if i+1 < len(args) {
				cfg.FieldSep = args[i+1]
				i++
			}
		case "-f", "--program":
			if i+1 < len(args) {
				cfg.Program = args[i+1]
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
		case "--dict":
			cfg.Dict = true
		case "--no-compact":
			cfg.NoCompact = true
		default:
			if cfg.Program == "" && !strings.HasPrefix(arg, "-") {
				cfg.Program = arg
			} else if !strings.HasPrefix(arg, "-") {
				positional = append(positional, arg)
			}
		}
	}

	if !cfg.XML && !cfg.JSON && !cfg.Plain {
		cfg.XML = xmlout.IsXMLMode()
	}

	return cfg, positional
}

func parseProgram(prog string) ([]rule, error) {
	var rules []rule
	prog = strings.TrimSpace(prog)

	for len(prog) > 0 {
		prog = strings.TrimSpace(prog)
		if len(prog) == 0 {
			break
		}

		var r rule

		// BEGIN/END
		if strings.HasPrefix(prog, "BEGIN") {
			prog = strings.TrimPrefix(prog, "BEGIN")
			prog = strings.TrimSpace(prog)
			action, rest, err := parseBlock(prog)
			if err != nil {
				return nil, err
			}
			r.pattern = "BEGIN"
			r.action = action
			prog = rest
			rules = append(rules, r)
			continue
		}
		if strings.HasPrefix(prog, "END") {
			prog = strings.TrimPrefix(prog, "END")
			prog = strings.TrimSpace(prog)
			action, rest, err := parseBlock(prog)
			if err != nil {
				return nil, err
			}
			r.pattern = "END"
			r.action = action
			prog = rest
			rules = append(rules, r)
			continue
		}

		// /regex/ pattern
		if strings.HasPrefix(prog, "/") {
			end := 1
			for end < len(prog) && prog[end] != '/' {
				if prog[end] == '\\' {
					end++
				}
				end++
			}
			if end >= len(prog) {
				return nil, fmt.Errorf("unclosed regex")
			}
			pattern := prog[1:end]
			re, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid regex: %w", err)
			}
			r.pattern = "/" + pattern + "/"
			r.patRe = re
			prog = strings.TrimSpace(prog[end+1:])
		}

		// action block
		if strings.HasPrefix(prog, "{") {
			action, rest, err := parseBlock(prog)
			if err != nil {
				return nil, err
			}
			r.action = action
			prog = strings.TrimSpace(rest)
		} else if r.pattern == "" {
			return nil, fmt.Errorf("expected pattern or '{'")
		}

		rules = append(rules, r)
	}

	return rules, nil
}

func parseBlock(s string) (action, rest string, err error) {
	if !strings.HasPrefix(s, "{") {
		return "", s, fmt.Errorf("expected '{'")
	}
	depth := 0
	for i, ch := range s {
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				return strings.TrimSpace(s[1:i]), strings.TrimSpace(s[i+1:]), nil
			}
		}
	}
	return "", s, fmt.Errorf("unclosed '{'")
}

func processFile(r io.Reader, path string, rules []rule, fs string, startNR int) (*AwkFile, int, error) {
	af := &AwkFile{Path: path}
	nr := startNR

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		nr++
		nf := 0
		fields := splitFields(line, fs)
		nf = len(fields)

		for _, rule := range rules {
			if rule.pattern == "BEGIN" || rule.pattern == "END" {
				continue
			}
			if rule.patRe != nil && !rule.patRe.MatchString(line) {
				continue
			}
			out := executeAction(rule.action, line, fields, nr, nf, fs)
			if out != "" {
				af.Lines = append(af.Lines, AwkLine{Number: nr, Output: out})
			}
		}
	}

	return af, nr, scanner.Err()
}

func splitFields(line, fs string) []string {
	if fs == " " || fs == "" {
		return strings.FieldsFunc(line, unicode.IsSpace)
	}
	return strings.Split(line, fs)
}

func executeAction(action, line string, fields []string, nr, nf int, fs string) string {
	action = strings.TrimSpace(action)

	// Handle FS= assignment (already handled in BEGIN)
	if strings.Contains(action, "FS=") {
		return ""
	}

	// print NR
	if action == "print NR" {
		return strconv.Itoa(nr)
	}

	// Handle print statements
	if !strings.HasPrefix(action, "print") {
		return ""
	}

	args := strings.TrimSpace(action[5:])
	return evalPrintArgs(args, line, fields, nr, nf, fs)
}

func evalPrintArgs(args, line string, fields []string, nr, nf int, fs string) string {
	if args == "" {
		return line
	}

	// Split by comma, handling quoted strings
	parts := splitPrintArgs(args)
	var out []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		out = append(out, evalExpr(part, line, fields, nr, nf, fs))
	}
	return strings.Join(out, " ")
}

func splitPrintArgs(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, ch := range s {
		switch ch {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func evalExpr(expr, line string, fields []string, nr, nf int, fs string) string {
	expr = strings.TrimSpace(expr)
	switch expr {
	case "$0":
		return line
	case "NR":
		return strconv.Itoa(nr)
	case "NF":
		return strconv.Itoa(nf)
	case "FS":
		return fs
	}

	// $N field reference
	if strings.HasPrefix(expr, "$") {
		numStr := expr[1:]
		n, err := strconv.Atoi(numStr)
		if err == nil {
			if n == 0 {
				return line
			}
			if n >= 1 && n <= len(fields) {
				return fields[n-1]
			}
			return ""
		}
	}

	// Quoted string literal
	if (strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"")) ||
		(strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'")) {
		return expr[1 : len(expr)-1]
	}

	// Fallback: return as-is
	return expr
}

func outputResult(result *AwkResult, cfg Config) error {
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *AwkResult) error {
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			if e.Path != "" {
				fmt.Fprintf(w, "awk: %s: %s\n", e.Path, e.Msg)
			} else {
				fmt.Fprintf(w, "awk: %s\n", e.Msg)
			}
		}
		return nil
	}
	for _, f := range result.Files {
		for _, l := range f.Lines {
			fmt.Fprintln(w, l.Output)
		}
	}
	return nil
}
