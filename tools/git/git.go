package git

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/synseqack/aict/internal/meta"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("git", Run)
	tool.RegisterMeta("git", tool.GenerateSchema("git", "Run git subcommands (status, diff, log, ls-files, blame)", Config{}))
	xmlout.RegisterDict("git", map[string]string{
		"sc": "subcommand",
		"st": "status",
		"sh": "short_hash",
		"ad": "author_date",
		"da": "author_date_ago_s",
		"ln": "line_num",
	})
}

type Config struct {
	Subcommand string
	XML        bool
	JSON       bool
	Plain      bool
	Pretty     bool
	Dict       bool
	NoCompact  bool
}

type GitResult struct {
	XMLName   xml.Name   `xml:"git" json:"-"`
	Timestamp int64      `xml:"timestamp,attr" json:"t"`
	Subcmd    string     `xml:"subcommand,attr" json:"sc"`
	Status    []Status   `xml:"status>file,omitempty" json:"status,omitempty"`
	Files     []File     `xml:"files>file,omitempty" json:"files,omitempty"`
	Log       []Commit   `xml:"log>commit,omitempty" json:"log,omitempty"`
	Blame     []Blame    `xml:"blame>line,omitempty" json:"blame,omitempty"`
	Errors    []GitError `xml:"error,omitempty" json:"errors,omitempty"`
}

func (*GitResult) isGitResult() {}

type GitError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"m"`
	Path    string   `xml:"path,attr" json:"p"`
}

type Status struct {
	XMLName  xml.Name `xml:"file" json:"-"`
	Path     string   `xml:"path,attr" json:"p"`
	Status   string   `xml:"status,attr" json:"st"`
	Original string   `xml:"original,attr,omitempty" json:"o,omitempty"`
}

type File struct {
	XMLName xml.Name `xml:"file" json:"-"`
	Path    string   `xml:"path,attr" json:"p"`
	Mode    string   `xml:"mode,attr,omitempty" json:"mo,omitempty"`
	Blob    string   `xml:"blob,attr,omitempty" json:"b,omitempty"`
}

type Commit struct {
	XMLName    xml.Name `xml:"commit" json:"-"`
	Hash       string   `xml:"hash,attr" json:"h"`
	ShortHash  string   `xml:"short_hash,attr" json:"sh"`
	Author     string   `xml:"author,attr" json:"a"`
	AuthorDate int64    `xml:"author_date,attr" json:"ad"`
	DateAgo    int64    `xml:"author_date_ago_s,attr" json:"da"`
	Message    string   `xml:"message,attr" json:"mg"`
	Files      []string `xml:"files>file,omitempty" json:"fs,omitempty"`
}

type Blame struct {
	XMLName    xml.Name `xml:"line" json:"-"`
	LineNum    int      `xml:"line_num,attr" json:"ln"`
	Commit     string   `xml:"commit,attr" json:"c"`
	Author     string   `xml:"author,attr" json:"a"`
	AuthorDate int64    `xml:"author_date,attr" json:"ad"`
	DateAgo    int64    `xml:"author_date_ago_s,attr" json:"da"`
	Content    string   `xml:"content,attr" json:"ct"`
}

func Run(args []string) error {
	cfg, subcmd, subArgs := parseFlags(args)

	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("git")
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

	if subcmd == "" {
		return fmt.Errorf("git subcommand required: status, diff, log, ls-files, blame, show")
	}

	cfg.Subcommand = subcmd

	var result *GitResult
	var err error

	switch subcmd {
	case "status":
		result, err = gitStatus(subArgs)
	case "diff":
		result, err = gitDiff(subArgs)
	case "log":
		result, err = gitLog(subArgs)
	case "ls-files":
		result, err = gitLsFiles(subArgs)
	case "blame":
		result, err = gitBlame(subArgs)
	case "show":
		result, err = gitShow(subArgs)
	default:
		return fmt.Errorf("unknown git subcommand: %s", subcmd)
	}

	if err != nil {
		return err
	}

	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, string, []string) {
	var cfg Config
	var subcmd string
	var subArgs []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
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
			if subcmd == "" && (arg == "status" || arg == "diff" || arg == "log" || arg == "ls-files" || arg == "blame" || arg == "show") {
				subcmd = arg
			} else {
				subArgs = append(subArgs, arg)
			}
		}
	}

	if !cfg.XML && !cfg.JSON && !cfg.Plain {
		cfg.XML = xmlout.IsXMLMode()
	}

	return cfg, subcmd, subArgs
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir, _ = os.Getwd()
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return string(ee.Stderr), fmt.Errorf("git error: %s", string(ee.Stderr))
		}
		return "", err
	}
	return string(out), nil
}

func gitStatus(args []string) (*GitResult, error) {
	result := &GitResult{
		Timestamp: meta.Now(),
		Subcmd:    "status",
	}

	output, err := runGit(append([]string{"status", "--porcelain"}, args...)...)
	if err != nil {
		result.Errors = append(result.Errors, GitError{Code: 1, Msg: err.Error()})
		return result, nil
	}

	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if len(line) < 2 {
			continue
		}
		status := Status{
			Path:   strings.TrimSpace(line[3:]),
			Status: string(line[0]) + string(line[1]),
		}
		if len(line) > 3 {
			if strings.Contains(status.Status, "R") {
				parts := strings.SplitN(status.Path, " -> ", 2)
				if len(parts) == 2 {
					status.Original = parts[0]
					status.Path = parts[1]
				}
			}
		}
		result.Status = append(result.Status, status)
	}

	return result, nil
}

func gitDiff(args []string) (*GitResult, error) {
	result := &GitResult{
		Timestamp: meta.Now(),
		Subcmd:    "diff",
	}

	output, err := runGit(append([]string{"diff", "--name-status"}, args...)...)
	if err != nil {
		result.Errors = append(result.Errors, GitError{Code: 1, Msg: err.Error()})
		return result, nil
	}

	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			status := Status{
				Status: parts[0],
				Path:   parts[len(parts)-1],
			}
			if len(parts) == 3 {
				status.Original = parts[1]
			}
			result.Status = append(result.Status, status)
		}
	}

	return result, nil
}

func gitLog(args []string) (*GitResult, error) {
	result := &GitResult{
		Timestamp: meta.Now(),
		Subcmd:    "log",
	}

	gitArgs := []string{"log", "--format=%H|%h|%an|%at|%s", "-n", "50"}
	gitArgs = append(gitArgs, args...)

	output, err := runGit(gitArgs...)
	if err != nil {
		result.Errors = append(result.Errors, GitError{Code: 1, Msg: err.Error()})
		return result, nil
	}

	now := meta.Now()

	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 5)
		if len(parts) >= 5 {
			authorDate, _ := strconv.ParseInt(parts[3], 10, 64)
			commit := Commit{
				Hash:       parts[0],
				ShortHash:  parts[1],
				Author:     parts[2],
				AuthorDate: authorDate,
				DateAgo:    now - authorDate,
				Message:    parts[4],
			}
			result.Log = append(result.Log, commit)
		}
	}

	return result, nil
}

func gitLsFiles(args []string) (*GitResult, error) {
	result := &GitResult{
		Timestamp: meta.Now(),
		Subcmd:    "ls-files",
	}

	output, err := runGit(append([]string{"ls-files", "-z"}, args...)...)
	if err != nil {
		result.Errors = append(result.Errors, GitError{Code: 1, Msg: err.Error()})
		return result, nil
	}

	files := strings.Split(strings.TrimSuffix(output, "\x00"), "\x00")
	for _, path := range files {
		if path == "" {
			continue
		}
		result.Files = append(result.Files, File{Path: path})
	}

	return result, nil
}

func gitBlame(args []string) (*GitResult, error) {
	result := &GitResult{
		Timestamp: meta.Now(),
		Subcmd:    "blame",
	}

	if len(args) == 0 {
		result.Errors = append(result.Errors, GitError{Code: 1, Msg: "file path required"})
		return result, nil
	}

	output, err := runGit(append([]string{"blame", "--line-porcelain", "-w"}, args...)...)
	if err != nil {
		result.Errors = append(result.Errors, GitError{Code: 1, Msg: err.Error()})
		return result, nil
	}

	lines := strings.Split(output, "\n")
	var current *Blame
	now := meta.Now()

	for _, line := range lines {
		if strings.HasPrefix(line, "author ") {
			if current != nil {
				current.Author = strings.TrimPrefix(line, "author ")
			}
		} else if strings.HasPrefix(line, "author-time ") {
			if current != nil {
				t, _ := strconv.ParseInt(strings.TrimPrefix(line, "author-time "), 10, 64)
				current.AuthorDate = t
				current.DateAgo = now - t
			}
		} else if strings.HasPrefix(line, "committer ") {
			// skip
		} else if strings.HasPrefix(line, "commit ") {
			if current != nil {
				result.Blame = append(result.Blame, *current)
			}
			current = &Blame{
				Commit: strings.TrimPrefix(line, "commit "),
			}
		} else if strings.HasPrefix(line, "filename ") {
			// skip for now
		} else if strings.HasPrefix(line, "\t") && current != nil {
			current.Content = strings.TrimPrefix(line, "\t")
			result.Blame = append(result.Blame, *current)
			current = nil
		}
	}

	if current != nil {
		result.Blame = append(result.Blame, *current)
	}

	// Fix line numbers
	for i := range result.Blame {
		result.Blame[i].LineNum = i + 1
	}

	return result, nil
}

func gitShow(args []string) (*GitResult, error) {
	result := &GitResult{
		Timestamp: meta.Now(),
		Subcmd:    "show",
	}

	ref := "HEAD"
	if len(args) > 0 {
		ref = args[0]
	}

	// Get commit metadata
	commitOut, err := runGit("show", "--no-patch", "--format=%H|%h|%an|%at|%s", ref)
	if err != nil {
		result.Errors = append(result.Errors, GitError{Code: 1, Msg: err.Error()})
		return result, nil
	}

	now := meta.Now()
	for _, line := range strings.Split(strings.TrimSpace(commitOut), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 5)
		if len(parts) >= 5 {
			authorDate, _ := strconv.ParseInt(parts[3], 10, 64)
			result.Log = append(result.Log, Commit{
				Hash:       parts[0],
				ShortHash:  parts[1],
				Author:     parts[2],
				AuthorDate: authorDate,
				DateAgo:    now - authorDate,
				Message:    parts[4],
			})
		}
	}

	// Get changed files
	statOut, err := runGit("show", "--stat", "--format=", ref)
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(statOut), "\n") {
			if line == "" || strings.Contains(line, "changed") {
				continue
			}
			parts := strings.SplitN(line, "|", 2)
			if len(parts) >= 1 {
				path := strings.TrimSpace(parts[0])
				if path != "" {
					result.Status = append(result.Status, Status{Path: path, Status: "M"})
				}
			}
		}
	}

	return result, nil
}

func outputResult(result *GitResult, cfg Config) error {
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *GitResult) error {
	switch result.Subcmd {
	case "status":
		for _, s := range result.Status {
			fmt.Fprintf(w, "%s %s\n", s.Status, s.Path)
		}
	case "diff":
		for _, s := range result.Status {
			fmt.Fprintf(w, "%s %s\n", s.Status, s.Path)
		}
	case "log":
		for _, c := range result.Log {
			fmt.Fprintf(w, "%s %s %s\n", c.ShortHash, c.Author, c.Message)
		}
	case "ls-files":
		for _, f := range result.Files {
			fmt.Fprintf(w, "%s\n", f.Path)
		}
	case "blame":
		for _, b := range result.Blame {
			fmt.Fprintf(w, "%s (%s %d) %s\n", b.Commit, b.Author, b.LineNum, b.Content)
		}
	case "show":
		for _, c := range result.Log {
			fmt.Fprintf(w, "commit %s\nAuthor: %s\n\n    %s\n\n", c.Hash, c.Author, c.Message)
		}
		for _, s := range result.Status {
			fmt.Fprintf(w, "%s\n", s.Path)
		}
	}
	return nil
}
