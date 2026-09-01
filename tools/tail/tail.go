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
	tool.RegisterMeta("tail", tool.GenerateSchema("tail", "Display the last N lines of a file", Config{}))

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
	xmlout.RegisterDict("tail", dict)
}

type Config struct {
	Lines     int `flag:"" desc:"Number of lines to show"`
	Bytes     int `flag:"" desc:"Number of bytes to show"`
	LinesFlag bool
	BytesFlag bool
	Follow    bool `flag:"" desc:"Follow file updates in real-time"`
	XML       bool
	JSON      bool
	Plain     bool
	Pretty    bool
	NoCompact  bool
	Dict      bool
}

type TailResult struct {
	XMLName        xml.Name    `xml:"tail" json:"-"`
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
	Files          []TailFile  `xml:"file,omitempty" json:"f"`
	Timestamp      int64       `xml:"timestamp,attr" json:"t"`
	Errors         []TailError `xml:"error,omitempty" json:"e"`
}

func (*TailResult) isTailResult() {}

type TailFile struct {
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

type TailError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"msg"`
	Path    string   `xml:"path,attr" json:"p"`
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

	_, err = f.Seek(-int64(n), io.SeekEnd)
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
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("tail")
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
