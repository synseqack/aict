package head

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/synseqack/aict/internal/detect"
	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("head", Run)
	tool.RegisterMeta("head", tool.GenerateSchema("head", "Display the first N lines of a file", Config{}))

	dict := map[string]string{
		"p":  "path",
		"a":  "absolute",
		"lr": "lines_requested",
		"br": "bytes_requested",
		"lret":"lines_returned",
		"bret":"bytes_returned",
		"ftl":"file_total_lines",
		"ftb":"file_total_bytes",
		"trunc":"truncated",
		"lang":"language",
		"mime":"mime",
		"ct": "content",
		"f":  "file",
		"t":  "timestamp",
		"e":  "error",
		"c":  "code",
		"msg":"msg",
	}
	xmlout.RegisterDict("head", dict)
}

type Config struct {
	Lines     int `flag:"" desc:"Number of lines to show"`
	Bytes     int `flag:"" desc:"Number of bytes to show"`
	LinesFlag bool
	BytesFlag bool
	XML       bool
	JSON      bool
	Plain     bool
	Pretty    bool
	NoCompact  bool
	Dict      bool
}

type HeadResult struct {
	XMLName        xml.Name    `xml:"head" json:"-"`
	Path           string      `xml:"path,attr" json:"p"`
	Absolute       string      `xml:"absolute,attr" json:"a"`
	LinesRequested int         `xml:"lines_requested,attr" json:"lr"`
	BytesRequested int         `xml:"bytes_requested,attr" json:"br"`
	LinesReturned  int         `xml:"lines_returned,attr" json:"lret"`
	BytesReturned  int         `xml:"bytes_returned,attr" json:"bret"`
	FileTotalLines int         `xml:"file_total_lines,attr" json:"ftl"`
	FileTotalBytes int64       `xml:"file_total_bytes,attr" json:"ftb"`
	Truncated      string      `xml:"truncated,attr" json:"trunc"`
	Language       string      `xml:"language,attr" json:"lang"`
	MIME           string      `xml:"mime,attr" json:"mime"`
	Content        string      `xml:"content,omitempty" json:"ct"`
	Files          []HeadFile  `xml:"file,omitempty" json:"f"`
	Timestamp      int64       `xml:"timestamp,attr" json:"t"`
	Errors         []HeadError `xml:"error,omitempty" json:"e"`
}

func (*HeadResult) isHeadResult() {}

type HeadFile struct {
	XMLName        xml.Name `xml:"file" json:"-"`
	Path           string   `xml:"path,attr" json:"p"`
	LinesRequested int      `xml:"lines_requested,attr" json:"lr"`
	BytesRequested int      `xml:"bytes_requested,attr" json:"br"`
	LinesReturned  int      `xml:"lines_returned,attr" json:"lret"`
	BytesReturned  int      `xml:"bytes_returned,attr" json:"bret"`
	FileTotalLines int      `xml:"file_total_lines,attr" json:"ftl"`
	FileTotalBytes int64    `xml:"file_total_bytes,attr" json:"ftb"`
	Truncated      string   `xml:"truncated,attr" json:"trunc"`
	Content        string   `xml:"content,omitempty" json:"ct"`
	Language       string   `xml:"language,attr" json:"lang"`
	MIME           string   `xml:"mime,attr" json:"mime"`
}

type HeadError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"msg"`
	Path    string   `xml:"path,attr" json:"p"`
}

func Run(args []string) error {
	cfg, paths := parseFlags(args)

	if len(paths) == 0 {
		return outputResult(&HeadResult{}, cfg)
	}

	if cfg.BytesFlag && cfg.LinesFlag {
		return fmt.Errorf("cannot use both -n and -c")
	}

	if len(paths) == 1 {
		result, err := headFile(paths[0], cfg)
		if err != nil {
			return err
		}
		return outputResult(result, cfg)
	}

	result := &HeadResult{Timestamp: meta.Now()}
	for _, p := range paths {
		hf, err := headFile(p, cfg)
		if err != nil {
			return err
		}
		if len(hf.Errors) == 0 {
			result.Files = append(result.Files, HeadFile{
				Path:           hf.Path,
				LinesRequested: hf.LinesRequested,
				LinesReturned:  hf.LinesReturned,
				BytesReturned:  hf.BytesReturned,
				FileTotalLines: hf.FileTotalLines,
				FileTotalBytes: hf.FileTotalBytes,
				Truncated:      hf.Truncated,
				Content:        hf.Content,
				Language:       hf.Language,
				MIME:           hf.MIME,
			})
		} else {
			result.Errors = append(result.Errors, hf.Errors...)
		}
	}

	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config
	cfg.Lines = 10
	cfg.Bytes = -1

	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-n", "--lines":
			cfg.LinesFlag = true
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				n, err := strconv.Atoi(args[i+1])
				if err == nil {
					cfg.Lines = n
					i++
				}
			}
		case "-c", "--bytes":
			cfg.BytesFlag = true
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				n, err := strconv.Atoi(args[i+1])
				if err == nil {
					cfg.Bytes = n
					i++
				}
			}
		case "-q", "--quiet", "-v", "--verbose":
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

func headFile(path string, cfg Config) (*HeadResult, error) {
	resolved, err := pathutil.Resolve(path)
	if err != nil {
		return &HeadResult{
			Path:      path,
			Timestamp: meta.Now(),
			Errors:    []HeadError{{Code: 1, Msg: err.Error(), Path: path}},
		}, nil
	}

	info, err := os.Lstat(resolved.Absolute)
	if err != nil {
		code := 1
		if os.IsNotExist(err) {
			code = 2
		}
		return &HeadResult{
			Path:      resolved.Given,
			Absolute:  resolved.Absolute,
			Timestamp: meta.Now(),
			Errors:    []HeadError{{Code: code, Msg: "no such file or directory", Path: resolved.Absolute}},
		}, nil
	}

	if info.IsDir() {
		return &HeadResult{
			Path:      resolved.Given,
			Absolute:  resolved.Absolute,
			Timestamp: meta.Now(),
			Errors:    []HeadError{{Code: 1, Msg: "is a directory", Path: resolved.Absolute}},
		}, nil
	}

	mime, isBinary, _ := detect.DetectFromFile(resolved.Absolute)
	var language string
	if !isBinary {
		language = detect.LanguageFromFile(resolved.Absolute)
	}

	result := &HeadResult{
		Path:           resolved.Given,
		Absolute:       resolved.Absolute,
		Timestamp:      meta.Now(),
		FileTotalBytes: info.Size(),
		MIME:           mime,
		Language:       language,
	}

	if cfg.BytesFlag {
		result.LinesRequested = 0
		result.BytesRequested = cfg.Bytes
	} else {
		if !cfg.LinesFlag {
			cfg.Lines = 10
		}
		result.LinesRequested = cfg.Lines
		result.BytesRequested = 0
	}

	if isBinary {
		result.Truncated = "false"
		result.FileTotalLines = 0
		result.Errors = append(result.Errors, HeadError{Code: 1, Msg: "is a binary file", Path: resolved.Absolute})
		return result, nil
	}

	if cfg.BytesFlag {
		content, truncated, err := readBytes(resolved.Absolute, cfg.Bytes)
		if err != nil {
			result.Errors = append(result.Errors, HeadError{Code: 1, Msg: err.Error(), Path: resolved.Absolute})
			return result, nil
		}
		result.Content = content
		result.BytesReturned = len(content)
		result.Truncated = strconv.FormatBool(truncated)
		result.FileTotalLines = countLines(resolved.Absolute)
		return result, nil
	}

	lines, truncated, err := readLines(resolved.Absolute, cfg.Lines)
	if err != nil {
		result.Errors = append(result.Errors, HeadError{Code: 1, Msg: err.Error(), Path: resolved.Absolute})
		return result, nil
	}

	result.Content = strings.Join(lines, "\n")
	if len(lines) > 0 {
		result.Content += "\n"
	}
	result.LinesReturned = len(lines)
	result.Truncated = strconv.FormatBool(truncated)
	result.FileTotalLines = countLines(resolved.Absolute)

	return result, nil
}

func readLines(path string, n int) ([]string, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lines := make([]string, 0, n)
	truncated := false

	for i := 0; i < n; i++ {
		if !scanner.Scan() {
			break
		}
		lines = append(lines, scanner.Text())
	}

	if scanner.Err() != nil {
		return lines, truncated, scanner.Err()
	}

	if len(lines) == n && scanner.Scan() {
		truncated = true
	}

	return lines, truncated, nil
}

func readBytes(path string, n int) (string, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer f.Close()

	buf := make([]byte, n)
	read, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", false, err
	}

	truncated := false
	if read == n {
		fi, _ := f.Stat()
		if fi != nil && fi.Size() > int64(n) {
			truncated = true
		}
	}

	return string(buf[:read]), truncated, nil
}

func countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		count++
	}
	return count
}

func outputResult(result *HeadResult, cfg Config) error {
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("head")
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

func writePlain(w io.Writer, result *HeadResult) error {
	if len(result.Errors) > 0 && result.Content == "" {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "head: %s: %s\n", e.Path, e.Msg)
		}
		return nil
	}

	if len(result.Files) == 0 {
		_, err := io.WriteString(w, result.Content)
		return err
	}

	for _, f := range result.Files {
		_, err := io.WriteString(w, f.Content)
		if err != nil {
			return err
		}
	}

	return nil
}
