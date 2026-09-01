package file

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/synseqack/aict/internal/detect"
	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("file", Run)
	tool.RegisterMeta("file", tool.GenerateSchema("file", "Determine file type using MIME detection and content analysis", Config{}))

	dict := map[string]string{
		"p":  "path",
		"a":  "absolute",
		"ty": "type",
		"mime":"mime",
		"cat":"category",
		"lang":"language",
		"chs":"charset",
		"exe":"executable",
		"t":  "timestamp",
		"e":  "error",
		"c":  "code",
		"msg":"msg",
	}
	xmlout.RegisterDict("file", dict)
}

type Config struct {
	Brief  bool `flag:"" desc:"Show brief file type only"`
	MIME   bool `flag:"" desc:"Show MIME type only"`
	XML    bool
	JSON   bool
	Plain  bool
	Pretty bool
	NoCompact bool
	Dict   bool
}

type FileResult struct {
	XMLName    xml.Name    `xml:"file" json:"-"`
	Path       string      `xml:"path,attr" json:"p"`
	Absolute   string      `xml:"absolute,attr" json:"a"`
	Type       string      `xml:"type,attr" json:"ty"`
	MIME       string      `xml:"mime,attr" json:"mime"`
	Category   string      `xml:"category,attr" json:"cat"`
	Language   string      `xml:"language,attr" json:"lang"`
	Charset    string      `xml:"charset,attr" json:"chs"`
	Executable string      `xml:"executable,attr" json:"exe"`
	Timestamp  int64       `xml:"timestamp,attr" json:"t"`
	Errors     []FileError `xml:"error,omitempty" json:"e"`
}

func (*FileResult) isFileResult() {}

type FileError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"msg"`
	Path    string   `xml:"path,attr" json:"p"`
}

func Run(args []string) error {
	cfg, paths := parseFlags(args)

	if len(paths) == 0 {
		return outputResult(&FileResult{}, cfg)
	}

	for i, p := range paths {
		r, err := identifyFile(p, cfg)
		if err != nil {
			return err
		}
		if i > 0 {
			fmt.Println()
		}
		if err := outputResult(r, cfg); err != nil {
			return err
		}
	}
	return nil
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config
	var positional []string

	for _, arg := range args {
		switch arg {
		case "-b", "--brief":
			cfg.Brief = true
		case "-i", "--mime":
			cfg.MIME = true
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

func identifyFile(path string, cfg Config) (*FileResult, error) {
	resolved, err := pathutil.Resolve(path)
	if err != nil {
		return &FileResult{
			Path:      path,
			Timestamp: meta.Now(),
			Errors:    []FileError{{Code: 1, Msg: err.Error(), Path: path}},
		}, nil
	}

	info, err := os.Lstat(resolved.Absolute)
	if err != nil {
		code := 1
		if os.IsNotExist(err) {
			code = 2
		}
		return &FileResult{
			Path:      resolved.Given,
			Absolute:  resolved.Absolute,
			Timestamp: meta.Now(),
			Errors:    []FileError{{Code: code, Msg: "no such file or directory", Path: resolved.Absolute}},
		}, nil
	}

	result := &FileResult{
		Path:      resolved.Given,
		Absolute:  resolved.Absolute,
		Timestamp: meta.Now(),
	}

	if info.IsDir() {
		result.Type = "directory"
		result.Category = "directory"
		result.MIME = "inode/directory"
		result.Executable = "false"
		return result, nil
	}

	if info.Mode()&os.ModeSymlink != 0 {
		result.Type = "symlink"
		result.Category = "symlink"
		result.MIME = "inode/symlink"
		result.Executable = "false"
		return result, nil
	}

	mime, isBinary, _ := detect.DetectFromFile(resolved.Absolute)
	result.MIME = mime
	exec := isExecutable(info.Mode())
	result.Executable = fmt.Sprintf("%t", exec)

	lang := detect.LanguageFromFile(resolved.Absolute)
	result.Language = lang

	result.Category = getCategory(mime, isBinary)
	result.Charset = getCharset(resolved.Absolute, isBinary)
	result.Type = getType(result.Category, lang, exec)

	return result, nil
}

func isExecutable(mode os.FileMode) bool {
	return mode&0111 != 0
}

func getCategory(mime string, isBinary bool) string {
	if strings.HasPrefix(mime, "text/") {
		return "text"
	}
	if strings.HasPrefix(mime, "image/") {
		return "image"
	}
	if strings.HasPrefix(mime, "audio/") {
		return "audio"
	}
	if strings.HasPrefix(mime, "video/") {
		return "video"
	}
	if strings.HasPrefix(mime, "application/") {
		return "application"
	}
	if strings.HasPrefix(mime, "inode/") {
		return "filesystem"
	}
	if isBinary {
		return "binary"
	}
	return "unknown"
}

func getCharset(path string, isBinary bool) string {
	if isBinary {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return "binary"
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil {
		return "binary"
	}
	buf = buf[:n]

	if len(buf) >= 3 && buf[0] == 0xEF && buf[1] == 0xBB && buf[2] == 0xBF {
		return "UTF-8"
	}
	if len(buf) >= 2 && buf[0] == 0xFF && buf[1] == 0xFE {
		return "UTF-16LE"
	}
	if len(buf) >= 2 && buf[0] == 0xFE && buf[1] == 0xFF {
		return "UTF-16BE"
	}

	return "UTF-8"
}

func getType(category, language string, executable bool) string {
	if category == "directory" {
		return "directory"
	}
	if category == "symlink" {
		return "symlink"
	}
	if executable {
		return "executable"
	}
	if language != "" {
		return "source"
	}
	if category == "text" {
		return "text"
	}
	if category == "image" || category == "audio" || category == "video" {
		return category
	}
	if category == "binary" {
		return "binary"
	}
	return "data"
}

func outputResult(result *FileResult, cfg Config) error {
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("file")
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

func writePlain(w io.Writer, result *FileResult, cfg Config) error {
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "file: %s: %s\n", e.Path, e.Msg)
		}
		return nil
	}

	if cfg.MIME {
		_, err := fmt.Fprintln(w, result.MIME)
		return err
	}

	if cfg.Brief {
		_, err := fmt.Fprintln(w, result.Type)
		return err
	}

	_, err := fmt.Fprintf(w, "%s: %s\n", result.Path, result.Type)
	return err
}
