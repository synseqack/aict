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
	"github.com/synseqack/aict/internal/filemode"
	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("find", Run)
	tool.RegisterMeta("find", tool.GenerateSchema("find", "Find files by name, type, or modification time", Config{}))

	dict := map[string]string{
		"p":  "path",
		"a":  "absolute",
		"t":  "timestamp",
		"sr": "search_root",
		"n":  "total_matches",
		"s":  "size_bytes",
		"m":  "modified",
		"ma": "modified_ago_s",
		"d":  "depth",
		"lang":"language",
		"mime":"mime",
		"ty": "type",
		"cond":"condition",
		"neg":"negated",
		"md": "maxdepth",
		"e":  "error",
		"c":  "code",
		"msg":"msg",
	}
	xmlout.RegisterDict("find", dict)
}

type Config struct {
	Name     string `flag:"" desc:"File name pattern (supports * and ?)"`
	Type     string `flag:"" desc:"File type: f (regular), d (directory), l (symlink)"`
	MTime    int    `flag:"" desc:"Modified within N days"`
	Size     int64  `flag:"" desc:"File size in bytes"`
	MaxDepth int    `flag:"" desc:"Maximum directory depth"`
	Invert   bool   `flag:"" desc:"Invert match conditions"`
	Or       bool   `flag:"" desc:"OR between conditions"`
	XML      bool
	JSON     bool
	Plain    bool
	Pretty   bool
	NoCompact bool
	Dict     bool

	// predicates preserves argument order and per-predicate negation
	// (-not binds to the next predicate only, GNU find semantics).
	predicates []predicate
}

type predicate struct {
	kind   string // "name", "type", "mtime", "size"
	strVal string
	numVal int64
	negate bool
}

type FindResult struct {
	XMLName      xml.Name        `xml:"find" json:"-"`
	SearchRoot   string          `xml:"search_root,attr" json:"sr"`
	Absolute     string          `xml:"absolute,attr" json:"a"`
	Conditions   []FindCondition `xml:"condition" json:"cond"`
	TotalMatches int             `xml:"total_matches,attr" json:"n"`
	Timestamp    int64           `xml:"timestamp,attr" json:"t"`
	Matches      []FindFile      `xml:"file" json:"matches"`
	Errors       []FindError     `xml:"error,omitempty" json:"e"`
}

func (*FindResult) isFindResult() {}

type FindCondition struct {
	XMLName xml.Name `xml:"condition"`
	Type    string   `xml:"type,attr"`
	Value   string   `xml:"value,attr"`
	Negated string   `xml:"negated,attr,omitempty"`
}

type FindFile struct {
	XMLName      xml.Name `xml:"file" json:"-"`
	Path         string   `xml:"path,attr" json:"p"`
	Absolute     string   `xml:"absolute,attr" json:"a"`
	Type         string   `xml:"type,attr" json:"ty"`
	SizeBytes    int64    `xml:"size_bytes,attr" json:"s"`
	Modified     int64    `xml:"modified,attr" json:"m"`
	ModifiedAgoS int64    `xml:"modified_ago_s,attr" json:"ma"`
	Language     string   `xml:"language,attr" json:"lang"`
	MIME         string   `xml:"mime,attr" json:"mime"`
	Depth        int      `xml:"depth,attr" json:"d"`
}

type FindError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"msg"`
	Path    string   `xml:"path,attr" json:"p"`
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
	pendingNot := false

	addPredicate := func(p predicate) {
		p.negate = pendingNot
		pendingNot = false
		cfg.predicates = append(cfg.predicates, p)
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-name":
			if i+1 < len(args) {
				cfg.Name = args[i+1]
				addPredicate(predicate{kind: "name", strVal: args[i+1]})
				i++
			}
		case "-type":
			if i+1 < len(args) {
				cfg.Type = args[i+1]
				addPredicate(predicate{kind: "type", strVal: args[i+1]})
				i++
			}
		case "-mtime":
			if i+1 < len(args) {
				n, _ := strconv.Atoi(args[i+1])
				cfg.MTime = n
				addPredicate(predicate{kind: "mtime", numVal: int64(n)})
				i++
			}
		case "-size":
			if i+1 < len(args) {
				sizeStr := args[i+1]
				size, err := strconv.ParseInt(strings.TrimSuffix(sizeStr, "c"), 10, 64)
				if err == nil {
					cfg.Size = size
					addPredicate(predicate{kind: "size", numVal: size})
				}
				i++
			}
		case "-maxdepth":
			if i+1 < len(args) {
				n, _ := strconv.Atoi(args[i+1])
				cfg.MaxDepth = n
				i++
			}
		case "-not", "!":
			pendingNot = true
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
		case "--no-compact":
			cfg.NoCompact = true
		case "--dict":
			cfg.Dict = true
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

	cfg.normalizePredicates()

	for _, p := range cfg.predicates {
		value := p.strVal
		if p.kind == "mtime" || p.kind == "size" {
			value = strconv.FormatInt(p.numVal, 10)
		}
		cond := FindCondition{Type: p.kind, Value: value}
		if p.negate {
			cond.Negated = "true"
		}
		result.Conditions = append(result.Conditions, cond)
	}
	if cfg.MaxDepth != 0 {
		result.Conditions = append(result.Conditions, FindCondition{Type: "maxdepth", Value: strconv.Itoa(cfg.MaxDepth)})
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
				Type:         filemode.FileType(info),
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

// normalizePredicates backfills the predicate list when Config was built
// directly from exported fields (e.g. the MCP path) rather than parseFlags.
// In that path the documented Invert flag negates each condition.
func (cfg *Config) normalizePredicates() {
	if len(cfg.predicates) > 0 {
		return
	}
	if cfg.Name != "" {
		cfg.predicates = append(cfg.predicates, predicate{kind: "name", strVal: cfg.Name, negate: cfg.Invert})
	}
	if cfg.Type != "" {
		cfg.predicates = append(cfg.predicates, predicate{kind: "type", strVal: cfg.Type, negate: cfg.Invert})
	}
	if cfg.MTime != 0 {
		cfg.predicates = append(cfg.predicates, predicate{kind: "mtime", numVal: int64(cfg.MTime), negate: cfg.Invert})
	}
	if cfg.Size != 0 {
		cfg.predicates = append(cfg.predicates, predicate{kind: "size", numVal: cfg.Size, negate: cfg.Invert})
	}
}

func evaluateConditions(path string, info os.FileInfo, cfg Config, depth int) bool {
	if len(cfg.predicates) == 0 {
		return true
	}

	for _, p := range cfg.predicates {
		matched := evalPredicate(p, path, info)
		if p.negate {
			matched = !matched
		}
		if cfg.Or {
			if matched {
				return true
			}
		} else if !matched {
			return false
		}
	}
	return !cfg.Or
}

func evalPredicate(p predicate, path string, info os.FileInfo) bool {
	switch p.kind {
	case "name":
		matched, _ := filepath.Match(p.strVal, filepath.Base(path))
		return matched
	case "type":
		switch p.strVal {
		case "f":
			return info.Mode().IsRegular()
		case "d":
			return info.IsDir()
		case "l":
			return info.Mode()&os.ModeSymlink != 0
		case "b":
			return info.Mode()&os.ModeDevice != 0
		case "c":
			return info.Mode()&os.ModeCharDevice != 0
		case "p":
			return info.Mode()&os.ModeNamedPipe != 0
		case "s":
			return info.Mode()&os.ModeSocket != 0
		}
		return false
	case "mtime":
		days := int64(time.Since(info.ModTime()).Hours() / 24)
		if p.numVal < 0 {
			return days < -p.numVal
		}
		return days > p.numVal
	case "size":
		if p.numVal < 0 {
			return info.Size() < -p.numVal
		}
		return info.Size() > p.numVal
	}
	return false
}

func outputResult(result *FindResult, cfg Config) error {
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("find")
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
