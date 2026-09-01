package tar

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/synseqack/aict/internal/format"
	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("tar", Run)
	tool.RegisterMeta("tar", tool.GenerateSchema("tar", "List or extract contents from tar archives (.tar, .tar.gz, .tgz, .tar.bz2)", Config{}))
	xmlout.RegisterDict("tar", map[string]string{
		"ab": "absolute",
		"sb": "size_bytes",
		"sh": "size_human",
		"m": "modified",
		"lt": "link_target",
	})
}

type Config struct {
	List      bool   `flag:"" desc:"List archive contents"`
	Extract   string `flag:"" desc:"Extract a specific file path from the archive to stdout"`
	XML       bool
	JSON      bool
	Plain     bool
	Pretty    bool
	Dict      bool
	NoCompact bool
}

type TarResult struct {
	XMLName   xml.Name   `xml:"tar" json:"-"`
	Archive   string     `xml:"archive,attr" json:"ar"`
	Absolute  string     `xml:"absolute,attr" json:"ab"`
	Format    string     `xml:"format,attr" json:"f"`
	Entries   int        `xml:"entries,attr" json:"e"`
	Timestamp int64      `xml:"timestamp,attr" json:"t"`
	Files     []TarFile  `xml:"file,omitempty" json:"files,omitempty"`
	Errors    []TarError `xml:"error,omitempty" json:"errors,omitempty"`
}

func (*TarResult) isTarResult() {}

type TarFile struct {
	XMLName   xml.Name `xml:"file" json:"-"`
	Path      string   `xml:"path,attr" json:"p"`
	SizeBytes int64    `xml:"size_bytes,attr" json:"sb"`
	SizeHuman string   `xml:"size_human,attr" json:"sh"`
	Modified  string   `xml:"modified,attr" json:"m"`
	Type      string   `xml:"type,attr" json:"tp"`
	Mode      string   `xml:"mode,attr" json:"mo"`
	LinkTarget string  `xml:"link_target,attr,omitempty" json:"lt,omitempty"`
}

type TarError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"m"`
	Path    string   `xml:"path,attr" json:"p"`
}

func Run(args []string) error {
	cfg, paths := parseFlags(args)

	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("tar")
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

	if len(paths) == 0 {
		return outputResult(&TarResult{
			Timestamp: meta.Now(),
			Errors:    []TarError{{Code: 1, Msg: "no archive specified"}},
		}, cfg)
	}

	archivePath := paths[0]
	resolved, err := pathutil.Resolve(archivePath)
	if err != nil {
		return outputResult(&TarResult{
			Archive:   archivePath,
			Timestamp: meta.Now(),
			Errors:    []TarError{{Code: 1, Msg: err.Error(), Path: archivePath}},
		}, cfg)
	}

	f, err := os.Open(resolved.Absolute)
	if err != nil {
		return outputResult(&TarResult{
			Archive:   resolved.Given,
			Absolute:  resolved.Absolute,
			Timestamp: meta.Now(),
			Errors:    []TarError{{Code: 1, Msg: err.Error(), Path: resolved.Absolute}},
		}, cfg)
	}
	defer f.Close()

	tr, archiveFmt, err := openArchive(f, resolved.Absolute)
	if err != nil {
		return outputResult(&TarResult{
			Archive:   resolved.Given,
			Absolute:  resolved.Absolute,
			Timestamp: meta.Now(),
			Errors:    []TarError{{Code: 1, Msg: err.Error(), Path: resolved.Absolute}},
		}, cfg)
	}

	result := &TarResult{
		Archive:   resolved.Given,
		Absolute:  resolved.Absolute,
		Format:    archiveFmt,
		Timestamp: meta.Now(),
	}

	if cfg.Extract != "" {
		return extractFile(tr, cfg.Extract)
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, TarError{Code: 1, Msg: err.Error()})
			break
		}

		tf := TarFile{
			Path:      hdr.Name,
			SizeBytes: hdr.Size,
			SizeHuman: format.Size(uint64(hdr.Size)),
			Modified:  hdr.ModTime.Format("2006-01-02 15:04:05"),
			Mode:      hdr.FileInfo().Mode().String(),
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			tf.Type = "directory"
		case tar.TypeSymlink:
			tf.Type = "symlink"
			tf.LinkTarget = hdr.Linkname
		case tar.TypeLink:
			tf.Type = "hardlink"
			tf.LinkTarget = hdr.Linkname
		default:
			tf.Type = "file"
		}

		result.Files = append(result.Files, tf)
	}

	result.Entries = len(result.Files)
	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-t", "--list":
			cfg.List = true
		case "-x", "--extract":
			if i+1 < len(args) {
				cfg.Extract = args[i+1]
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
		case "--dict":
			cfg.Dict = true
		case "--no-compact":
			cfg.NoCompact = true
		default:
			if !strings.HasPrefix(arg, "-") {
				positional = append(positional, arg)
			}
		}
	}

	if !cfg.XML && !cfg.JSON && !cfg.Plain {
		cfg.XML = xmlout.IsXMLMode()
	}

	return cfg, positional
}

func openArchive(f *os.File, path string) (*tar.Reader, string, error) {
	lower := strings.ToLower(path)

	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, "", fmt.Errorf("gzip: %w", err)
		}
		return tar.NewReader(gz), "tar.gz", nil
	}

	if strings.HasSuffix(lower, ".tar.bz2") {
		bz := bzip2.NewReader(f)
		return tar.NewReader(bz), "tar.bz2", nil
	}

	if strings.HasSuffix(lower, ".tar") {
		return tar.NewReader(f), "tar", nil
	}

	// Try to detect by magic bytes
	header := make([]byte, 512)
	n, _ := f.Read(header)
	f.Seek(0, 0)

	if n >= 2 && header[0] == 0x1f && header[1] == 0x8b {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, "", fmt.Errorf("gzip: %w", err)
		}
		return tar.NewReader(gz), "tar.gz", nil
	}

	if n >= 3 && header[0] == 'B' && header[1] == 'Z' && header[2] == 'h' {
		bz := bzip2.NewReader(f)
		return tar.NewReader(bz), "tar.bz2", nil
	}

	// Try raw tar
	name := filepath.Base(path)
	if !strings.HasSuffix(strings.ToLower(name), ".tar") {
		return nil, "", fmt.Errorf("unsupported archive format: %s", name)
	}
	return tar.NewReader(f), "tar", nil
}

func extractFile(tr *tar.Reader, target string) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("file not found in archive: %s", target)
		}
		if err != nil {
			return err
		}
		if hdr.Name == target || strings.TrimPrefix(hdr.Name, "./") == strings.TrimPrefix(target, "./") {
			_, err := io.Copy(os.Stdout, tr)
			return err
		}
	}
}

func outputResult(result *TarResult, cfg Config) error {
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *TarResult) error {
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			if e.Path != "" {
				fmt.Fprintf(w, "tar: %s: %s\n", e.Path, e.Msg)
			} else {
				fmt.Fprintf(w, "tar: %s\n", e.Msg)
			}
		}
		return nil
	}
	for _, f := range result.Files {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", f.Mode, f.SizeHuman, f.Modified, f.Path)
	}
	return nil
}
