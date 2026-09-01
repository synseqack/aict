package sed

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("sed", Run)
	tool.RegisterMeta("sed", tool.GenerateSchema("sed", "Apply sed-style transformations to file contents (s, d, p, q commands)", Config{}))
	xmlout.RegisterDict("sed", map[string]string{
		"s": "script",
		"lr": "lines_read",
		"lo": "lines_output",
		"su": "substitutions",
		"n": "number",
		"ct": "content",
		"ch": "changed",
	})
}

type Config struct {
	Suppress      bool   `flag:"" desc:"Suppress default output (-n)"`
	Script        string `flag:"" desc:"Sed script (-e)"`
	ExtendedRegex bool   `flag:"" desc:"Use extended regular expressions (-E)"`
	XML           bool
	JSON          bool
	Plain         bool
	Pretty        bool
	Dict          bool
	NoCompact     bool
}

type SedResult struct {
	XMLName   xml.Name   `xml:"sed" json:"-"`
	Script    string     `xml:"script,attr" json:"s"`
	Timestamp int64      `xml:"timestamp,attr" json:"t"`
	Files     []SedFile  `xml:"file,omitempty" json:"files,omitempty"`
	Errors    []SedError `xml:"error,omitempty" json:"errors,omitempty"`
}

func (*SedResult) isSedResult() {}

type SedFile struct {
	XMLName      xml.Name  `xml:"file" json:"-"`
	Path         string    `xml:"path,attr" json:"p"`
	Absolute     string    `xml:"absolute,attr" json:"a"`
	LinesRead    int       `xml:"lines_read,attr" json:"lr"`
	LinesOutput  int       `xml:"lines_output,attr" json:"lo"`
	Substitutions int      `xml:"substitutions,attr" json:"su"`
	Lines        []SedLine `xml:"line,omitempty" json:"lines,omitempty"`
}

type SedLine struct {
	XMLName xml.Name `xml:"line" json:"-"`
	Number  int      `xml:"number,attr" json:"n"`
	Content string   `xml:"content,attr" json:"ct"`
	Changed bool     `xml:"changed,attr,omitempty" json:"ch,omitempty"`
}

type SedError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"m"`
	Path    string   `xml:"path,attr,omitempty" json:"p,omitempty"`
}

type command struct {
	addrType  string // "none", "line", "last", "regex", "range"
	addrStart int
	addrEnd   int
	addrRe    *regexp.Regexp
	addrRe2   *regexp.Regexp
	negate    bool
	cmd       byte
	subRe     *regexp.Regexp
	subRepl   string
	subGlobal bool
	subCase   bool // case-insensitive
	quit      int  // line number for q
}

func Run(args []string) error {
	cfg, paths := parseFlags(args)

	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("sed")
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

	result := &SedResult{
		Script:    cfg.Script,
		Timestamp: meta.Now(),
	}

	if cfg.Script == "" {
		result.Errors = append(result.Errors, SedError{Code: 1, Msg: "no script specified (use -e 'script')"})
		return outputResult(result, cfg)
	}

	cmds, err := parseScript(cfg.Script, cfg.ExtendedRegex)
	if err != nil {
		result.Errors = append(result.Errors, SedError{Code: 1, Msg: "parse error: " + err.Error()})
		return outputResult(result, cfg)
	}

	if len(paths) == 0 {
		// no files — return empty result
		return outputResult(result, cfg)
	}

	for _, p := range paths {
		resolved, err := pathutil.Resolve(p)
		if err != nil {
			result.Errors = append(result.Errors, SedError{Code: 1, Msg: err.Error(), Path: p})
			continue
		}

		f, err := os.Open(resolved.Absolute)
		if err != nil {
			result.Errors = append(result.Errors, SedError{Code: 1, Msg: err.Error(), Path: p})
			continue
		}

		sf, err := processFile(f, resolved.Given, resolved.Absolute, cmds, cfg)
		f.Close()
		if err != nil {
			result.Errors = append(result.Errors, SedError{Code: 1, Msg: err.Error(), Path: p})
			continue
		}
		result.Files = append(result.Files, *sf)
	}

	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-n", "--quiet":
			cfg.Suppress = true
		case "-e", "--expression":
			if i+1 < len(args) {
				cfg.Script = args[i+1]
				i++
			}
		case "-E", "-r", "--extended-regexp":
			cfg.ExtendedRegex = true
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
			if !strings.HasPrefix(arg, "-") {
				positional = append(positional, arg)
			}
		}
	}

	if !cfg.XML && !cfg.JSON && !cfg.Plain {
		cfg.XML = xmlout.IsXMLMode()
	}

	return cfg, positional
}

func parseScript(script string, extended bool) ([]command, error) {
	var cmds []command
	parts := strings.Split(script, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		c, err := parseCommand(part, extended)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, c)
	}
	return cmds, nil
}

func parseCommand(s string, extended bool) (command, error) {
	var c command
	i := 0

	// Parse optional address
	if i < len(s) && s[i] == '$' {
		c.addrType = "last"
		i++
	} else if i < len(s) && s[i] >= '0' && s[i] <= '9' {
		j := i
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		n, _ := strconv.Atoi(s[i:j])
		c.addrStart = n
		i = j
		if i < len(s) && s[i] == ',' {
			i++
			if i < len(s) && s[i] == '$' {
				c.addrType = "range"
				c.addrEnd = -1 // sentinel for "last"
				i++
			} else {
				j = i
				for j < len(s) && s[j] >= '0' && s[j] <= '9' {
					j++
				}
				c.addrEnd, _ = strconv.Atoi(s[i:j])
				c.addrType = "range"
				i = j
			}
		} else {
			c.addrType = "line"
		}
	} else if i < len(s) && s[i] == '/' {
		// regex address
		end, re, err := parseRegex(s, i, extended)
		if err != nil {
			return c, err
		}
		c.addrType = "regex"
		c.addrRe = re
		i = end
		if i < len(s) && s[i] == ',' {
			i++
			if i < len(s) && s[i] == '/' {
				end2, re2, err := parseRegex(s, i, extended)
				if err != nil {
					return c, err
				}
				c.addrRe2 = re2
				c.addrType = "range-regex"
				i = end2
			}
		}
	}

	// Optional negation
	if i < len(s) && s[i] == '!' {
		c.negate = true
		i++
	}

	if i >= len(s) {
		return c, fmt.Errorf("missing command")
	}

	c.cmd = s[i]
	i++

	switch c.cmd {
	case 's':
		if i >= len(s) {
			return c, fmt.Errorf("s command: missing delimiter")
		}
		delim := s[i]
		i++

		pat, repl, flags, newI, err := parseSub(s, i, delim)
		if err != nil {
			return c, err
		}
		i = newI

		reFlags := ""
		if extended {
			reFlags = "(?m)"
		}
		for _, f := range flags {
			switch f {
			case 'i', 'I':
				reFlags += "(?i)"
				c.subCase = true
			case 'g', 'G':
				c.subGlobal = true
			}
		}

		re, err := regexp.Compile(reFlags + pat)
		if err != nil {
			return c, fmt.Errorf("s command: invalid regex: %w", err)
		}
		c.subRe = re
		c.subRepl = repl
		_ = i
	case 'd', 'p':
		// no args
	case 'q':
		c.quit = 0
		if i < len(s) {
			n, err := strconv.Atoi(strings.TrimSpace(s[i:]))
			if err == nil {
				c.quit = n
			}
		}
	default:
		return c, fmt.Errorf("unsupported command: %c", c.cmd)
	}

	if c.addrType == "" {
		c.addrType = "none"
	}

	return c, nil
}

func parseRegex(s string, start int, extended bool) (int, *regexp.Regexp, error) {
	delim := s[start]
	i := start + 1
	var pat strings.Builder
	for i < len(s) && s[i] != delim {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			pat.WriteByte(s[i])
		} else {
			pat.WriteByte(s[i])
		}
		i++
	}
	if i < len(s) {
		i++ // skip closing delim
	}
	reStr := pat.String()
	if extended {
		reStr = "(?m)" + reStr
	}
	re, err := regexp.Compile(reStr)
	if err != nil {
		return i, nil, fmt.Errorf("invalid regex: %w", err)
	}
	return i, re, nil
}

func parseSub(s string, start int, delim byte) (pat, repl string, flags []byte, end int, err error) {
	i := start
	var p strings.Builder
	for i < len(s) && s[i] != delim {
		if s[i] == '\\' && i+1 < len(s) {
			p.WriteByte(s[i])
			i++
			p.WriteByte(s[i])
		} else {
			p.WriteByte(s[i])
		}
		i++
	}
	pat = p.String()
	if i < len(s) {
		i++ // skip delim
	}

	var r strings.Builder
	for i < len(s) && s[i] != delim {
		if s[i] == '\\' && i+1 < len(s) {
			r.WriteByte(s[i])
			i++
			r.WriteByte(s[i])
		} else {
			r.WriteByte(s[i])
		}
		i++
	}
	repl = r.String()
	if i < len(s) {
		i++ // skip delim
	}

	for i < len(s) && s[i] != ';' && s[i] != ' ' {
		flags = append(flags, s[i])
		i++
	}
	end = i
	return
}

func processFile(r io.Reader, given, absolute string, cmds []command, cfg Config) (*SedFile, error) {
	sf := &SedFile{
		Path:     given,
		Absolute: absolute,
	}

	scanner := bufio.NewScanner(r)
	lineNum := 0
	var allLines []string
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	total := len(allLines)

	for _, line := range allLines {
		lineNum++
		sf.LinesRead++

		output, changed, quit := applyCommands(line, lineNum, total, cmds, cfg.Suppress)

		for _, outLine := range output {
			sf.Lines = append(sf.Lines, SedLine{
				Number:  lineNum,
				Content: outLine,
				Changed: changed,
			})
			sf.LinesOutput++
			if changed {
				sf.Substitutions++
				changed = false // count once per line
			}
		}

		if quit {
			break
		}
	}

	return sf, nil
}

func applyCommands(line string, lineNum, total int, cmds []command, suppress bool) (output []string, changed bool, quit bool) {
	current := line
	deleted := false
	var explicit []string

	for _, c := range cmds {
		if deleted {
			break
		}
		if !matchesAddr(c, lineNum, total, current) {
			continue
		}

		switch c.cmd {
		case 's':
			newLine := applySubstitution(c, current)
			if newLine != current {
				current = newLine
				changed = true
			}
		case 'd':
			deleted = true
		case 'p':
			explicit = append(explicit, current)
		case 'q':
			quit = true
		}
	}

	if deleted {
		return nil, changed, quit
	}

	// Default print (unless suppressed)
	if !suppress {
		output = append(output, current)
	}
	// Explicit p outputs always included
	output = append(output, explicit...)

	return output, changed, quit
}

func matchesAddr(c command, lineNum, total int, line string) bool {
	var match bool
	switch c.addrType {
	case "none":
		match = true
	case "line":
		match = lineNum == c.addrStart
	case "last":
		match = lineNum == total
	case "range":
		end := c.addrEnd
		if end == -1 {
			end = total
		}
		match = lineNum >= c.addrStart && lineNum <= end
	case "regex":
		match = c.addrRe.MatchString(line)
	case "range-regex":
		match = c.addrRe.MatchString(line)
	default:
		match = true
	}
	if c.negate {
		match = !match
	}
	return match
}

func applySubstitution(c command, line string) string {
	if c.subGlobal {
		return c.subRe.ReplaceAllString(line, convertReplacement(c.subRepl))
	}
	return c.subRe.ReplaceAllStringFunc(line, func(match string) string {
		return c.subRe.ReplaceAllString(match, convertReplacement(c.subRepl))
	})
}

// convertReplacement converts sed \1 back-references to Go $1 syntax.
func convertReplacement(repl string) string {
	var sb strings.Builder
	i := 0
	for i < len(repl) {
		if repl[i] == '\\' && i+1 < len(repl) {
			i++
			ch := repl[i]
			if ch >= '1' && ch <= '9' {
				sb.WriteByte('$')
				sb.WriteByte(ch)
			} else {
				sb.WriteByte(ch)
			}
		} else {
			sb.WriteByte(repl[i])
		}
		i++
	}
	return sb.String()
}

func outputResult(result *SedResult, cfg Config) error {
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *SedResult) error {
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			if e.Path != "" {
				fmt.Fprintf(w, "sed: %s: %s\n", e.Path, e.Msg)
			} else {
				fmt.Fprintf(w, "sed: %s\n", e.Msg)
			}
		}
		return nil
	}
	for _, f := range result.Files {
		for _, l := range f.Lines {
			fmt.Fprintln(w, l.Content)
		}
	}
	return nil
}
