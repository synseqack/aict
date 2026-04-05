package wc

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/synseqack/aict/internal/detect"
	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("wc", Run)
}

type Config struct {
	Lines    bool
	Words    bool
	Bytes    bool
	MaxLines bool
	AllFiles bool
	XML      bool
	JSON     bool
	Plain    bool
	Pretty   bool
}

type WCResult struct {
	XMLName    xml.Name  `xml:"wc"`
	Files      []WCFile  `xml:"file"`
	TotalLines int64     `xml:"total_lines,attr"`
	TotalWords int64     `xml:"total_words,attr"`
	TotalBytes int64     `xml:"total_bytes,attr"`
	Timestamp  int64     `xml:"timestamp,attr"`
	Errors     []WCError `xml:"error,omitempty"`
}

func (*WCResult) isWCResult() {}

type WCFile struct {
	XMLName    xml.Name  `xml:"file"`
	Path       string    `xml:"path,attr"`
	Absolute   string    `xml:"absolute,attr"`
	Lines      int64     `xml:"lines,attr"`
	Words      int64     `xml:"words,attr"`
	Bytes      int64     `xml:"bytes,attr"`
	MaxLineLen int64     `xml:"max_line_len,attr"`
	Language   string    `xml:"language,attr"`
	Errors     []WCError `xml:"error,omitempty"`
}

type WCError struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Msg     string   `xml:"msg,attr"`
	Path    string   `xml:"path,attr"`
}

func Run(args []string) error {
	cfg, paths := parseFlags(args)

	if len(paths) == 0 {
		paths = []string{"."}
	}

	result := &WCResult{
		Timestamp: meta.Now(),
	}

	expandedPaths := expandPaths(paths)

	for _, p := range expandedPaths {
		wc, err := countFile(p, cfg)
		if err != nil {
			return err
		}
		result.Files = append(result.Files, *wc)
		if wc.Errors == nil {
			result.TotalLines += wc.Lines
			result.TotalWords += wc.Words
			result.TotalBytes += wc.Bytes
		}
	}

	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config
	var positional []string

	for _, arg := range args {
		switch arg {
		case "-l", "--lines":
			cfg.Lines = true
		case "-w", "--words":
			cfg.Words = true
		case "-c", "--bytes":
			cfg.Bytes = true
		case "-L", "--max-line-length":
			cfg.MaxLines = true
		case "-a", "--all":
			cfg.AllFiles = true
		case "--xml", "-xml":
			cfg.XML = true
		case "--json", "-json":
			cfg.JSON = true
		case "--plain", "-plain":
			cfg.Plain = true
		case "--pretty", "-pretty":
			cfg.Pretty = true
		default:
			positional = append(positional, arg)
		}
	}

	if !cfg.XML && !cfg.JSON && !cfg.Plain {
		cfg.XML = xmlout.IsXMLMode()
	}

	if !cfg.Lines && !cfg.Words && !cfg.Bytes && !cfg.MaxLines {
		cfg.Lines = true
		cfg.Words = true
		cfg.Bytes = true
	}

	return cfg, positional
}

func expandPaths(paths []string) []string {
	var expanded []string
	for _, p := range paths {
		if strings.Contains(p, "*") || strings.Contains(p, "?") {
			matches, err := filepath.Glob(p)
			if err == nil {
				expanded = append(expanded, matches...)
			}
		} else {
			expanded = append(expanded, p)
		}
	}
	return expanded
}

func countFile(path string, cfg Config) (*WCFile, error) {
	resolved, err := pathutil.Resolve(path)
	if err != nil {
		return &WCFile{
			Path:   path,
			Errors: []WCError{{Code: 1, Msg: err.Error(), Path: path}},
		}, nil
	}

	info, err := os.Lstat(resolved.Absolute)
	if err != nil {
		code := 1
		if os.IsNotExist(err) {
			code = 2
		}
		return &WCFile{
			Path:     resolved.Given,
			Absolute: resolved.Absolute,
			Errors:   []WCError{{Code: code, Msg: "no such file or directory", Path: resolved.Absolute}},
		}, nil
	}

	if info.IsDir() {
		return &WCFile{
			Path:     resolved.Given,
			Absolute: resolved.Absolute,
			Errors:   []WCError{{Code: 1, Msg: "is a directory", Path: resolved.Absolute}},
		}, nil
	}

	result := &WCFile{
		Path:     resolved.Given,
		Absolute: resolved.Absolute,
		Language: detect.LanguageFromFile(resolved.Absolute),
	}

	f, err := os.Open(resolved.Absolute)
	if err != nil {
		result.Errors = append(result.Errors, WCError{Code: 1, Msg: err.Error(), Path: resolved.Absolute})
		return result, nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineCount := int64(0)
	wordCount := int64(0)
	byteCount := int64(0)
	maxLineLen := int64(0)

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		wordCount += countWords(line)
		lineLen := int64(len(line))
		if lineLen > maxLineLen {
			maxLineLen = lineLen
		}
	}

	if err := scanner.Err(); err != nil {
		result.Errors = append(result.Errors, WCError{Code: 1, Msg: err.Error(), Path: resolved.Absolute})
		return result, nil
	}

	stat, _ := f.Stat()
	byteCount = stat.Size()

	result.Lines = lineCount
	result.Words = wordCount
	result.Bytes = byteCount
	result.MaxLineLen = maxLineLen

	return result, nil
}

func countWords(s string) int64 {
	count := int64(0)
	inWord := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			inWord = false
		} else {
			if !inWord {
				count++
				inWord = true
			}
		}
	}
	return count
}

func outputResult(result *WCResult, cfg Config) error {
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result, cfg)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *WCResult, cfg Config) error {
	if len(result.Errors) > 0 && len(result.Files) == 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "wc: %s: %s\n", e.Path, e.Msg)
		}
		return nil
	}

	for _, f := range result.Files {
		if len(f.Errors) > 0 {
			for _, e := range f.Errors {
				fmt.Fprintf(w, "wc: %s: %s\n", e.Path, e.Msg)
			}
			continue
		}

		var parts []string
		if cfg.Lines {
			parts = append(parts, fmt.Sprintf("%d", f.Lines))
		}
		if cfg.Words {
			parts = append(parts, fmt.Sprintf("%d", f.Words))
		}
		if cfg.Bytes {
			parts = append(parts, fmt.Sprintf("%d", f.Bytes))
		}
		if cfg.MaxLines {
			parts = append(parts, fmt.Sprintf("%d", f.MaxLineLen))
		}

		fmt.Fprintf(w, "%s %s\n", strings.Join(parts, " "), f.Path)
	}

	if len(result.Files) > 1 {
		var parts []string
		if cfg.Lines {
			parts = append(parts, fmt.Sprintf("%d", result.TotalLines))
		}
		if cfg.Words {
			parts = append(parts, fmt.Sprintf("%d", result.TotalWords))
		}
		if cfg.Bytes {
			parts = append(parts, fmt.Sprintf("%d", result.TotalBytes))
		}
		if cfg.MaxLines {
			parts = append(parts, "0")
		}

		fmt.Fprintf(w, "%s total\n", strings.Join(parts, " "))
	}

	return nil
}
