package pwd

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/synseqack/aict/internal/meta"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("pwd", Run)
	tool.RegisterMeta("pwd", tool.GenerateSchema("pwd", "Print current working directory", Config{}))

	dict := map[string]string{
		"p":  "path",
		"a":  "absolute",
		"h":  "home",
		"rth":"relative_to_home",
		"t":  "timestamp",
		"e":  "error",
		"c":  "code",
		"msg":"msg",
	}
	xmlout.RegisterDict("pwd", dict)
}

type Config struct {
	XML    bool
	JSON   bool
	Plain  bool
	Pretty bool
	NoCompact bool
	Dict   bool
}

type PwdResult struct {
	XMLName        xml.Name   `xml:"pwd" json:"-"`
	Path           string     `xml:"path,attr" json:"p"`
	Absolute       string     `xml:"absolute,attr" json:"a"`
	Home           string     `xml:"home,attr" json:"h"`
	RelativeToHome string     `xml:"relative_to_home,attr" json:"rth"`
	Timestamp      int64      `xml:"timestamp,attr" json:"t"`
	Errors         []PwdError `xml:"error,omitempty" json:"e"`
}

func (*PwdResult) isPwdResult() {}

type PwdError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"msg"`
}

func Run(args []string) error {
	cfg, _ := parseFlags(args)

	cwd, err := os.Getwd()
	if err != nil {
		return outputResult(&PwdResult{
			Timestamp: meta.Now(),
			Errors:    []PwdError{{Code: 1, Msg: err.Error()}},
		}, cfg)
	}

	abs, _ := filepath.Abs(cwd)

	home := os.Getenv("HOME")
	relative := abs
	if home != "" && filepath.IsAbs(abs) {
		if filepath.Dir(abs) == home || strings.HasPrefix(abs, home+"/") {
			relative = "~" + strings.TrimPrefix(abs, home)
		}
	}

	return outputResult(&PwdResult{
		Path:           cwd,
		Absolute:       abs,
		Home:           home,
		RelativeToHome: relative,
		Timestamp:      meta.Now(),
	}, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config

	for _, arg := range args {
		switch arg {
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
		}
	}

	if !cfg.XML && !cfg.JSON && !cfg.Plain {
		cfg.XML = xmlout.IsXMLMode()
	}

	return cfg, nil
}

func outputResult(result *PwdResult, cfg Config) error {
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("pwd")
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

func writePlain(w io.Writer, result *PwdResult) error {
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "pwd: %s\n", e.Msg)
		}
		return nil
	}
	_, err := fmt.Fprintln(w, result.Path)
	return err
}
