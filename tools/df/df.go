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
}

type Config struct {
	HumanSize bool `flag:"" desc:"Show sizes in human-readable format"`
	XML       bool
	JSON      bool
	Plain     bool
	Pretty    bool
}

type DfResult struct {
	XMLName     xml.Name  `xml:"df"`
	Timestamp   int64     `xml:"timestamp,attr"`
	Filesystems []FsEntry `xml:"filesystem,omitempty"`
	Errors      []DfError `xml:"error,omitempty"`
}

func (*DfResult) isDfResult() {}

type FsEntry struct {
	XMLName     xml.Name `xml:"filesystem"`
	Device      string   `xml:"device,attr"`
	Mount       string   `xml:"mount,attr"`
	Type        string   `xml:"type,attr"`
	SizeBytes   int64    `xml:"size_bytes,attr"`
	SizeHuman   string   `xml:"size_human,attr"`
	UsedBytes   int64    `xml:"used_bytes,attr"`
	UsedHuman   string   `xml:"used_human,attr"`
	AvailBytes  int64    `xml:"avail_bytes,attr"`
	AvailHuman  string   `xml:"avail_human,attr"`
	UsePct      int      `xml:"use_pct,attr"`
	InodesTotal int64    `xml:"inodes_total,attr"`
	InodesUsed  int64    `xml:"inodes_used,attr"`
	InodesAvail int64    `xml:"inodes_avail,attr"`
	InodesPct   int      `xml:"inodes_pct,attr"`
}

type DfError struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Msg     string   `xml:"msg,attr"`
	Path    string   `xml:"path,attr"`
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
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result)
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
