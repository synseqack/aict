package env

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/synseqack/aict/internal/meta"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("env", Run)
	tool.RegisterMeta("env", tool.GenerateSchema("env", "Display environment variables with types and redaction", Config{}))
}

var secretKeywords = []string{
	"KEY", "SECRET", "TOKEN", "PASSWORD", "DSN", "AUTH", "CREDENTIAL",
	"PRIVATE", "API", "ACCESS", "SIGNATURE", "CERT", "APIKEY", "API_KEY",
}

type Config struct {
	XML    bool
	JSON   bool
	Plain  bool
	Pretty bool
	Compact bool
}

type EnvResult struct {
	XMLName   xml.Name    `xml:"env"`
	Timestamp int64       `xml:"timestamp,attr"`
	Variables []EnvVar    `xml:"var,omitempty"`
	Path      []PathEntry `xml:"path_entry,omitempty"`
	Errors    []EnvError  `xml:"error,omitempty"`
}

func (*EnvResult) isEnvResult() {}

type EnvVar struct {
	XMLName    xml.Name `xml:"var"`
	Name       string   `xml:"name,attr"`
	Value      string   `xml:"value,attr"`
	Type       string   `xml:"type,attr"`
	Present    string   `xml:"present,attr"`
	Redacted   string   `xml:"redacted,attr"`
	PathExists string   `xml:"path_exists,attr,omitempty"`
}

type PathEntry struct {
	XMLName xml.Name `xml:"path_entry"`
	Index   int      `xml:"index,attr"`
	Path    string   `xml:"path,attr"`
	Exists  string   `xml:"exists,attr"`
}

type EnvError struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Msg     string   `xml:"msg,attr"`
}

func Run(args []string) error {
	cfg, _ := parseFlags(args)

	result := &EnvResult{
		Timestamp: meta.Now(),
	}

	envVars := os.Environ()
	for _, e := range envVars {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}

		name := parts[0]
		value := parts[1]

		isSecret := isSecret(name)
		varType := classifyType(name, value)
		pathExists := ""

		if name == "PATH" {
			entries := parsePath(value)
			for i, p := range entries {
				exists := "false"
				if _, err := os.Stat(p); err == nil {
					exists = "true"
				}
				result.Path = append(result.Path, PathEntry{
					Index:  i,
					Path:   p,
					Exists: exists,
				})
			}
		} else if strings.HasPrefix(name, "PATH") {
			entries := parsePath(value)
			for _, p := range entries {
				exists := "false"
				if _, err := os.Stat(p); err == nil {
					exists = "true"
				}
				pathExists = exists
			}
		}

		displayValue := value
		if isSecret {
			displayValue = "[REDACTED]"
		}
		result.Variables = append(result.Variables, EnvVar{
			Name:       name,
			Value:      displayValue,
			Type:       varType,
			Present:    "true",
			Redacted:   fmt.Sprintf("%t", isSecret),
			PathExists: pathExists,
		})
	}

	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config
	var positional []string

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
		default:
			positional = append(positional, arg)
		}
	}

	if !cfg.XML && !cfg.JSON && !cfg.Plain {
		cfg.XML = xmlout.IsXMLMode()
	}

	return cfg, positional
}

func isSecret(name string) bool {
	upper := strings.ToUpper(name)
	for _, kw := range secretKeywords {
		if strings.Contains(upper, kw) {
			return true
		}
	}
	return false
}

func classifyType(name, value string) string {
	nameLower := strings.ToLower(name)

	if strings.HasSuffix(nameLower, "_path") || nameLower == "path" {
		return "path"
	}
	if strings.HasSuffix(nameLower, "_paths") || strings.HasSuffix(nameLower, "_path_list") {
		return "path_list"
	}
	if nameLower == "home" || nameLower == "user" || nameLower == "username" {
		return "path"
	}

	if isSecret(name) {
		return "secret"
	}

	lower := strings.ToLower(value)
	if lower == "true" || lower == "false" {
		return "boolean"
	}

	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return "numeric"
	}

	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return "numeric"
	}

	if strings.Contains(value, "/") && !strings.Contains(value, " ") {
		if strings.Contains(value, "://") {
			return "url"
		}
		return "path"
	}

	return "string"
}

func parsePath(pathValue string) []string {
	var entries []string
	for _, p := range strings.Split(pathValue, string(os.PathListSeparator)) {
		if p != "" {
			abs, _ := filepath.Abs(p)
			entries = append(entries, abs)
		}
	}
	return entries
}

func outputResult(result *EnvResult, cfg Config) error {
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

func writePlain(w io.Writer, result *EnvResult) error {
	for _, v := range result.Variables {
		if v.Redacted == "true" {
			fmt.Fprintf(w, "%s=\n", v.Name)
		} else {
			fmt.Fprintf(w, "%s=%s\n", v.Name, v.Value)
		}
	}
	return nil
}
