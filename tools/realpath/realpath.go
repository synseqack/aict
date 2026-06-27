package realpath

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
	tool.Register("realpath", Run)
	tool.RegisterMeta("realpath", tool.GenerateSchema("realpath", "Print resolved absolute paths", Config{}))
}

type Config struct {
	XML    bool
	JSON   bool
	Plain  bool
	Pretty bool
	Compact bool
}

type RealpathResult struct {
	XMLName   xml.Name        `xml:"realpath"`
	Paths     []RealpathEntry `xml:"entry,omitempty"`
	Timestamp int64           `xml:"timestamp,attr"`
	Errors    []RealpathError `xml:"error,omitempty"`
}

func (*RealpathResult) isRealpathResult() {}

type RealpathEntry struct {
	XMLName  xml.Name `xml:"entry"`
	Path     string   `xml:"path,attr"`
	Absolute string   `xml:"absolute,attr"`
	Exists   string   `xml:"exists,attr"`
	Type     string   `xml:"type,attr"`
}

type RealpathError struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Msg     string   `xml:"msg,attr"`
	Path    string   `xml:"path,attr"`
}

func Run(args []string) error {
	cfg, paths := parseFlags(args)

	if len(paths) == 0 {
		return outputResult(&RealpathResult{Timestamp: meta.Now()}, cfg)
	}

	result := &RealpathResult{Timestamp: meta.Now()}
	for _, p := range paths {
		entry := resolvePath(p)
		result.Paths = append(result.Paths, entry)
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

func resolvePath(path string) RealpathEntry {
	abs, err := filepath.Abs(path)
	if err != nil {
		return RealpathEntry{
			Path:   path,
			Exists: "false",
			Type:   "unknown",
		}
	}

	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		real = abs
	}

	info, err := os.Lstat(real)
	if err != nil {
		return RealpathEntry{
			Path:     path,
			Absolute: real,
			Exists:   "false",
			Type:     "unknown",
		}
	}

	var ftype string
	if info.IsDir() {
		ftype = "directory"
	} else if info.Mode()&os.ModeSymlink != 0 {
		ftype = "symlink"
	} else {
		ftype = "file"
	}

	return RealpathEntry{
		Path:     path,
		Absolute: real,
		Exists:   "true",
		Type:     ftype,
	}
}

func outputResult(result *RealpathResult, cfg Config) error {
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

func writePlain(w io.Writer, result *RealpathResult) error {
	if len(result.Errors) > 0 && len(result.Paths) == 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "realpath: %s: %s\n", e.Path, e.Msg)
		}
		return nil
	}

	for _, p := range result.Paths {
		fmt.Fprintln(w, p.Absolute)
	}
	return nil
}
