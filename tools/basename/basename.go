package basename

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/synseqack/aict/internal/meta"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("basename", Run)
	tool.RegisterMeta("basename", tool.GenerateSchema("basename", "Print filename portion of file paths", Config{}))

	dict := map[string]string{
		"t":  "timestamp",
		"e":  "entry",
		"p":  "path",
		"b":  "base",
		"stm":"stem",
		"ext":"extension",
		"err":"error",
		"c":  "code",
		"msg":"msg",
	}
	xmlout.RegisterDict("basename", dict)
}

type Config struct {
	XML       bool
	JSON      bool
	Plain     bool
	Pretty    bool
	Dict      bool
	NoCompact bool
}

type BasenameResult struct {
	XMLName   xml.Name        `xml:"basename" json:"-"`
	Paths     []BasenameEntry `xml:"entry,omitempty" json:"e"`
	Timestamp int64           `xml:"timestamp,attr" json:"t"`
	Errors    []BasenameError `xml:"error,omitempty" json:"err"`
}

func (*BasenameResult) isBasenameResult() {}

type BasenameEntry struct {
	XMLName   xml.Name `xml:"entry" json:"-"`
	Path      string   `xml:"path,attr" json:"p"`
	Base      string   `xml:"base,attr" json:"b"`
	Stem      string   `xml:"stem,attr" json:"stm"`
	Extension string   `xml:"extension,attr" json:"ext"`
}

type BasenameError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"msg"`
	Path    string   `xml:"path,attr" json:"p"`
}

func Run(args []string) error {
	cfg, paths := parseFlags(args)

	if len(paths) == 0 {
		return outputResult(&BasenameResult{Timestamp: meta.Now()}, cfg)
	}

	suffix := ""
	if len(paths) > 1 && !strings.HasPrefix(paths[1], "-") {
		suffix = paths[1]
		paths = paths[1:]
	}

	result := &BasenameResult{Timestamp: meta.Now()}
	for _, p := range paths {
		base := filepath.Base(p)
		if suffix != "" && strings.HasSuffix(base, suffix) {
			base = strings.TrimSuffix(base, suffix)
		}

		ext := filepath.Ext(base)
		stem := strings.TrimSuffix(base, ext)

		result.Paths = append(result.Paths, BasenameEntry{
			Path:      p,
			Base:      base,
			Stem:      stem,
			Extension: ext,
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
		case "--dict":
			cfg.Dict = true
		case "--no-compact":
			cfg.NoCompact = true
		default:
			positional = append(positional, arg)
		}
	}

	if !cfg.XML && !cfg.JSON && !cfg.Plain {
		cfg.XML = xmlout.IsXMLMode()
	}

	return cfg, positional
}

func outputResult(result *BasenameResult, cfg Config) error {
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("basename")
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

func writePlain(w io.Writer, result *BasenameResult) error {
	for _, p := range result.Paths {
		fmt.Fprintln(w, p.Base)
	}
	return nil
}
