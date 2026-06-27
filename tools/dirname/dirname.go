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
}

type Config struct {
	XML    bool
	JSON   bool
	Plain  bool
	Pretty bool
	Compact bool
}

type DirnameResult struct {
	XMLName   xml.Name       `xml:"dirname"`
	Paths     []DirnameEntry `xml:"entry,omitempty"`
	Timestamp int64          `xml:"timestamp,attr"`
	Errors    []DirnameError `xml:"error,omitempty"`
}

func (*DirnameResult) isDirnameResult() {}

type DirnameEntry struct {
	XMLName xml.Name `xml:"entry"`
	Path    string   `xml:"path,attr"`
	Dir     string   `xml:"dir,attr"`
}

type DirnameError struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Msg     string   `xml:"msg,attr"`
	Path    string   `xml:"path,attr"`
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
		case "--compact", "-compact":
			cfg.Compact = true
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
	if cfg.JSON {
		if cfg.Compact {
		return xmlout.WriteJSONCompact(os.Stdout, result)
	}
	return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *DirnameResult) error {
	for _, p := range result.Paths {
		fmt.Fprintln(w, p.Dir)
	}
	return nil
}
