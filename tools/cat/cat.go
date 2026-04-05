package cat

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/synseqack/aict/internal/detect"
	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("cat", Run)
}

type Config struct {
	LineNumbers bool
	XML         bool
	JSON        bool
	Plain       bool
	Pretty      bool
}

type CatResult struct {
	XMLName      xml.Name    `xml:"cat"`
	Path         string      `xml:"path,attr"`
	Absolute     string      `xml:"absolute,attr"`
	SizeBytes    int64       `xml:"size_bytes,attr"`
	Lines        int         `xml:"lines,attr"`
	Encoding     string      `xml:"encoding,attr"`
	Language     string      `xml:"language,attr"`
	Binary       string      `xml:"binary,attr"`
	MIME         string      `xml:"mime,attr"`
	Modified     int64       `xml:"modified,attr"`
	ModifiedAgoS int64       `xml:"modified_ago_s,attr"`
	Content      string      `xml:"content,omitempty"`
	Files        []CatResult `xml:"file,omitempty"`
	Errors       []CatError  `xml:"error,omitempty"`
}

func (*CatResult) isCatResult() {}

type CatError struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Msg     string   `xml:"msg,attr"`
	Path    string   `xml:"path,attr"`
}

func Run(args []string) error {
	cfg, paths := parseFlags(args)

	if len(paths) == 0 {
		return outputResult(&CatResult{}, cfg)
	}

	if len(paths) == 1 {
		result, err := catFile(paths[0], cfg)
		if err != nil {
			return err
		}
		return outputResult(result, cfg)
	}

	result := &CatResult{}
	for _, p := range paths {
		cr, err := catFile(p, cfg)
		if err != nil {
			return err
		}
		result.Files = append(result.Files, *cr)
		if cr.Errors == nil {
			result.Lines += cr.Lines
			result.SizeBytes += cr.SizeBytes
		}
	}
	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config
	var positional []string

	for _, arg := range args {
		switch arg {
		case "-n", "--number":
			cfg.LineNumbers = true
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

	return cfg, positional
}

func catFile(path string, cfg Config) (*CatResult, error) {
	resolved, err := pathutil.Resolve(path)
	if err != nil {
		return &CatResult{
			Path:     path,
			Errors:   []CatError{{Code: 1, Msg: err.Error(), Path: path}},
			Binary:   "false",
			Encoding: "binary",
		}, nil
	}

	info, err := os.Lstat(resolved.Absolute)
	if err != nil {
		code := 1
		if os.IsNotExist(err) {
			code = 2
		}
		return &CatResult{
			Path:     resolved.Given,
			Absolute: resolved.Absolute,
			Errors:   []CatError{{Code: code, Msg: "no such file or directory", Path: resolved.Absolute}},
			Binary:   "false",
			Encoding: "binary",
		}, nil
	}

	if info.IsDir() {
		return &CatResult{
			Path:     resolved.Given,
			Absolute: resolved.Absolute,
			Errors:   []CatError{{Code: 1, Msg: "is a directory", Path: resolved.Absolute}},
			Binary:   "false",
			Encoding: "binary",
		}, nil
	}

	mime, isBinary, _ := detect.DetectFromFile(resolved.Absolute)
	language := ""
	if !isBinary {
		language = detect.LanguageFromFile(resolved.Absolute)
	}

	result := &CatResult{
		Path:         resolved.Given,
		Absolute:     resolved.Absolute,
		SizeBytes:    info.Size(),
		Modified:     info.ModTime().Unix(),
		ModifiedAgoS: meta.AgoSeconds(info.ModTime().Unix()),
		MIME:         mime,
		Language:     language,
		Binary:       strconv.FormatBool(isBinary),
		Encoding:     "utf-8",
	}

	if isBinary {
		result.Encoding = "binary"
		result.Content = ""
		result.SizeBytes = info.Size()
		return result, nil
	}

	encoding, content, lines, err := readFileContent(resolved.Absolute)
	if err != nil {
		result.Errors = append(result.Errors, CatError{Code: 1, Msg: err.Error(), Path: resolved.Absolute})
		return result, nil
	}

	result.Encoding = encoding
	result.Lines = lines
	result.Content = content

	return result, nil
}

func readFileContent(path string) (encoding string, content string, lines int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "binary", "", 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "binary", "", 0, err
	}

	if info.Size() == 0 {
		return "utf-8", "", 0, nil
	}

	header := make([]byte, 512)
	n, err := f.Read(header)
	if err != nil && err != io.EOF {
		return "binary", "", 0, err
	}
	header = header[:n]

	if isBinaryContent(header) {
		return "binary", "", 0, nil
	}

	if n >= 3 && header[0] == 0xEF && header[1] == 0xBB && header[2] == 0xBF {
		encoding = "utf-8-bom"
	}

	f.Seek(0, 0)

	const maxFileSize = 10 * 1024 * 1024
	isLargeFile := info.Size() > maxFileSize

	scanner := bufio.NewScanner(f)
	if isLargeFile {
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
	}

	lineCount := 0
	var allLines []string
	var contentBuilder strings.Builder
	maxContentSize := 5 * 1024 * 1024

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		if isLargeFile && contentBuilder.Len()+len(line) > maxContentSize {
			contentBuilder.WriteString("\n... (truncated)")
			break
		}

		if lineCount > 1 {
			contentBuilder.WriteByte('\n')
		}
		contentBuilder.WriteString(line)
		allLines = append(allLines, line)
	}

	if err := scanner.Err(); err != nil {
		return encoding, "", lineCount, err
	}

	if encoding == "utf-8-bom" {
		var buf bytes.Buffer
		for i, line := range allLines {
			if i > 0 {
				buf.WriteByte('\n')
			}
			buf.WriteString(line)
		}
		return encoding, buf.String(), lineCount, nil
	}

	if isLargeFile {
		return encoding, contentBuilder.String(), lineCount, nil
	}

	content = strings.Join(allLines, "\n")
	if len(allLines) > 0 {
		content += "\n"
	}

	return encoding, content, lineCount, nil
}

func isBinaryContent(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	checkLen := len(data)
	if checkLen > 512 {
		checkLen = 512
	}

	for i := 0; i < checkLen; i++ {
		c := data[i]
		if c == 0 {
			return true
		}
	}

	return false
}

func outputResult(result *CatResult, cfg Config) error {
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *CatResult) error {
	if len(result.Errors) > 0 && len(result.Files) == 0 && result.Content == "" {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "cat: %s: %s\n", e.Path, e.Msg)
		}
		return nil
	}

	if len(result.Files) == 0 {
		_, err := io.WriteString(w, result.Content)
		return err
	}

	for _, f := range result.Files {
		if len(f.Errors) > 0 {
			for _, e := range f.Errors {
				fmt.Fprintf(w, "cat: %s: %s\n", e.Path, e.Msg)
			}
			continue
		}
		_, err := io.WriteString(w, f.Content)
		if err != nil {
			return err
		}
	}

	return nil
}

func RunForTest(path string, cfg Config) ([]CatResult, error) {
	if !cfg.XML && !cfg.JSON && !cfg.Plain {
		cfg.XML = true
	}

	paths := []string{path}
	if strings.HasSuffix(path, "/*") || strings.HasSuffix(path, "/**") {
		dir := filepath.Dir(path)
		pattern := filepath.Base(path)
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return nil, err
		}
		paths = matches
	}

	var results []CatResult
	for _, p := range paths {
		result, err := catFile(p, cfg)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}

	return results, nil
}
