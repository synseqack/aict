package ls

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/synseqack/aict/internal/detect"
	"github.com/synseqack/aict/internal/filemode"
	"github.com/synseqack/aict/internal/format"
	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("ls", Run)
	tool.RegisterMeta("ls", tool.GenerateSchema("ls", "List directory contents with file metadata including permissions, size, and modification time", Config{}))

	// Register dictionary for compact output
	dict := map[string]string{
		// LSResult
		"p":  "path",
		"a":  "absolute",
		"n":  "total_entries",
		"h":  "hidden",
		"r":  "recursive",
		"t":  "timestamp",
		"ent": "entries",
		"err": "error",
		// FileEntry
		"f":    "file",
		"name": "name",
		"s":    "size_bytes",
		"sh":   "size_human",
		"m":    "modified",
		"ma":   "modified_ago_s",
		"per":  "permissions",
		"mo":   "mode",
		"o":    "owner",
		"g":    "group",
		"exe":  "executable",
		"sym":  "symlink",
		"mime": "mime",
		"lang": "language",
		"bin":  "binary",
		// DirEntry
		"d":   "directory",
		// SymlinkEntry
		"sy":  "symlink_entry",
		"tgt": "target",
		"ta":  "target_absolute",
		"te":  "target_exists",
	}
	xmlout.RegisterDict("ls", dict)
}

type Config struct {
	All        bool `flag:"" desc:"Show hidden files (starting with .)"`
	AlmostAll  bool `flag:"" desc:"Show almost all (exclude . and ..)"`
	SortTime   bool `flag:"" desc:"Sort by modification time, newest first"`
	Reverse    bool `flag:"" desc:"Reverse sort order"`
	Recursive  bool `flag:"" desc:"List subdirectories recursively"`
	XML        bool
	JSON       bool
	Plain      bool
	Pretty     bool
	Dict       bool
	NoCompact  bool
}

type LSItem interface {
	isLSItem()
}

type LSResult struct {
	XMLName      xml.Name      `xml:"ls" json:"-"`
	Path         string        `xml:"path,attr" json:"p"`
	Absolute     string        `xml:"absolute,attr" json:"a"`
	TotalEntries int           `xml:"total_entries,attr" json:"n"`
	Hidden       bool          `xml:"hidden,attr" json:"h"`
	Recursive    bool          `xml:"recursive,attr" json:"r"`
	Timestamp    int64         `xml:"timestamp,attr" json:"t"`
	Entries      []interface{} `xml:",any" json:"ent"`
	Errors       []LSError     `xml:"error,omitempty" json:"err"`
}

func (*LSResult) isLSItem() {}

type FileEntry struct {
	XMLName      xml.Name `xml:"file" json:"-"`
	Name         string   `xml:"name,attr" json:"n"`
	Path         string   `xml:"path,attr" json:"p"`
	Absolute     string   `xml:"absolute,attr" json:"a"`
	SizeBytes    uint64   `xml:"size_bytes,attr" json:"s"`
	SizeHuman    string   `xml:"size_human,attr" json:"sh"`
	Modified     int64    `xml:"modified,attr" json:"m"`
	ModifiedAgoS int64    `xml:"modified_ago_s,attr" json:"ma"`
	Permissions  string   `xml:"permissions,attr" json:"per"`
	Mode         string   `xml:"mode,attr" json:"mo"`
	Owner        string   `xml:"owner,attr" json:"o"`
	Group        string   `xml:"group,attr" json:"g"`
	Executable   string   `xml:"executable,attr" json:"exe"`
	Symlink      string   `xml:"symlink,attr" json:"sym"`
	MIME         string   `xml:"mime,attr" json:"mime"`
	Language     string   `xml:"language,attr" json:"lang"`
	Binary       string   `xml:"binary,attr" json:"bin"`
}

func (FileEntry) isLSItem() {}

type DirEntry struct {
	XMLName      xml.Name `xml:"directory" json:"-"`
	Name         string   `xml:"name,attr" json:"n"`
	Path         string   `xml:"path,attr" json:"p"`
	Modified     int64    `xml:"modified,attr" json:"m"`
	ModifiedAgoS int64    `xml:"modified_ago_s,attr" json:"ma"`
	Permissions  string   `xml:"permissions,attr" json:"per"`
	Mode         string   `xml:"mode,attr" json:"mo"`
	Owner        string   `xml:"owner,attr" json:"o"`
	Group        string   `xml:"group,attr" json:"g"`
}

func (DirEntry) isLSItem() {}

type SymlinkEntry struct {
	XMLName        xml.Name `xml:"symlink" json:"-"`
	Name           string   `xml:"name,attr" json:"n"`
	Path           string   `xml:"path,attr" json:"p"`
	Target         string   `xml:"target,attr" json:"tgt"`
	TargetAbsolute string   `xml:"target_absolute,attr" json:"ta"`
	TargetExists   string   `xml:"target_exists,attr" json:"te"`
	Modified       int64    `xml:"modified,attr" json:"m"`
	ModifiedAgoS   int64    `xml:"modified_ago_s,attr" json:"ma"`
	Permissions    string   `xml:"permissions,attr" json:"per"`
	Mode           string   `xml:"mode,attr" json:"mo"`
}

func (SymlinkEntry) isLSItem() {}

type LSError struct {
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

	for i, p := range paths {
		result, err := listDir(p, cfg)
		if err != nil {
			return err
		}
		if i > 0 {
			fmt.Println()
		}
		if err := outputResult(result, cfg); err != nil {
			return err
		}
	}
	return nil
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config
	var positional []string

	for _, arg := range args {
		switch arg {
		case "-a":
			cfg.All = true
		case "-A":
			cfg.AlmostAll = true
		case "-t":
			cfg.SortTime = true
		case "-r":
			cfg.Reverse = true
		case "-R":
			cfg.Recursive = true
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

func listDir(inputPath string, cfg Config) (*LSResult, error) {
	resolved, err := pathutil.Resolve(inputPath)
	if err != nil {
		return &LSResult{
			Path:      inputPath,
			Timestamp: meta.Now(),
			Errors:    []LSError{{Code: 1, Msg: err.Error(), Path: inputPath}},
		}, nil
	}

	info, err := os.Lstat(resolved.Absolute)
	if err != nil {
		return &LSResult{
			Path:      resolved.Given,
			Absolute:  resolved.Absolute,
			Timestamp: meta.Now(),
			Errors:    []LSError{{Code: 1, Msg: "path not found", Path: resolved.Absolute}},
		}, nil
	}

	if !info.IsDir() {
		entry, err := buildEntry(resolved.Absolute, info, filepath.Base(resolved.Absolute), cfg)
		if err != nil {
			return &LSResult{
				Path:      resolved.Given,
				Absolute:  resolved.Absolute,
				Timestamp: meta.Now(),
				Errors:    []LSError{{Code: 1, Msg: err.Error(), Path: resolved.Absolute}},
			}, nil
		}
		return &LSResult{
			Path:         resolved.Given,
			Absolute:     resolved.Absolute,
			TotalEntries: 1,
			Hidden:       cfg.All || cfg.AlmostAll,
			Recursive:    cfg.Recursive,
			Timestamp:    meta.Now(),
			Entries:      []interface{}{entry},
		}, nil
	}

	result := &LSResult{
		Path:      resolved.Given,
		Absolute:  resolved.Absolute,
		Hidden:    cfg.All || cfg.AlmostAll,
		Recursive: cfg.Recursive,
		Timestamp: meta.Now(),
	}

	if err := populateDir(result, resolved.Absolute, cfg); err != nil {
		return nil, err
	}

	return result, nil
}

type fsEntry struct {
	name string
	path string
	info fs.FileInfo
}

func populateDir(result *LSResult, dirPath string, cfg Config) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		result.Errors = append(result.Errors, LSError{
			Code: 1,
			Msg:  err.Error(),
			Path: dirPath,
		})
		return nil
	}

	var fileInfos []fsEntry
	for _, de := range entries {
		name := de.Name()
		if !cfg.All && !cfg.AlmostAll && strings.HasPrefix(name, ".") {
			continue
		}
		if cfg.AlmostAll && (name == "." || name == "..") {
			continue
		}

		fullPath := filepath.Join(dirPath, name)
		info, err := os.Lstat(fullPath)
		if err != nil {
			result.Errors = append(result.Errors, LSError{
				Code: 1,
				Msg:  err.Error(),
				Path: fullPath,
			})
			continue
		}

		fileInfos = append(fileInfos, fsEntry{name: name, path: fullPath, info: info})
	}

	sortEntries(fileInfos, cfg)

	for _, fe := range fileInfos {
		entry, err := buildEntry(fe.path, fe.info, fe.name, cfg)
		if err != nil {
			result.Errors = append(result.Errors, LSError{
				Code: 1,
				Msg:  err.Error(),
				Path: fe.path,
			})
			continue
		}
		result.Entries = append(result.Entries, entry)
	}

	result.TotalEntries = len(result.Entries)

	if cfg.Recursive {
		for _, fe := range fileInfos {
			if fe.info.IsDir() {
				subResult := &LSResult{
					Path:      fe.name,
					Absolute:  fe.path,
					Hidden:    cfg.All || cfg.AlmostAll,
					Recursive: cfg.Recursive,
					Timestamp: meta.Now(),
				}
				if err := populateDir(subResult, fe.path, cfg); err != nil {
					return err
				}
				result.Entries = append(result.Entries, subResult)
				result.TotalEntries += subResult.TotalEntries
			}
		}
	}

	return nil
}

func sortEntries(entries []fsEntry, cfg Config) {
	if cfg.SortTime {
		sort.SliceStable(entries, func(i, j int) bool {
			ti := entries[i].info.ModTime().Unix()
			tj := entries[j].info.ModTime().Unix()
			if cfg.Reverse {
				return ti < tj
			}
			return ti > tj
		})
	} else {
		sort.SliceStable(entries, func(i, j int) bool {
			if cfg.Reverse {
				return entries[i].name > entries[j].name
			}
			return entries[i].name < entries[j].name
		})
	}
}

func buildEntry(fullPath string, info fs.FileInfo, name string, cfg Config) (LSItem, error) {
	mode := info.Mode()
	modTime := info.ModTime().Unix()
	perms := filemode.FormatPermissions(mode, info.IsDir(), filemode.IsSymlink(mode))
	modeStr := filemode.ModeOctal(mode)
	owner, group := lookupOwner(info)

	if mode&fs.ModeSymlink != 0 {
		target, targetAbs, targetExists, err := pathutil.ResolveSymlink(fullPath)
		if err != nil {
			target = ""
			targetAbs = ""
			targetExists = false
		}

		return SymlinkEntry{
			Name:           name,
			Path:           fullPath,
			Target:         target,
			TargetAbsolute: targetAbs,
			TargetExists:   strconv.FormatBool(targetExists),
			Modified:       modTime,
			ModifiedAgoS:   meta.AgoSeconds(modTime),
			Permissions:    perms,
			Mode:           modeStr,
		}, nil
	}

	if info.IsDir() {
		return DirEntry{
			Name:         name,
			Path:         fullPath,
			Modified:     modTime,
			ModifiedAgoS: meta.AgoSeconds(modTime),
			Permissions:  perms,
			Mode:         modeStr,
			Owner:        owner,
			Group:        group,
		}, nil
	}

	mime := "application/octet-stream"
	isBinary := true
	language := ""

	if size := info.Size(); size > 0 && !cfg.Plain {
		mime, isBinary, _ = detect.DetectFromFile(fullPath)
		if !isBinary {
			language = detect.LanguageFromFile(fullPath)
		}
	}

	sizeBytes := uint64(info.Size())
	if info.Size() < 0 {
		sizeBytes = 0
	}

	return FileEntry{
		Name:         name,
		Path:         fullPath,
		Absolute:     fullPath,
		SizeBytes:    sizeBytes,
		SizeHuman:    format.Size(sizeBytes),
		Modified:     modTime,
		ModifiedAgoS: meta.AgoSeconds(modTime),
		Permissions:  perms,
		Mode:         modeStr,
		Owner:        owner,
		Group:        group,
		Executable:   strconv.FormatBool(mode&0111 != 0),
		Symlink:      "false",
		MIME:         mime,
		Language:     language,
		Binary:       strconv.FormatBool(isBinary),
	}, nil
}

func lookupOwner(info fs.FileInfo) (owner, group string) {
	sysInfo := info.Sys()
	if sysInfo == nil {
		return "unknown", "unknown"
	}

	uid := filemode.UID(sysInfo)
	gid := filemode.GID(sysInfo)

	owner = "unknown"
	if u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10)); err == nil {
		owner = u.Username
	} else {
		owner = strconv.FormatUint(uint64(uid), 10)
	}

	group = "unknown"
	if g, err := user.LookupGroupId(strconv.FormatUint(uint64(gid), 10)); err == nil {
		group = g.Name
	} else {
		group = strconv.FormatUint(uint64(gid), 10)
	}

	return owner, group
}

func outputResult(result *LSResult, cfg Config) error {
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("ls")
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
	if cfg.NoCompact {
		if cfg.JSON {
			return xmlout.WriteJSON(os.Stdout, result)
		}
		return xmlout.WriteXMLNoCompact(os.Stdout, result, cfg.Pretty)
	}
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *LSResult) error {
	if len(result.Errors) > 0 && len(result.Entries) == 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "ls: %s: %s\n", e.Path, e.Msg)
		}
		return nil
	}

	for _, e := range result.Entries {
		switch entry := e.(type) {
		case DirEntry:
			line := fmt.Sprintf("%s %s %s %-10s %s",
				entry.Permissions,
				entry.Owner,
				entry.Group,
				meta.FormatTime(entry.Modified),
				entry.Name,
			)
			fmt.Fprintln(w, line)
		case FileEntry:
			line := fmt.Sprintf("%s %s %s %8s %s %s",
				entry.Permissions,
				entry.Owner,
				entry.Group,
				entry.SizeHuman,
				meta.FormatTime(entry.Modified),
				entry.Name,
			)
			fmt.Fprintln(w, line)
		case SymlinkEntry:
			target := entry.Target
			if entry.TargetExists != "true" {
				target += " -> [broken]"
			}
			line := fmt.Sprintf("%s %s %s -> %s",
				entry.Permissions,
				meta.FormatTime(entry.Modified),
				entry.Name,
				target,
			)
			fmt.Fprintln(w, line)
		case *LSResult:
			fmt.Fprintln(w)
			fmt.Fprintf(w, "%s:\n", entry.Absolute)
			writePlain(w, entry)
		}
	}

	return nil
}
