package dirname

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/synseqack/aict/internal/meta"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("dirname", Run)
	tool.RegisterMeta("dirname", tool.GenerateSchema("dirname", "Print directory portion of file paths", Config{}))

	dict := map[string]string{
		"t":  "timestamp",
		"e":  "entry",
		"p":  "path",
		"dir":"dir",
		"err":"error",
		"c":  "code",
		"msg":"msg",
	}
	xmlout.RegisterDict("dirname", dict)
}

type Config struct {
	XML    bool
	JSON   bool
	Plain  bool
	Pretty bool
	NoCompact bool
	Dict   bool
}

type DirnameResult struct {
	XMLName   xml.Name       `xml:"dirname" json:"-"`
	Paths     []DirnameEntry `xml:"entry,omitempty" json:"e"`
	Timestamp int64          `xml:"timestamp,attr" json:"t"`
	Errors    []DirnameError `xml:"error,omitempty" json:"err"`
}

func (*DirnameResult) isDirnameResult() {}

type DirnameEntry struct {
	XMLName xml.Name `xml:"entry" json:"-"`
	Path    string   `xml:"path,attr" json:"p"`
	Dir     string   `xml:"dir,attr" json:"dir"`
}

type DirnameError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"msg"`
	Path    string   `xml:"path,attr" json:"p"`
}

func Run(args []string) error {
	cfg, paths := parseFlags(args)

	if len(paths) == 0 {
		return outputResult(&DirnameResult{Timestamp: meta.Now()}, cfg)
	}

	result := &DirnameResult{Timestamp: meta.Now()}
	for _, p := range paths {
		dir := filepath.Dir(p)
		if dir == "" {
			dir = "."
		}

		result.Paths = append(result.Paths, DirnameEntry{
			Path: p,
			Dir:  dir,
		})
	}

	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config
	var positional []string

	for _, arg := range args {
		switch arg {
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

func outputResult(result *DirnameResult, cfg Config) error {
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("dirname")
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

func writePlain(w io.Writer, result *DirnameResult) error {
	for _, p := range result.Paths {
		fmt.Fprintln(w, p.Dir)
	}
	return nil
}
