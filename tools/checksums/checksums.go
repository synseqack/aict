package checksums

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"

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
	xmlout.RegisterDict("checksums", map[string]string{
		"ab":   "absolute",
		"sb":   "size_bytes",
		"m5":   "md5",
		"s1":   "sha1",
		"s2":   "sha256",
		"algo": "algorithm",
		"exp":  "expected",
		"act":  "actual",
		"st":   "status",
	})
}

type Config struct {
	Algorithms []string `flag:"" desc:"Hash algorithm (md5, sha1, sha256)"`
	Verify     bool     `flag:"" desc:"Read a manifest of HASH  PATH lines and check each entry"`
	XML        bool
	JSON       bool
	Plain      bool
	Pretty     bool
	Dict       bool
	NoCompact  bool
}

type ChecksumResult struct {
	XMLName   xml.Name        `xml:"checksums" json:"-"`
	Timestamp int64           `xml:"timestamp,attr" json:"t"`
	Files     []ChecksumFile  `xml:"file,omitempty" json:"files,omitempty"`
	Verified  []VerifiedFile  `xml:"verified,omitempty" json:"verified,omitempty"`
	Errors    []ChecksumError `xml:"error,omitempty" json:"errors,omitempty"`
}

func (*ChecksumResult) isChecksumResult() {}

type ChecksumFile struct {
	XMLName   xml.Name `xml:"file" json:"-"`
	Path      string   `xml:"path,attr" json:"p"`
	Absolute  string   `xml:"absolute,attr" json:"ab"`
	SizeBytes int64    `xml:"size_bytes,attr" json:"sb"`
	MD5       string   `xml:"md5,attr" json:"m5"`
	SHA1      string   `xml:"sha1,attr" json:"s1"`
	SHA256    string   `xml:"sha256,attr" json:"s2"`
}

// VerifiedFile is one entry from a -c/--check manifest after re-hashing the
// named file. Status is "ok" when the recomputed hash matches the manifest,
// "failed" otherwise; a path that could not be read becomes a ChecksumError.
type VerifiedFile struct {
	XMLName   xml.Name `xml:"verified" json:"-"`
	Path      string   `xml:"path,attr" json:"p"`
	Absolute  string   `xml:"absolute,attr" json:"ab"`
	Algorithm string   `xml:"algorithm,attr" json:"algo"`
	Expected  string   `xml:"expected,attr" json:"exp"`
	Actual    string   `xml:"actual,attr" json:"act"`
	Status    string   `xml:"status,attr" json:"st"`
}

type ChecksumError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"m"`
	Path    string   `xml:"path,attr" json:"p"`
}

func Run(args []string) error       { return runWithAlgos(args, []string{"md5", "sha1", "sha256"}) }
func RunMD5(args []string) error    { return runWithAlgos(args, []string{"md5"}) }
func RunSHA256(args []string) error { return runWithAlgos(args, []string{"sha256"}) }
func RunSHA1(args []string) error   { return runWithAlgos(args, []string{"sha1"}) }

func runWithAlgos(args []string, defaultAlgos []string) error {
	cfg, paths := parseFlags(args, defaultAlgos)

	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("checksums")
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
		return outputResult(&ChecksumResult{Timestamp: meta.Now()}, cfg)
	}

	result := &ChecksumResult{
		Timestamp: meta.Now(),
	}

	if cfg.Verify {
		for _, manifest := range paths {
			entries, errs := verifyManifest(manifest)
			result.Errors = append(result.Errors, errs...)
			result.Verified = append(result.Verified, entries...)
		}
		return outputResult(result, cfg)
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

// hashLength names the digest size a hex string implies, which is how a
// manifest written by md5sum, sha1sum or sha256sum is told apart without a
// separate field.
var hashLength = map[int]string{
	32: "md5",
	40: "sha1",
	64: "sha256",
}

// algorithmFor reports the algorithm a manifest line's digest implies, or an
// error when the digest is not hex of a known length.
func algorithmFor(digest string) (string, error) {
	algo, ok := hashLength[len(digest)]
	if !ok {
		return "", fmt.Errorf("no algorithm produces a %d-character digest", len(digest))
	}
	for _, c := range digest {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return "", fmt.Errorf("digest is not hexadecimal")
		}
	}
	return algo, nil
}

// hashWith re-hashes path with the named algorithm, returning the lowercase
// hex digest.
func hashWith(path, algorithm string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var h hash.Hash
	switch algorithm {
	case "md5":
		h = md5.New()
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	default:
		return "", fmt.Errorf("unknown algorithm %q", algorithm)
	}

	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// verifyManifest reads a manifest of "HASH  PATH" lines, GNU's format, and
// re-hashes each entry. The algorithm is inferred from the digest length, so
// a manifest produced by md5sum checks as md5 without a separate switch.
func verifyManifest(manifestPath string) ([]VerifiedFile, []ChecksumError) {
	f, err := os.Open(manifestPath)
	if err != nil {
		return nil, []ChecksumError{{Code: 1, Msg: err.Error(), Path: manifestPath}}
	}
	defer f.Close()

	var verified []VerifiedFile
	var errs []ChecksumError

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// GNU separates the digest from the path with two spaces, or one
		// space and a star in binary mode; any of them parse here.
		fields := strings.Fields(line)
		if len(fields) < 2 {
			errs = append(errs, ChecksumError{Code: 1, Msg: "malformed checksum line", Path: manifestPath})
			continue
		}

		expected := fields[0]
		givenPath := strings.TrimPrefix(strings.Join(fields[1:], " "), "*")

		algorithm, err := algorithmFor(expected)
		if err != nil {
			errs = append(errs, ChecksumError{Code: 1, Msg: err.Error(), Path: manifestPath})
			continue
		}

		entry := VerifiedFile{
			Path:      givenPath,
			Algorithm: algorithm,
			Expected:  expected,
		}

		resolved, err := pathutil.Resolve(givenPath)
		if err != nil {
			entry.Status = "failed"
			verified = append(verified, entry)
			continue
		}
		entry.Absolute = resolved.Absolute

		actual, err := hashWith(resolved.Absolute, algorithm)
		if err != nil {
			errs = append(errs, ChecksumError{Code: 1, Msg: err.Error(), Path: resolved.Absolute})
			continue
		}
		entry.Actual = actual
		if strings.EqualFold(actual, expected) {
			entry.Status = "ok"
		} else {
			entry.Status = "failed"
		}

		verified = append(verified, entry)
	}

	if err := scanner.Err(); err != nil {
		errs = append(errs, ChecksumError{Code: 1, Msg: err.Error(), Path: manifestPath})
	}

	return verified, errs
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
		return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result, cfg)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *ChecksumResult, cfg Config) error {
	if cfg.Verify {
		var failed, unreadable int
		for _, v := range result.Verified {
			if v.Status != "ok" {
				failed++
			}
			if _, err := fmt.Fprintf(w, "%s: %s\n", v.Path, strings.ToUpper(v.Status)); err != nil {
				return err
			}
		}
		for _, e := range result.Errors {
			unreadable++
			if _, err := fmt.Fprintf(w, "aict: %s: %s\n", e.Path, e.Msg); err != nil {
				return err
			}
		}
		if failed > 0 || unreadable > 0 {
			fmt.Fprintf(w, "WARNING: %d failed, %d could not be read\n", failed, unreadable)
		}
		return nil
	}

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
