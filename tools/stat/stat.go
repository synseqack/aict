package stat

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/synseqack/aict/internal/detect"
	"github.com/synseqack/aict/internal/format"
	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("stat", Run)
}

type Config struct {
	FollowSymlinks bool
	XML            bool
	JSON           bool
	Plain          bool
	Pretty         bool
}

type StatResult struct {
	XMLName     xml.Name    `xml:"stat"`
	Path        string      `xml:"path,attr"`
	Absolute    string      `xml:"absolute,attr"`
	Inode       uint64      `xml:"inode,attr"`
	Links       int         `xml:"links,attr"`
	Device      uint64      `xml:"device,attr"`
	Permissions string      `xml:"permissions,attr"`
	ModeOctal   string      `xml:"mode_octal,attr"`
	UID         uint32      `xml:"uid,attr"`
	GID         uint32      `xml:"gid,attr"`
	Owner       string      `xml:"owner,attr"`
	Group       string      `xml:"group,attr"`
	SizeBytes   int64       `xml:"size_bytes,attr"`
	SizeHuman   string      `xml:"size_human,attr"`
	Atime       int64       `xml:"atime,attr"`
	AtimeAgoS   int64       `xml:"atime_ago_s,attr"`
	Mtime       int64       `xml:"mtime,attr"`
	MtimeAgoS   int64       `xml:"mtime_ago_s,attr"`
	Ctime       int64       `xml:"ctime,attr"`
	CtimeAgoS   int64       `xml:"ctime_ago_s,attr"`
	Birth       int64       `xml:"birth,attr"`
	BirthAgoS   int64       `xml:"birth_ago_s,attr"`
	Type        string      `xml:"type,attr"`
	MIME        string      `xml:"mime,attr"`
	Language    string      `xml:"language,attr"`
	Timestamp   int64       `xml:"timestamp,attr"`
	Errors      []StatError `xml:"error,omitempty"`
}

func (*StatResult) isStatResult() {}

type StatError struct {
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

	for i, p := range paths {
		result, err := statPath(p, cfg)
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
		case "-L", "--dereference":
			cfg.FollowSymlinks = true
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

func statPath(path string, cfg Config) (*StatResult, error) {
	resolved, err := pathutil.Resolve(path)
	if err != nil {
		return &StatResult{
			Path:      path,
			Timestamp: meta.Now(),
			Errors:    []StatError{{Code: 1, Msg: err.Error(), Path: path}},
		}, nil
	}

	var info os.FileInfo
	var errStat error

	if cfg.FollowSymlinks {
		info, errStat = os.Stat(resolved.Absolute)
	} else {
		info, errStat = os.Lstat(resolved.Absolute)
	}

	if errStat != nil {
		code := 1
		if os.IsNotExist(errStat) {
			code = 2
		}
		return &StatResult{
			Path:      resolved.Given,
			Absolute:  resolved.Absolute,
			Timestamp: meta.Now(),
			Errors:    []StatError{{Code: code, Msg: "no such file or directory", Path: resolved.Absolute}},
		}, nil
	}

	result := &StatResult{
		Path:      resolved.Given,
		Absolute:  resolved.Absolute,
		Timestamp: meta.Now(),
	}

	sys, ok := info.Sys().(*syscall.Stat_t)
	if ok {
		result.Inode = sys.Ino
		result.Links = int(sys.Nlink)
		result.Device = sys.Dev
		result.UID = sys.Uid
		result.GID = sys.Gid

		if owner, err := user.LookupId(strconv.FormatUint(uint64(sys.Uid), 10)); err == nil {
			result.Owner = owner.Username
		} else {
			result.Owner = "unknown"
		}

		if group, err := user.LookupGroupId(strconv.FormatUint(uint64(sys.Gid), 10)); err == nil {
			result.Group = group.Name
		} else {
			result.Group = "unknown"
		}

		result.Atime = sys.Atim.Sec
		result.AtimeAgoS = meta.AgoSeconds(sys.Atim.Sec)
		result.Mtime = sys.Mtim.Sec
		result.MtimeAgoS = meta.AgoSeconds(sys.Mtim.Sec)
		result.Ctime = sys.Ctim.Sec
		result.CtimeAgoS = meta.AgoSeconds(sys.Ctim.Sec)

		result.Birth = 0
		result.BirthAgoS = 0
	} else {
		result.Atime = info.ModTime().Unix()
		result.AtimeAgoS = meta.AgoSeconds(info.ModTime().Unix())
		result.Mtime = info.ModTime().Unix()
		result.MtimeAgoS = meta.AgoSeconds(info.ModTime().Unix())
		result.Ctime = info.ModTime().Unix()
		result.CtimeAgoS = meta.AgoSeconds(info.ModTime().Unix())
		result.Birth = 0
		result.BirthAgoS = 0
	}

	result.SizeBytes = info.Size()
	result.SizeHuman = format.Size(uint64(result.SizeBytes))
	result.Permissions = formatPermissions(info.Mode())
	result.ModeOctal = "0" + strconv.FormatUint(uint64(info.Mode().Perm()), 8)
	result.Type = getFileType(info)

	mime := "application/octet-stream"
	language := ""
	if !info.IsDir() {
		mime, _, _ = detect.DetectFromFile(resolved.Absolute)
		language = detect.LanguageFromFile(resolved.Absolute)
	}
	result.MIME = mime
	result.Language = language

	return result, nil
}

func formatPermissions(mode os.FileMode) string {
	var b strings.Builder
	b.Grow(10)

	if mode&os.ModeSymlink != 0 {
		b.WriteByte('l')
	} else if mode.IsDir() {
		b.WriteByte('d')
	} else {
		b.WriteByte('-')
	}

	for i := 8; i >= 0; i-- {
		bit := uint(1) << uint(i)
		switch {
		case mode&os.FileMode(bit) != 0:
			switch i % 3 {
			case 0:
				b.WriteByte('x')
			case 1:
				b.WriteByte('w')
			case 2:
				b.WriteByte('r')
			}
		default:
			b.WriteByte('-')
		}
	}

	return b.String()
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

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%dB", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(size)/float64(div), "KMGTPE"[exp])
}

func outputResult(result *StatResult, cfg Config) error {
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *StatResult) error {
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "stat: %s: %s\n", e.Path, e.Msg)
		}
		return nil
	}

	fmt.Fprintf(w, "  File: %s\n", result.Path)
	fmt.Fprintf(w, "  Size: %d\t\tBlocks: %d\tIO Block: %d\t%s\n",
		result.SizeBytes, result.Links, result.Device, result.Type)
	fmt.Fprintf(w, "Device: %d\t\tInode: %d\tLinks: %d\n",
		result.Device, result.Inode, result.Links)
	fmt.Fprintf(w, "Access: %s (%s)\n", result.Permissions, result.ModeOctal)
	fmt.Fprintf(w, "Uid: %d\t(%s)\tGid: %d\t(%s)\n",
		result.UID, result.Owner, result.GID, result.Group)
	fmt.Fprintf(w, "Access: %s\n", time.Unix(result.Atime, 0).Format(time.RubyDate))
	fmt.Fprintf(w, "Modify: %s\n", time.Unix(result.Mtime, 0).Format(time.RubyDate))
	fmt.Fprintf(w, "Change: %s\n", time.Unix(result.Ctime, 0).Format(time.RubyDate))

	return nil
}
