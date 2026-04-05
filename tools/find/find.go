package find

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/synseqack/aict/internal/detect"
	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("find", Run)
}

type Config struct {
	Name     string
	Type     string
	MTime    int
	Size     int64
	MaxDepth int
	Invert   bool
	Or       bool
	XML      bool
	JSON     bool
	Plain    bool
	Pretty   bool
}

type FindResult struct {
	XMLName      xml.Name        `xml:"find"`
	SearchRoot   string          `xml:"search_root,attr"`
	Absolute     string          `xml:"absolute,attr"`
	Conditions   []FindCondition `xml:"condition"`
	TotalMatches int             `xml:"total_matches,attr"`
	Timestamp    int64           `xml:"timestamp,attr"`
	Matches      []FindFile      `xml:"match"`
	Errors       []FindError     `xml:"error,omitempty"`
}

func (*FindResult) isFindResult() {}

type FindCondition struct {
	XMLName xml.Name `xml:"condition"`
	Type    string   `xml:"type,attr"`
	Value   string   `xml:"value,attr"`
}

type FindFile struct {
	XMLName      xml.Name `xml:"file"`
	Path         string   `xml:"path,attr"`
	Absolute     string   `xml:"absolute,attr"`
	Type         string   `xml:"type,attr"`
	SizeBytes    int64    `xml:"size_bytes,attr"`
	Modified     int64    `xml:"modified,attr"`
	ModifiedAgoS int64    `xml:"modified_ago_s,attr"`
	Language     string   `xml:"language,attr"`
	MIME         string   `xml:"mime,attr"`
	Depth        int      `xml:"depth,attr"`
}

type FindError struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Msg     string   `xml:"msg,attr"`
	Path    string   `xml:"path,attr"`
}

func Run(args []string) error {
	cfg, searchRoot := parseFlags(args)

	if searchRoot == "" {
		searchRoot = "."
	}

	resolved, err := pathutil.Resolve(searchRoot)
	if err != nil {
		return outputResult(&FindResult{
			SearchRoot: searchRoot,
			Timestamp:  meta.Now(),
			Errors:     []FindError{{Code: 1, Msg: err.Error(), Path: searchRoot}},
		}, cfg)
	}

	info, err := os.Lstat(resolved.Absolute)
	if err != nil {
		code := 2
		if os.IsNotExist(err) {
			code = 2
		}
		return outputResult(&FindResult{
			SearchRoot: searchRoot,
			Absolute:   resolved.Absolute,
			Timestamp:  meta.Now(),
			Errors:     []FindError{{Code: code, Msg: "no such file or directory", Path: resolved.Absolute}},
		}, cfg)
	}

	result := searchPath(resolved.Absolute, resolved.Given, info, cfg)
	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, string) {
	var cfg Config
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-name":
			if i+1 < len(args) {
				cfg.Name = args[i+1]
				i++
			}
		case "-type":
			if i+1 < len(args) {
				cfg.Type = args[i+1]
				i++
			}
		case "-mtime":
			if i+1 < len(args) {
				n, _ := strconv.Atoi(args[i+1])
				cfg.MTime = n
				i++
			}
		case "-size":
			if i+1 < len(args) {
				sizeStr := args[i+1]
				size, err := strconv.ParseInt(strings.TrimSuffix(sizeStr, "c"), 10, 64)
				if err == nil {
					cfg.Size = size
				}
				i++
			}
		case "-maxdepth":
			if i+1 < len(args) {
				n, _ := strconv.Atoi(args[i+1])
				cfg.MaxDepth = n
				i++
			}
		case "-not":
			cfg.Invert = true
		case "-o", "-or":
			cfg.Or = true
		case "--xml", "-xml":
			cfg.XML = true
		case "--json", "-json":
			cfg.JSON = true
		case "--plain", "-plain":
			cfg.Plain = true
		case "--pretty", "-pretty":
			cfg.Pretty = true
		default:
			if !strings.HasPrefix(arg, "-") {
				positional = append(positional, arg)
			}
		}
	}

	if !cfg.XML && !cfg.JSON && !cfg.Plain {
		cfg.XML = xmlout.IsXMLMode()
	}

	var searchRoot string
	if len(positional) > 0 {
		searchRoot = positional[0]
	}

	return cfg, searchRoot
}

func searchPath(absPath, givenPath string, info os.FileInfo, cfg Config) *FindResult {
	result := &FindResult{
		SearchRoot: givenPath,
		Absolute:   absPath,
		Timestamp:  meta.Now(),
	}

	if cfg.Name != "" {
		result.Conditions = append(result.Conditions, FindCondition{Type: "name", Value: cfg.Name})
	}
	if cfg.Type != "" {
		result.Conditions = append(result.Conditions, FindCondition{Type: "type", Value: cfg.Type})
	}
	if cfg.MTime != 0 {
		result.Conditions = append(result.Conditions, FindCondition{Type: "mtime", Value: strconv.Itoa(cfg.MTime)})
	}
	if cfg.Size != 0 {
		result.Conditions = append(result.Conditions, FindCondition{Type: "size", Value: strconv.FormatInt(cfg.Size, 10)})
	}
	if cfg.MaxDepth != 0 {
		result.Conditions = append(result.Conditions, FindCondition{Type: "maxdepth", Value: strconv.Itoa(cfg.MaxDepth)})
	}
	if cfg.Invert {
		result.Conditions = append(result.Conditions, FindCondition{Type: "invert", Value: "true"})
	}

	baseDepth := countPathParts(absPath)

	filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		currentDepth := countPathParts(path) - baseDepth

		if cfg.MaxDepth > 0 && currentDepth > cfg.MaxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		matches := evaluateConditions(path, info, cfg, currentDepth)

		if matches {
			mime := "application/octet-stream"
			language := ""

			if !info.IsDir() {
				mime, _, _ = detect.DetectFromFile(path)
				language = detect.LanguageFromFile(path)
			}

			result.Matches = append(result.Matches, FindFile{
				Path:         path,
				Absolute:     path,
				Type:         getFileType(info),
				SizeBytes:    info.Size(),
				Modified:     info.ModTime().Unix(),
				ModifiedAgoS: meta.AgoSeconds(info.ModTime().Unix()),
				Language:     language,
				MIME:         mime,
				Depth:        currentDepth,
			})
		}

		return nil
	})

	result.TotalMatches = len(result.Matches)

	return result
}

func countPathParts(p string) int {
	parts := strings.Split(filepath.ToSlash(p), "/")
	count := 0
	for _, part := range parts {
		if part != "" {
			count++
		}
	}
	return count
}

func evaluateConditions(path string, info os.FileInfo, cfg Config, depth int) bool {
	matches := true

	if cfg.Name != "" {
		name := filepath.Base(path)
		matched, _ := filepath.Match(cfg.Name, name)
		if cfg.Invert {
			matched = !matched
		}
		matches = matches && matched
	}

	if cfg.Type != "" {
		typeMatch := false
		switch cfg.Type {
		case "f":
			typeMatch = info.Mode().IsRegular()
		case "d":
			typeMatch = info.IsDir()
		case "l":
			typeMatch = info.Mode()&os.ModeSymlink != 0
		case "b":
			typeMatch = info.Mode()&os.ModeDevice != 0
		case "c":
			typeMatch = info.Mode()&os.ModeCharDevice != 0
		case "p":
			typeMatch = info.Mode()&os.ModeNamedPipe != 0
		case "s":
			typeMatch = info.Mode()&os.ModeSocket != 0
		}
		if cfg.Invert {
			typeMatch = !typeMatch
		}
		matches = matches && typeMatch
	}

	if cfg.MTime != 0 {
		age := time.Since(info.ModTime())
		days := int(age.Hours() / 24)
		mtimeMatch := false
		if cfg.MTime < 0 {
			mtimeMatch = days < -cfg.MTime
		} else if cfg.MTime > 0 {
			mtimeMatch = days > cfg.MTime
		}
		if cfg.Invert {
			mtimeMatch = !mtimeMatch
		}
		matches = matches && mtimeMatch
	}

	if cfg.Size != 0 {
		sizeMatch := false
		if cfg.Size < 0 {
			sizeMatch = info.Size() < -cfg.Size
		} else if cfg.Size > 0 {
			sizeMatch = info.Size() > cfg.Size
		}
		if cfg.Invert {
			sizeMatch = !sizeMatch
		}
		matches = matches && sizeMatch
	}

	if cfg.Or {
		matches = matches || (cfg.Name == "" && cfg.Type == "" && cfg.MTime == 0 && cfg.Size == 0)
	}

	return matches
}

func getFileType(info os.FileInfo) string {
	mode := info.Mode()
	if mode&os.ModeSymlink != 0 {
		return "symlink"
	}
	if mode.IsDir() {
		return "directory"
	}
	if mode.IsRegular() {
		return "file"
	}
	if mode&os.ModeDevice != 0 {
		return "block"
	}
	if mode&os.ModeCharDevice != 0 {
		return "character"
	}
	if mode&os.ModeNamedPipe != 0 {
		return "pipe"
	}
	if mode&os.ModeSocket != 0 {
		return "socket"
	}
	return "unknown"
}

func outputResult(result *FindResult, cfg Config) error {
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *FindResult) error {
	if len(result.Errors) > 0 && len(result.Matches) == 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "find: %s: %s\n", e.Path, e.Msg)
		}
		return nil
	}

	for _, m := range result.Matches {
		fmt.Fprintln(w, m.Path)
	}

	return nil
}
