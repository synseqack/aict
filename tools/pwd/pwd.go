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
}

type Config struct {
	XML    bool
	JSON   bool
	Plain  bool
	Pretty bool
	Compact bool
}

type PwdResult struct {
	XMLName        xml.Name   `xml:"pwd"`
	Path           string     `xml:"path,attr"`
	Absolute       string     `xml:"absolute,attr"`
	Home           string     `xml:"home,attr"`
	RelativeToHome string     `xml:"relative_to_home,attr"`
	Timestamp      int64      `xml:"timestamp,attr"`
	Errors         []PwdError `xml:"error,omitempty"`
}

func (*PwdResult) isPwdResult() {}

type PwdError struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Msg     string   `xml:"msg,attr"`
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
		case "--compact", "-compact":
			cfg.Compact = true
		}
	}

	if !cfg.XML && !cfg.JSON && !cfg.Plain {
		cfg.XML = xmlout.IsXMLMode()
	}

	return cfg, nil
}

func outputResult(result *PwdResult, cfg Config) error {
	if cfg.JSON {
		if cfg.Compact {
		return xmlout.WriteJSONCompact(os.Stdout, result)
	}
	return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result)
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
