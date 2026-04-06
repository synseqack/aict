package du

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/synseqack/aict/internal/format"
	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("du", Run)
}

type Config struct {
	Summarize bool
	HumanSize bool
	All       bool
	MaxDepth  int
	XML       bool
	JSON      bool
	Plain     bool
	Pretty    bool
}

type DuResult struct {
	XMLName    xml.Name  `xml:"du"`
	Timestamp  int64     `xml:"timestamp,attr"`
	TotalBytes int64     `xml:"total_bytes,attr"`
	TotalHuman string    `xml:"total_human,attr"`
	Paths      []DuEntry `xml:"entry,omitempty"`
	Errors     []DuError `xml:"error,omitempty"`
}

func (*DuResult) isDuResult() {}

type DuEntry struct {
	XMLName   xml.Name `xml:"entry"`
	Path      string   `xml:"path,attr"`
	Absolute  string   `xml:"absolute,attr"`
	SizeBytes int64    `xml:"size_bytes,attr"`
	SizeHuman string   `xml:"size_human,attr"`
	Depth     int      `xml:"depth,attr"`
}

type DuError struct {
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

	result := &DuResult{
		Timestamp: meta.Now(),
	}

	for _, p := range paths {
		entries, total, err := calculateDu(p, cfg)
		if err != nil {
			result.Errors = append(result.Errors, DuError{Code: 1, Msg: err.Error(), Path: p})
			continue
		}
		result.Paths = append(result.Paths, entries...)
		result.TotalBytes += total
	}

	result.TotalHuman = format.Size(uint64(result.TotalBytes))

	if cfg.Summarize {
		summaries := make([]DuEntry, 0, len(paths))
		for _, p := range paths {
			summaries = append(summaries, DuEntry{
				Path:      p,
				SizeBytes: result.TotalBytes,
				SizeHuman: result.TotalHuman,
				Depth:     0,
			})
		}
		result.Paths = summaries
	}

	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config
	cfg.MaxDepth = -1

	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-s", "--summarize":
			cfg.Summarize = true
		case "-h", "--human-readable":
			cfg.HumanSize = true
		case "-a", "--all":
			cfg.All = true
		case "--max-depth":
			cfg.MaxDepth = 0
			if i+1 < len(args) {
				n, err := strconv.Atoi(args[i+1])
				if err == nil {
					cfg.MaxDepth = n
					i++
				}
			}
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

func calculateDu(path string, cfg Config) ([]DuEntry, int64, error) {
	if cfg.MaxDepth == 0 {
		cfg.MaxDepth = -1
	}

	resolved, err := pathutil.Resolve(path)
	if err != nil {
		return nil, 0, err
	}

	info, err := os.Lstat(resolved.Absolute)
	if err != nil {
		return nil, 0, err
	}

	var entries []DuEntry
	var total int64

	if info.IsDir() {
		entries, total = walkDir(resolved.Absolute, resolved.Given, 0, cfg)
	} else {
		total = info.Size()
		entries = append(entries, DuEntry{
			Path:      resolved.Given,
			Absolute:  resolved.Absolute,
			SizeBytes: total,
			SizeHuman: format.Size(uint64(total)),
			Depth:     0,
		})
	}

	return entries, total, nil
}

func walkDir(dirpath, displayPath string, depth int, cfg Config) ([]DuEntry, int64) {
	var entries []DuEntry
	var total int64

	ents, err := os.ReadDir(dirpath)
	if err != nil {
		return entries, total
	}

	for _, ent := range ents {
		fullPath := filepath.Join(dirpath, ent.Name())
		displayFullPath := filepath.Join(displayPath, ent.Name())

		info, err := ent.Info()
		if err != nil {
			continue
		}

		if ent.IsDir() {
			subEntries, subSize := walkDir(fullPath, displayFullPath, depth+1, cfg)
			total += subSize

			if cfg.MaxDepth < 0 || depth+1 <= cfg.MaxDepth {
				entries = append(entries, subEntries...)
				entries = append(entries, DuEntry{
					Path:      displayFullPath,
					Absolute:  fullPath,
					SizeBytes: subSize,
					SizeHuman: format.Size(uint64(subSize)),
					Depth:     depth + 1,
				})
			}
		} else {
			size := info.Size()
			total += size

			if cfg.All && (cfg.MaxDepth < 0 || depth+1 <= cfg.MaxDepth) {
				entries = append(entries, DuEntry{
					Path:      displayFullPath,
					Absolute:  fullPath,
					SizeBytes: size,
					SizeHuman: format.Size(uint64(size)),
					Depth:     depth + 1,
				})
			}
		}
	}

	return entries, total
}

func outputResult(result *DuResult, cfg Config) error {
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result, cfg)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *DuResult, cfg Config) error {
	if len(result.Errors) > 0 && len(result.Paths) == 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "du: %s: %s\n", e.Path, e.Msg)
		}
		return nil
	}

	for _, e := range result.Paths {
		fmt.Fprintf(w, "%s\t%s\n", e.SizeHuman, e.Path)
	}

	return nil
}
