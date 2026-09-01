package stat

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"
	"time"

	"github.com/synseqack/aict/internal/detect"
	"github.com/synseqack/aict/internal/filemode"
	"github.com/synseqack/aict/internal/format"
	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("stat", Run)
	tool.RegisterMeta("stat", tool.GenerateSchema("stat", "Display detailed file metadata including timestamps, permissions, and ownership", Config{}))

	dict := map[string]string{
		"p":  "path",
		"a":  "absolute",
		"ino":"inode",
		"lnk":"links",
		"dev":"device",
		"per":"permissions",
		"mo": "mode_octal",
		"uid":"uid",
		"gid":"gid",
		"o":  "owner",
		"g":  "group",
		"s":  "size_bytes",
		"sh": "size_human",
		"at": "atime",
		"aat":"atime_ago_s",
		"mt": "mtime",
		"mat":"mtime_ago_s",
		"ct": "ctime",
		"cat":"ctime_ago_s",
		"b":  "birth",
		"ba": "birth_ago_s",
		"ty": "type",
		"mime":"mime",
		"lang":"language",
		"t":  "timestamp",
		"e":  "error",
		"c":  "code",
		"msg":"msg",
	}
	xmlout.RegisterDict("stat", dict)
}

type Config struct {
	FollowSymlinks bool `flag:"" desc:"Follow symlinks and show target file info"`
	XML            bool
	JSON           bool
	Plain          bool
	Pretty         bool
	NoCompact     bool
	Dict           bool
}

type StatResult struct {
	XMLName     xml.Name    `xml:"stat" json:"-"`
	Path        string      `xml:"path,attr" json:"p"`
	Absolute    string      `xml:"absolute,attr" json:"a"`
	Inode       uint64      `xml:"inode,attr" json:"ino"`
	Links       int         `xml:"links,attr" json:"lnk"`
	Device      uint64      `xml:"device,attr" json:"dev"`
	Permissions string      `xml:"permissions,attr" json:"per"`
	ModeOctal   string      `xml:"mode_octal,attr" json:"mo"`
	UID         uint32      `xml:"uid,attr" json:"uid"`
	GID         uint32      `xml:"gid,attr" json:"gid"`
	Owner       string      `xml:"owner,attr" json:"o"`
	Group       string      `xml:"group,attr" json:"g"`
	SizeBytes   int64       `xml:"size_bytes,attr" json:"s"`
	SizeHuman   string      `xml:"size_human,attr" json:"sh"`
	Atime       int64       `xml:"atime,attr" json:"at"`
	AtimeAgoS   int64       `xml:"atime_ago_s,attr" json:"aat"`
	Mtime       int64       `xml:"mtime,attr" json:"mt"`
	MtimeAgoS   int64       `xml:"mtime_ago_s,attr" json:"mat"`
	Ctime       int64       `xml:"ctime,attr" json:"ct"`
	CtimeAgoS   int64       `xml:"ctime_ago_s,attr" json:"cat"`
	Birth       int64       `xml:"birth,attr" json:"b"`
	BirthAgoS   int64       `xml:"birth_ago_s,attr" json:"ba"`
	Type        string      `xml:"type,attr" json:"ty"`
	MIME        string      `xml:"mime,attr" json:"mime"`
	Language    string      `xml:"language,attr" json:"lang"`
	Timestamp   int64       `xml:"timestamp,attr" json:"t"`
	Errors      []StatError `xml:"error,omitempty" json:"e"`
}

func (*StatResult) isStatResult() {}

type StatError struct {
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

	sysInfo := info.Sys()
	if sysInfo != nil {
		result.Inode = getIno(sysInfo)
		result.Links = int(getNlink(sysInfo))
		result.Device = getDev(sysInfo)
		result.UID = filemode.UID(sysInfo)
		result.GID = filemode.GID(sysInfo)

		if owner, err := user.LookupId(strconv.FormatUint(uint64(result.UID), 10)); err == nil {
			result.Owner = owner.Username
		} else {
			result.Owner = "unknown"
		}

		if group, err := user.LookupGroupId(strconv.FormatUint(uint64(result.GID), 10)); err == nil {
			result.Group = group.Name
		} else {
			result.Group = "unknown"
		}

		sec := getATimSec(sysInfo)
		if sec > 0 {
			result.Atime = sec
			result.AtimeAgoS = meta.AgoSeconds(sec)
		} else {
			result.Atime = info.ModTime().Unix()
			result.AtimeAgoS = meta.AgoSeconds(info.ModTime().Unix())
		}

		sec = getMTimSec(sysInfo)
		if sec > 0 {
			result.Mtime = sec
			result.MtimeAgoS = meta.AgoSeconds(sec)
		} else {
			result.Mtime = info.ModTime().Unix()
			result.MtimeAgoS = meta.AgoSeconds(info.ModTime().Unix())
		}

		sec = getCTimSec(sysInfo)
		if sec > 0 {
			result.Ctime = sec
			result.CtimeAgoS = meta.AgoSeconds(sec)
		} else {
			result.Ctime = info.ModTime().Unix()
			result.CtimeAgoS = meta.AgoSeconds(info.ModTime().Unix())
		}

		if birthSec := getBirthSec(resolved.Absolute); birthSec > 0 {
			result.Birth = birthSec
			result.BirthAgoS = meta.AgoSeconds(birthSec)
		}
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
	result.Permissions = filemode.FormatPermissions(info.Mode(), info.IsDir(), filemode.IsSymlink(info.Mode()))
	result.ModeOctal = filemode.ModeOctal(info.Mode())
	result.Type = filemode.FileType(info)

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

func outputResult(result *StatResult, cfg Config) error {
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("stat")
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

func getIno(sysInfo any) uint64 {
	switch v := sysInfo.(type) {
	case interface{ Ino() uint64 }:
		return v.Ino()
	case interface{ Ino() uint32 }:
		return uint64(v.Ino())
	default:
		return 0
	}
}

func getNlink(sysInfo any) uint64 {
	switch v := sysInfo.(type) {
	case interface{ Nlink() uint64 }:
		return v.Nlink()
	case interface{ Nlink() uint32 }:
		return uint64(v.Nlink())
	default:
		return 0
	}
}

func getDev(sysInfo any) uint64 {
	switch v := sysInfo.(type) {
	case interface{ Dev() uint64 }:
		return v.Dev()
	default:
		return 0
	}
}

func getATimSec(sysInfo any) int64 {
	switch v := sysInfo.(type) {
	case interface {
		Atim() interface{ Sec() int64 }
	}:
		return v.Atim().Sec()
	case interface {
		Atime() interface{ Sec() int64 }
	}:
		return v.Atime().Sec()
	default:
		return 0
	}
}

func getMTimSec(sysInfo any) int64 {
	switch v := sysInfo.(type) {
	case interface {
		Mtim() interface{ Sec() int64 }
	}:
		return v.Mtim().Sec()
	case interface {
		Mtime() interface{ Sec() int64 }
	}:
		return v.Mtime().Sec()
	default:
		return 0
	}
}

func getCTimSec(sysInfo any) int64 {
	switch v := sysInfo.(type) {
	case interface {
		Ctim() interface{ Sec() int64 }
	}:
		return v.Ctim().Sec()
	case interface {
		Ctime() interface{ Sec() int64 }
	}:
		return v.Ctime().Sec()
	default:
		return 0
	}
}
