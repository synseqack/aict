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
	tool.RegisterMeta("du", tool.GenerateSchema("du", "Estimate disk usage of directories and files", Config{}))

	dict := map[string]string{
		"t":  "timestamp",
		"tb": "total_bytes",
		"th": "total_human",
		"e":  "entry",
		"p":  "path",
		"a":  "absolute",
		"s":  "size_bytes",
		"sh": "size_human",
		"d":  "depth",
		"err":"error",
		"c":  "code",
		"msg":"msg",
	}
	xmlout.RegisterDict("du", dict)
}

type Config struct {
	Summarize bool `flag:"" desc:"Show only total for each argument"`
	HumanSize bool `flag:"" desc:"Show sizes in human-readable format"`
	All       bool `flag:"" desc:"Count all files, not just directories"`
	MaxDepth  int  `flag:"" desc:"Maximum depth to show entries"`
	XML       bool
	JSON      bool
	Plain     bool
	Pretty    bool
	NoCompact  bool
	Dict      bool
}

type DuResult struct {
	XMLName    xml.Name  `xml:"du" json:"-"`
	Timestamp  int64     `xml:"timestamp,attr" json:"t"`
	TotalBytes int64     `xml:"total_bytes,attr" json:"tb"`
	TotalHuman string    `xml:"total_human,attr" json:"th"`
	Paths      []DuEntry `xml:"entry,omitempty" json:"e"`
	Errors     []DuError `xml:"error,omitempty" json:"err"`
}

func (*DuResult) isDuResult() {}

type DuEntry struct {
	XMLName   xml.Name `xml:"entry" json:"-"`
	Path      string   `xml:"path,attr" json:"p"`
	Absolute  string   `xml:"absolute,attr" json:"a"`
	SizeBytes int64    `xml:"size_bytes,attr" json:"s"`
	SizeHuman string   `xml:"size_human,attr" json:"sh"`
	Depth     int      `xml:"depth,attr" json:"d"`
}

type DuError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"msg"`
	Path    string   `xml:"path,attr" json:"p"`
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

func calculateDu(path string, cfg Config) ([]DuEntry, int64, error) {
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
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("du")
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
