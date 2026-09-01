package grep

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/synseqack/aict/internal/detect"
	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("grep", Run)
	tool.RegisterMeta("grep", tool.GenerateSchema("grep", "Search for patterns in files with line numbers and context", Config{}))

	dict := map[string]string{
		"p":  "pattern",
		"fl": "flags",
		"r":  "recursive",
		"cs": "case_sensitive",
		"mt": "match_type",
		"sf": "searched_files",
		"mf": "matched_files",
		"n":  "total_matches",
		"sr": "search_root",
		"t":  "timestamp",
		"m":  "match",
		"f":  "file",
		"ln": "line_num",
		"txt":"text",
		"ob": "offset_bytes",
		"lang":"language",
		"mif":"matches_in_file",
		"e":  "error",
		"c":  "code",
		"msg":"msg",
	}
	xmlout.RegisterDict("grep", dict)
}

type Config struct {
	Pattern          string `flag:"" desc:"Search pattern (regex or literal)"`
	Recursive        bool   `flag:"" desc:"Search recursively in directories"`
	LineNumbers      bool   `flag:"" desc:"Show line numbers"`
	FilesWithMatches bool   `flag:"" desc:"Show only file names with matches"`
	CaseInsensitive  bool   `flag:"" desc:"Case insensitive search"`
	WordMatch        bool   `flag:"" desc:"Match whole words only"`
	AfterContext     int    `flag:"" desc:"Number of context lines after match"`
	BeforeContext    int    `flag:"" desc:"Number of context lines before match"`
	ContextLines     int    `flag:"" desc:"Number of context lines around match"`
	CountOnly        bool   `flag:"" desc:"Count matches only, don't show content"`
	InvertMatch      bool   `flag:"" desc:"Invert match - show non-matching lines"`
	ExtendedRegex    bool   `flag:"" desc:"Use extended regular expressions"`
	FixedStrings     bool   `flag:"" desc:"Treat pattern as literal string"`
	Include          string `flag:"" desc:"Include files matching pattern (e.g., *.go)"`
	ExcludeDir       string `flag:"" desc:"Exclude directories matching pattern"`
	MaxCount         int    `flag:"" desc:"Stop after N matches"`
	Workers          string `flag:"" desc:"Number of parallel workers (integer or 'auto')"`
	XML              bool
	JSON             bool
	Plain            bool
	Pretty           bool
	NoCompact        bool
	Dict             bool
}

type GrepResult struct {
	XMLName       xml.Name        `xml:"grep" json:"-"`
	Pattern       string          `xml:"pattern,attr" json:"p"`
	Flags         string          `xml:"flags,attr" json:"fl"`
	Recursive     string          `xml:"recursive,attr" json:"r"`
	CaseSensitive string          `xml:"case_sensitive,attr" json:"cs"`
	MatchType     string          `xml:"match_type,attr" json:"mt"`
	SearchedFiles int             `xml:"searched_files,attr" json:"sf"`
	MatchedFiles  int             `xml:"matched_files,attr" json:"mf"`
	TotalMatches  int             `xml:"total_matches,attr" json:"n"`
	SearchRoot    string          `xml:"search_root,attr" json:"sr"`
	Timestamp     int64           `xml:"timestamp,attr" json:"t"`
	Matches       []GrepFileMatch `xml:"match" json:"m"`
	Errors        []GrepError     `xml:"error,omitempty" json:"e"`
}

func (*GrepResult) isGrepResult() {}

type GrepFileMatch struct {
	XMLName       xml.Name    `xml:"file" json:"-"`
	Path          string      `xml:"path,attr" json:"p"`
	Absolute      string      `xml:"absolute,attr" json:"a"`
	Language      string      `xml:"language,attr" json:"lang"`
	MatchesInFile int         `xml:"matches_in_file,attr" json:"mif"`
	Lines         []GrepMatch `xml:"line" json:"ln"`
}

type GrepMatch struct {
	XMLName     xml.Name `xml:"line" json:"-"`
	Number      int      `xml:"number,attr" json:"num"`
	Text        string   `xml:"text,attr" json:"txt"`
	OffsetBytes int64    `xml:"offset_bytes,attr" json:"ob"`
	Before      string   `xml:"before,omitempty" json:"bf"`
	After       string   `xml:"after,omitempty" json:"af"`
}

type GrepError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"msg"`
	Path    string   `xml:"path,attr" json:"p"`
}

func Run(args []string) error {
	cfg, searchRoot := parseFlags(args)

	if cfg.Pattern == "" {
		return outputResult(&GrepResult{
			Timestamp:    meta.Now(),
			MatchedFiles: 0,
			TotalMatches: 0,
		}, cfg)
	}

	if searchRoot == "" {
		searchRoot = "."
	}

	resolved, err := pathutil.Resolve(searchRoot)
	if err != nil {
		return outputResult(&GrepResult{
			Pattern:       cfg.Pattern,
			SearchRoot:    searchRoot,
			Timestamp:     meta.Now(),
			SearchedFiles: 0,
			MatchedFiles:  0,
			TotalMatches:  0,
			Errors:        []GrepError{{Code: 1, Msg: err.Error(), Path: searchRoot}},
		}, cfg)
	}

	info, err := os.Lstat(resolved.Absolute)
	if err != nil {
		code := 2
		if os.IsNotExist(err) {
			code = 2
		}
		return outputResult(&GrepResult{
			Pattern:       cfg.Pattern,
			SearchRoot:    searchRoot,
			Timestamp:     meta.Now(),
			SearchedFiles: 0,
			MatchedFiles:  0,
			TotalMatches:  0,
			Errors:        []GrepError{{Code: code, Msg: "no such file or directory", Path: resolved.Absolute}},
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
		case "-r", "-R", "--recursive":
			cfg.Recursive = true
		case "-n", "--line-number":
			cfg.LineNumbers = true
		case "-l", "--files-with-matches":
			cfg.FilesWithMatches = true
		case "-i", "--ignore-case":
			cfg.CaseInsensitive = true
		case "-w", "--word-regexp":
			cfg.WordMatch = true
		case "-A", "--after-context":
			if i+1 < len(args) {
				n, _ := strconv.Atoi(args[i+1])
				cfg.AfterContext = n
				i++
			}
		case "-B", "--before-context":
			if i+1 < len(args) {
				n, _ := strconv.Atoi(args[i+1])
				cfg.BeforeContext = n
				i++
			}
		case "-C", "--context":
			if i+1 < len(args) {
				n, _ := strconv.Atoi(args[i+1])
				cfg.ContextLines = n
				i++
			}
		case "-c", "--count":
			cfg.CountOnly = true
		case "-v", "--invert-match":
			cfg.InvertMatch = true
		case "-E", "--extended-regexp":
			cfg.ExtendedRegex = true
		case "-F", "--fixed-strings":
			cfg.FixedStrings = true
		case "--include":
			if i+1 < len(args) {
				cfg.Include = args[i+1]
				i++
			}
		case "--exclude-dir":
			if i+1 < len(args) {
				cfg.ExcludeDir = args[i+1]
				i++
			}
		case "-m", "--max-count":
			if i+1 < len(args) {
				n, _ := strconv.Atoi(args[i+1])
				cfg.MaxCount = n
				i++
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
		case "--workers":
			if i+1 < len(args) {
				cfg.Workers = args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(arg, "-") && cfg.Pattern == "" {
				cfg.Pattern = arg
			} else {
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

func searchPath(absPath, givenPath string, info os.FileInfo, cfg Config) *GrepResult {
	result := &GrepResult{
		Pattern:       cfg.Pattern,
		Recursive:     strconv.FormatBool(cfg.Recursive),
		CaseSensitive: strconv.FormatBool(!cfg.CaseInsensitive),
		SearchRoot:    givenPath,
		Timestamp:     meta.Now(),
	}

	if cfg.Recursive && info.IsDir() {
		return searchDirectory(absPath, givenPath, cfg)
	}

	result.SearchedFiles = 1

	if info.IsDir() {
		return result
	}

	_, isBinary, _ := detect.DetectFromFile(absPath)
	if isBinary {
		return result
	}

	if cfg.Include != "" {
		matched, _ := filepath.Match(cfg.Include, filepath.Base(absPath))
		if !matched {
			return result
		}
	}

	re, err := buildRegexp(cfg.Pattern, cfg)
	if err != nil {
		result.Errors = append(result.Errors, GrepError{Code: 1, Msg: "invalid pattern: " + err.Error(), Path: absPath})
		return result
	}

	matches := findMatches(absPath, re, cfg)
	if len(matches) > 0 {
		result.MatchedFiles = 1
		result.TotalMatches = len(matches)

		language := detect.LanguageFromFile(absPath)
		result.Matches = append(result.Matches, GrepFileMatch{
			Path:          givenPath,
			Absolute:      absPath,
			Language:      language,
			MatchesInFile: len(matches),
			Lines:         matches,
		})
	}

	return result
}

func searchDirectory(dirPath, givenPath string, cfg Config) *GrepResult {
	result := &GrepResult{
		Pattern:       cfg.Pattern,
		Recursive:     "true",
		CaseSensitive: strconv.FormatBool(!cfg.CaseInsensitive),
		SearchRoot:    givenPath,
		Timestamp:     meta.Now(),
	}

	re, err := buildRegexp(cfg.Pattern, cfg)
	if err != nil {
		result.Errors = append(result.Errors, GrepError{Code: 1, Msg: "invalid pattern: " + err.Error(), Path: dirPath})
		return result
	}

	type searchResult struct {
		path         string
		relPath      string
		matches      []GrepMatch
		matchedFiles bool
	}

	fileChan := make(chan string, 100)
	resultChan := make(chan searchResult, 50)

	var wg sync.WaitGroup
	var searchedFiles atomic.Int64

	workerCount := 4
	if cfg.Workers == "auto" {
		workerCount = runtime.NumCPU()
	} else if cfg.Workers != "" {
		n, err := strconv.Atoi(cfg.Workers)
		if err == nil && n > 0 {
			workerCount = n
		}
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileChan {
				_, isBinary, _ := detect.DetectFromFile(path)
				if isBinary {
					continue
				}

				matches := findMatches(path, re, cfg)
				relPath, _ := filepath.Rel(dirPath, path)

				resultChan <- searchResult{
					path:         path,
					relPath:      relPath,
					matches:      matches,
					matchedFiles: len(matches) > 0 || cfg.FilesWithMatches,
				}
			}
		}()
	}

	go func() {
		walker := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				if cfg.ExcludeDir != "" {
					matched, _ := filepath.Match(cfg.ExcludeDir, info.Name())
					if matched {
						return filepath.SkipDir
					}
				}
				return nil
			}

			if cfg.Include != "" {
				matched, _ := filepath.Match(cfg.Include, info.Name())
				if !matched {
					return nil
				}
			}

			searchedFiles.Add(1)
			fileChan <- path

			return nil
		}

		filepath.Walk(dirPath, walker)
		close(fileChan)
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for sr := range resultChan {
		if sr.matchedFiles {
			result.MatchedFiles++
			result.TotalMatches += len(sr.matches)
			language := detect.LanguageFromFile(sr.path)
			result.Matches = append(result.Matches, GrepFileMatch{
				Path:          sr.relPath,
				Absolute:      sr.path,
				Language:      language,
				MatchesInFile: len(sr.matches),
				Lines:         sr.matches,
			})
		}
	}

	result.SearchedFiles = int(searchedFiles.Load())
	return result
}

func buildRegexp(pattern string, cfg Config) (*regexp.Regexp, error) {
	if cfg.FixedStrings {
		pattern = regexp.QuoteMeta(pattern)
	}

	if cfg.WordMatch {
		pattern = `\b` + pattern + `\b`
	}

	flags := ""
	if cfg.CaseInsensitive {
		flags += "i"
	}

	if flags != "" {
		return regexp.Compile("(?" + flags + ")" + pattern)
	}

	return regexp.Compile(pattern)
}

func findMatches(path string, re *regexp.Regexp, cfg Config) []GrepMatch {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var matches []GrepMatch
	var beforeLines []string
	reader := bufio.NewReaderSize(f, 128*1024)
	lineNum := 0
	matchCount := 0
	var offset int64

	for {
		line, _ := reader.ReadString('\n')
		if len(line) == 0 {
			break
		}
		lineNum++
		line = strings.TrimSuffix(line, "\n")
		// err != nil here means EOF on the last line without trailing newline — still process it

		if cfg.InvertMatch {
			if !re.MatchString(line) {
				if cfg.ContextLines > 0 || cfg.BeforeContext > 0 {
					beforeLines = append(beforeLines, line)
					if len(beforeLines) > cfg.ContextLines+cfg.BeforeContext {
						beforeLines = beforeLines[1:]
					}
				}
				matches = append(matches, GrepMatch{
					Number:      lineNum,
					Text:        line,
					OffsetBytes: offset,
				})
			}
		} else {
			if re.MatchString(line) {
				bm := re.FindStringSubmatchIndex(line)
				if len(bm) > 0 {
					before := ""
					if (cfg.ContextLines > 0 || cfg.BeforeContext > 0) && len(beforeLines) > 0 {
						ctx := cfg.ContextLines
						if cfg.BeforeContext > ctx {
							ctx = cfg.BeforeContext
						}
						if len(beforeLines) > ctx {
							beforeLines = beforeLines[len(beforeLines)-ctx:]
						}
						before = strings.Join(beforeLines, "\n")
					}

					after := ""
					if cfg.ContextLines > 0 || cfg.AfterContext > 0 {
						var afterLines []string
						for i := 0; i < cfg.ContextLines || (i < cfg.AfterContext); i++ {
							l, err := reader.ReadString('\n')
							if err != nil {
								break
							}
							lineNum++
							afterLines = append(afterLines, strings.TrimSuffix(l, "\n"))
						}
						if len(afterLines) > 0 {
							after = strings.Join(afterLines, "\n")
						}
					}

					text := line

					matches = append(matches, GrepMatch{
						Number:      lineNum,
						Text:        text,
						OffsetBytes: offset,
						Before:      before,
						After:       after,
					})

					beforeLines = nil

					if cfg.MaxCount > 0 && matchCount >= cfg.MaxCount {
						break
					}
					matchCount++
				}
			}
		}

		offset += int64(len(line) + 1)
	}

	return matches
}

func outputResult(result *GrepResult, cfg Config) error {
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("grep")
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

func writePlain(w io.Writer, result *GrepResult, cfg Config) error {
	if len(result.Errors) > 0 && len(result.Matches) == 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "grep: %s: %s\n", e.Path, e.Msg)
		}
		return nil
	}

	for _, m := range result.Matches {
		if cfg.FilesWithMatches {
			if len(m.Lines) > 0 {
				fmt.Fprintf(w, "%s\n", m.Path)
			}
			continue
		}

		for _, l := range m.Lines {
			if cfg.CountOnly {
				fmt.Fprintf(w, "%s:%d\n", m.Path, l.Number)
				continue
			}
			if cfg.LineNumbers {
				fmt.Fprintf(w, "%s:%d:%s\n", m.Path, l.Number, l.Text)
			} else {
				fmt.Fprintf(w, "%s:%s\n", m.Path, l.Text)
			}
		}
	}

	return nil
}
