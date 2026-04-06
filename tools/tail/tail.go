package tail

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
	tool.Register("tail", Run)
}

type Config struct {
	Lines     int
	Bytes     int
	LinesFlag bool
	BytesFlag bool
	Follow    bool
	XML       bool
	JSON      bool
	Plain     bool
	Pretty    bool
}

type TailResult struct {
	XMLName        xml.Name    `xml:"tail"`
	Path           string      `xml:"path,attr"`
	Absolute       string      `xml:"absolute,attr"`
	LinesRequested int         `xml:"lines_requested,attr"`
	BytesRequested int         `xml:"bytes_requested,attr"`
	LinesReturned  int         `xml:"lines_returned,attr"`
	BytesReturned  int         `xml:"bytes_returned,attr"`
	FileTotalLines int         `xml:"file_total_lines,attr"`
	FileTotalBytes int64       `xml:"file_total_bytes,attr"`
	Truncated      string      `xml:"truncated,attr"`
	Language       string      `xml:"language,attr"`
	MIME           string      `xml:"mime,attr"`
	Content        string      `xml:"content,omitempty"`
	Files          []TailFile  `xml:"file,omitempty"`
	Timestamp      int64       `xml:"timestamp,attr"`
	Errors         []TailError `xml:"error,omitempty"`
}

func (*TailResult) isTailResult() {}

type TailFile struct {
	XMLName        xml.Name `xml:"file"`
	Path           string   `xml:"path,attr"`
	LinesRequested int      `xml:"lines_requested,attr"`
	BytesRequested int      `xml:"bytes_requested,attr"`
	LinesReturned  int      `xml:"lines_returned,attr"`
	BytesReturned  int      `xml:"bytes_returned,attr"`
	FileTotalLines int      `xml:"file_total_lines,attr"`
	FileTotalBytes int64    `xml:"file_total_bytes,attr"`
	Truncated      string   `xml:"truncated,attr"`
	Content        string   `xml:"content,omitempty"`
	Language       string   `xml:"language,attr"`
	MIME           string   `xml:"mime,attr"`
}

type TailError struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Msg     string   `xml:"msg,attr"`
	Path    string   `xml:"path,attr"`
}

func Run(args []string) error {
	cfg, paths := parseFlags(args)

	if len(paths) == 0 {
		return outputResult(&TailResult{}, cfg)
	}

	if cfg.BytesFlag && cfg.LinesFlag {
		return fmt.Errorf("cannot use both -n and -c")
	}

	if len(paths) == 1 {
		result, err := tailFile(paths[0], cfg)
		if err != nil {
			return err
		}
		return outputResult(result, cfg)
	}

	result := &TailResult{Timestamp: meta.Now()}
	for _, p := range paths {
		tf, err := tailFile(p, cfg)
		if err != nil {
			return err
		}
		if len(tf.Errors) == 0 {
			result.Files = append(result.Files, TailFile{
				Path:           tf.Path,
				LinesRequested: tf.LinesRequested,
				BytesRequested: tf.BytesRequested,
				LinesReturned:  tf.LinesReturned,
				BytesReturned:  tf.BytesReturned,
				FileTotalLines: tf.FileTotalLines,
				FileTotalBytes: tf.FileTotalBytes,
				Truncated:      tf.Truncated,
				Content:        tf.Content,
				Language:       tf.Language,
				MIME:           tf.MIME,
			})
		} else {
			result.Errors = append(result.Errors, tf.Errors...)
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
		case "-f", "--follow":
			cfg.Follow = true
		case "-q", "--quiet", "-v", "--verbose":
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

func tailFile(path string, cfg Config) (*TailResult, error) {
	resolved, err := pathutil.Resolve(path)
	if err != nil {
		return &TailResult{
			Path:      path,
			Timestamp: meta.Now(),
			Errors:    []TailError{{Code: 1, Msg: err.Error(), Path: path}},
		}, nil
	}

	info, err := os.Lstat(resolved.Absolute)
	if err != nil {
		code := 1
		if os.IsNotExist(err) {
			code = 2
		}
		return &TailResult{
			Path:      resolved.Given,
			Absolute:  resolved.Absolute,
			Timestamp: meta.Now(),
			Errors:    []TailError{{Code: code, Msg: "no such file or directory", Path: resolved.Absolute}},
		}, nil
	}

	if info.IsDir() {
		return &TailResult{
			Path:      resolved.Given,
			Absolute:  resolved.Absolute,
			Timestamp: meta.Now(),
			Errors:    []TailError{{Code: 1, Msg: "is a directory", Path: resolved.Absolute}},
		}, nil
	}

	mime, isBinary, _ := detect.DetectFromFile(resolved.Absolute)
	var language string
	if !isBinary {
		language = detect.LanguageFromFile(resolved.Absolute)
	}

	result := &TailResult{
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
		result.Errors = append(result.Errors, TailError{Code: 1, Msg: "is a binary file", Path: resolved.Absolute})
		return result, nil
	}

	totalLines := countLines(resolved.Absolute)
	result.FileTotalLines = totalLines

	if cfg.BytesFlag {
		content, truncated, err := tailBytes(resolved.Absolute, cfg.Bytes, info.Size())
		if err != nil {
			result.Errors = append(result.Errors, TailError{Code: 1, Msg: err.Error(), Path: resolved.Absolute})
			return result, nil
		}
		result.Content = content
		result.BytesReturned = len(content)
		result.Truncated = strconv.FormatBool(truncated)
		return result, nil
	}

	lines, truncated, err := tailLines(resolved.Absolute, cfg.Lines)
	if err != nil {
		result.Errors = append(result.Errors, TailError{Code: 1, Msg: err.Error(), Path: resolved.Absolute})
		return result, nil
	}

	result.Content = strings.Join(lines, "\n")
	if len(lines) > 0 {
		result.Content += "\n"
	}
	result.LinesReturned = len(lines)
	result.Truncated = strconv.FormatBool(truncated)

	return result, nil
}

func tailLines(path string, n int) ([]string, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, false, err
	}

	if n >= len(lines) {
		return lines, false, nil
	}

	return lines[len(lines)-n:], true, nil
}

func tailBytes(path string, n int, totalSize int64) (string, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer f.Close()

	if totalSize <= int64(n) {
		content, err := io.ReadAll(f)
		return string(content), false, err
	}

	_, err = f.Seek(-int64(n), os.SEEK_END)
	if err != nil {
		return "", false, err
	}

	content, err := io.ReadAll(f)
	if err != nil {
		return "", false, err
	}

	return string(content), true, nil
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

func outputResult(result *TailResult, cfg Config) error {
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *TailResult) error {
	if len(result.Errors) > 0 && result.Content == "" {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "tail: %s: %s\n", e.Path, e.Msg)
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
