package checksums

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"os"

	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("checksums", Run)
	tool.RegisterMeta("checksums", tool.GenerateSchema("checksums", "Calculate MD5, SHA1, and SHA256 checksums for files", Config{}))
	tool.Register("md5sum", RunMD5)
	tool.RegisterMeta("md5sum", tool.GenerateSchema("md5sum", "Calculate MD5 checksum for files", Config{}))
	tool.Register("sha256sum", RunSHA256)
	tool.RegisterMeta("sha256sum", tool.GenerateSchema("sha256sum", "Calculate SHA256 checksum for files", Config{}))
	tool.Register("sha1sum", RunSHA1)
	tool.RegisterMeta("sha1sum", tool.GenerateSchema("sha1sum", "Calculate SHA1 checksum for files", Config{}))
}

type Config struct {
	Algorithms []string `flag:"" desc:"Hash algorithm (md5, sha1, sha256)"`
	Verify     bool     `flag:"" desc:"Verify checksums from file"`
	XML        bool
	JSON       bool
	Plain      bool
	Pretty     bool
	Compact bool
}

type ChecksumResult struct {
	XMLName   xml.Name        `xml:"checksums"`
	Timestamp int64           `xml:"timestamp,attr"`
	Files     []ChecksumFile  `xml:"file,omitempty"`
	Errors    []ChecksumError `xml:"error,omitempty"`
}

func (*ChecksumResult) isChecksumResult() {}

type ChecksumFile struct {
	XMLName   xml.Name `xml:"file"`
	Path      string   `xml:"path,attr"`
	Absolute  string   `xml:"absolute,attr"`
	SizeBytes int64    `xml:"size_bytes,attr"`
	MD5       string   `xml:"md5,attr"`
	SHA1      string   `xml:"sha1,attr"`
	SHA256    string   `xml:"sha256,attr"`
}

type ChecksumError struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Msg     string   `xml:"msg,attr"`
	Path    string   `xml:"path,attr"`
}

func Run(args []string) error      { return runWithAlgos(args, []string{"md5", "sha1", "sha256"}) }
func RunMD5(args []string) error    { return runWithAlgos(args, []string{"md5"}) }
func RunSHA256(args []string) error { return runWithAlgos(args, []string{"sha256"}) }
func RunSHA1(args []string) error   { return runWithAlgos(args, []string{"sha1"}) }

func runWithAlgos(args []string, defaultAlgos []string) error {
	cfg, paths := parseFlags(args, defaultAlgos)

	if len(paths) == 0 {
		return outputResult(&ChecksumResult{Timestamp: meta.Now()}, cfg)
	}

	result := &ChecksumResult{
		Timestamp: meta.Now(),
	}

	for _, path := range paths {
		checksum, err := calculateChecksums(path, cfg)
		if err != nil {
			result.Errors = append(result.Errors, ChecksumError{Code: 1, Msg: err.Error(), Path: path})
			continue
		}
		result.Files = append(result.Files, checksum)
	}

	return outputResult(result, cfg)
}

func parseFlags(args []string, defaultAlgos []string) (Config, []string) {
	var cfg Config
	cfg.Algorithms = defaultAlgos

	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-a", "--algorithm":
			if i+1 < len(args) {
				cfg.Algorithms = []string{args[i+1]}
				i++
			}
		case "-c", "--check":
			cfg.Verify = true
		case "--xml", "-xml":
			cfg.XML = true
		case "--json", "-json":
			cfg.JSON = true
		case "--plain", "-plain":
			cfg.Plain = true
		case "--pretty", "-pretty":
			cfg.Pretty = true
		case "--compact", "-compact":
			cfg.Compact = true
		default:
			positional = append(positional, arg)
		}
	}

	if !cfg.XML && !cfg.JSON && !cfg.Plain {
		cfg.XML = xmlout.IsXMLMode()
	}

	return cfg, positional
}

func calculateChecksums(path string, cfg Config) (ChecksumFile, error) {
	result := ChecksumFile{}

	resolved, err := pathutil.Resolve(path)
	if err != nil {
		return result, err
	}

	result.Path = resolved.Given
	result.Absolute = resolved.Absolute

	info, err := os.Lstat(resolved.Absolute)
	if err != nil {
		return result, err
	}

	result.SizeBytes = info.Size()

	if info.IsDir() {
		return result, fmt.Errorf("is a directory")
	}

	f, err := os.Open(resolved.Absolute)
	if err != nil {
		return result, err
	}
	defer f.Close()

	md5h := md5.New()
	sha1h := sha1.New()
	sha256h := sha256.New()

	reader := bufio.NewReader(f)

	buffer := make([]byte, 32*1024)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			md5h.Write(buffer[:n])
			sha1h.Write(buffer[:n])
			sha256h.Write(buffer[:n])
		}
		if err != nil {
			break
		}
	}

	for _, algo := range cfg.Algorithms {
		switch algo {
		case "md5":
			result.MD5 = hex.EncodeToString(md5h.Sum(nil))
		case "sha1":
			result.SHA1 = hex.EncodeToString(sha1h.Sum(nil))
		case "sha256":
			result.SHA256 = hex.EncodeToString(sha256h.Sum(nil))
		}
	}

	return result, nil
}

func outputResult(result *ChecksumResult, cfg Config) error {
	if cfg.JSON {
		if cfg.Compact {
		return xmlout.WriteJSONCompact(os.Stdout, result)
	}
	return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result, cfg)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *ChecksumResult, cfg Config) error {
	for _, f := range result.Files {
		hash := f.MD5
		if len(cfg.Algorithms) == 1 {
			switch cfg.Algorithms[0] {
			case "md5":
				hash = f.MD5
			case "sha1":
				hash = f.SHA1
			case "sha256":
				hash = f.SHA256
			}
		} else {
			hash = f.SHA256
		}
		fmt.Fprintf(w, "%s  %s\n", hash, f.Path)
	}
	return nil
}
