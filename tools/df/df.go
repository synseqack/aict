package df

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"

	"github.com/synseqack/aict/internal/meta"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("df", Run)
	tool.RegisterMeta("df", tool.GenerateSchema("df", "Display disk filesystem usage statistics", Config{}))

	dict := map[string]string{
		"t":  "timestamp",
		"fss":"filesystems",
		"dev":"device",
		"mnt":"mount",
		"ty": "type",
		"s":  "size_bytes",
		"sh": "size_human",
		"ub": "used_bytes",
		"uh": "used_human",
		"ab": "avail_bytes",
		"ah": "avail_human",
		"up": "use_pct",
		"it": "inodes_total",
		"iu": "inodes_used",
		"ia": "inodes_avail",
		"ip": "inodes_pct",
		"e":  "error",
		"c":  "code",
		"msg":"msg",
	}
	xmlout.RegisterDict("df", dict)
}

type Config struct {
	HumanSize bool `flag:"" desc:"Show sizes in human-readable format"`
	XML       bool
	JSON      bool
	Plain     bool
	Pretty    bool
	NoCompact  bool
	Dict      bool
}

type DfResult struct {
	XMLName     xml.Name  `xml:"df" json:"-"`
	Timestamp   int64     `xml:"timestamp,attr" json:"t"`
	Filesystems []FsEntry `xml:"filesystem,omitempty" json:"fss"`
	Errors      []DfError `xml:"error,omitempty" json:"e"`
}

func (*DfResult) isDfResult() {}

type FsEntry struct {
	XMLName     xml.Name `xml:"filesystem" json:"-"`
	Device      string   `xml:"device,attr" json:"dev"`
	Mount       string   `xml:"mount,attr" json:"mnt"`
	Type        string   `xml:"type,attr" json:"ty"`
	SizeBytes   int64    `xml:"size_bytes,attr" json:"s"`
	SizeHuman   string   `xml:"size_human,attr" json:"sh"`
	UsedBytes   int64    `xml:"used_bytes,attr" json:"ub"`
	UsedHuman   string   `xml:"used_human,attr" json:"uh"`
	AvailBytes  int64    `xml:"avail_bytes,attr" json:"ab"`
	AvailHuman  string   `xml:"avail_human,attr" json:"ah"`
	UsePct      int      `xml:"use_pct,attr" json:"up"`
	InodesTotal int64    `xml:"inodes_total,attr" json:"it"`
	InodesUsed  int64    `xml:"inodes_used,attr" json:"iu"`
	InodesAvail int64    `xml:"inodes_avail,attr" json:"ia"`
	InodesPct   int      `xml:"inodes_pct,attr" json:"ip"`
}

type DfError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"msg"`
	Path    string   `xml:"path,attr" json:"p"`
}

func Run(args []string) error {
	cfg, _ := parseFlags(args)

	result, err := getFilesystems(cfg)
	if err != nil {
		return outputResult(&DfResult{
			Timestamp: meta.Now(),
			Errors:    []DfError{{Code: 1, Msg: err.Error()}},
		}, cfg)
	}

	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config

	var positional []string

	for _, arg := range args {
		switch arg {
		case "-h", "--human-readable":
			cfg.HumanSize = true
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

func getFilesystems(cfg Config) (*DfResult, error) {
	result := &DfResult{
		Timestamp: meta.Now(),
	}

	mounts, err := getMounts()
	if err != nil {
		result.Errors = append(result.Errors, DfError{Code: 1, Msg: err.Error()})
		return result, nil
	}

	for _, m := range mounts {
		fs, err := getFsInfo(m.Device, m.Mountpoint, m.Fstype)
		if err != nil {
			continue
		}
		result.Filesystems = append(result.Filesystems, fs)
	}

	return result, nil
}

type mountInfo struct {
	Device     string
	Mountpoint string
	Fstype     string
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitFields(s string) []string {
	var fields []string
	start := 0
	for i, c := range s {
		if c == ' ' {
			if start < i {
				fields = append(fields, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		fields = append(fields, s[start:])
	}
	return fields
}

func outputResult(result *DfResult, cfg Config) error {
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("df")
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

func writePlain(w io.Writer, result *DfResult) error {
	if len(result.Errors) > 0 && len(result.Filesystems) == 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "df: %s\n", e.Msg)
		}
		return nil
	}

	fmt.Fprintf(w, "Filesystem      Size  Used Avail Use%% Mounted on\n")
	for _, fs := range result.Filesystems {
		fmt.Fprintf(w, "%-13s %6s %6s %6s %3d%% %s\n",
			fs.Device, fs.SizeHuman, fs.UsedHuman, fs.AvailHuman, fs.UsePct, fs.Mount)
	}

	return nil
}
