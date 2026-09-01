package jq

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/synseqack/aict/internal/meta"
	pathutil "github.com/synseqack/aict/internal/path"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("jq", Run)
	tool.RegisterMeta("jq", tool.GenerateSchema("jq", "Extract values from JSON files using path expressions", Config{}))
	xmlout.RegisterDict("jq", map[string]string{
		"pa": "path",
		"i": "index",
		"tp": "type",
		"r": "raw",
	})
}

type Config struct {
	Path      string `flag:"" desc:"JSON path expression (e.g. .key, .[0], .[].field)"`
	Raw       bool   `flag:"" desc:"Output raw string values without quotes"`
	XML       bool
	JSON      bool
	Plain     bool
	Pretty    bool
	Dict      bool
	NoCompact bool
}

type JQResult struct {
	XMLName  xml.Name   `xml:"jq" json:"-"`
	Path     string     `xml:"path,attr" json:"pa"`
	Count    int        `xml:"count,attr" json:"c"`
	Timestamp int64     `xml:"timestamp,attr" json:"t"`
	Values   []JQValue  `xml:"value,omitempty" json:"values,omitempty"`
	Errors   []JQError  `xml:"error,omitempty" json:"errors,omitempty"`
}

func (*JQResult) isJQResult() {}

type JQValue struct {
	XMLName xml.Name `xml:"value" json:"-"`
	Index   int      `xml:"index,attr" json:"i"`
	Type    string   `xml:"type,attr" json:"tp"`
	Raw     string   `xml:"raw,attr" json:"r"`
}

type JQError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"m"`
	Path    string   `xml:"path,attr" json:"p"`
}

func Run(args []string) error {
	cfg, paths := parseFlags(args)

	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("jq")
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

	if cfg.Path == "" {
		cfg.Path = "."
	}

	result := &JQResult{
		Path:      cfg.Path,
		Timestamp: meta.Now(),
	}

	for _, p := range paths {
		resolved, err := pathutil.Resolve(p)
		if err != nil {
			result.Errors = append(result.Errors, JQError{Code: 1, Msg: err.Error(), Path: p})
			continue
		}

		f, err := os.Open(resolved.Absolute)
		if err != nil {
			result.Errors = append(result.Errors, JQError{Code: 1, Msg: err.Error(), Path: p})
			continue
		}

		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			result.Errors = append(result.Errors, JQError{Code: 1, Msg: err.Error(), Path: p})
			continue
		}

		var root interface{}
		if err := json.Unmarshal(data, &root); err != nil {
			result.Errors = append(result.Errors, JQError{Code: 1, Msg: "invalid JSON: " + err.Error(), Path: p})
			continue
		}

		values, err := evalPath(cfg.Path, root)
		if err != nil {
			result.Errors = append(result.Errors, JQError{Code: 1, Msg: err.Error(), Path: p})
			continue
		}

		for _, v := range values {
			encoded, _ := json.Marshal(v)
			result.Values = append(result.Values, JQValue{
				Index: len(result.Values),
				Type:  jsonType(v),
				Raw:   string(encoded),
			})
		}
	}

	result.Count = len(result.Values)
	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-r", "--raw":
			cfg.Raw = true
		case "-p", "--path":
			if i+1 < len(args) {
				cfg.Path = args[i+1]
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
			positional = append(positional, arg)
		}
	}

	if !cfg.XML && !cfg.JSON && !cfg.Plain {
		cfg.XML = xmlout.IsXMLMode()
	}

	// First positional arg is path expression if it starts with '.'
	var files []string
	for _, p := range positional {
		if cfg.Path == "" && strings.HasPrefix(p, ".") {
			cfg.Path = p
		} else {
			files = append(files, p)
		}
	}
	return cfg, files
}

type step struct {
	kind  string // "field", "index", "iterate"
	field string
	index int
}

func parsePath(path string) ([]step, error) {
	if path == "." || path == "" {
		return nil, nil
	}

	if !strings.HasPrefix(path, ".") {
		return nil, fmt.Errorf("path must start with '.'")
	}

	path = path[1:] // strip leading '.'
	var steps []step

	for path != "" {
		if strings.HasPrefix(path, "[]") {
			steps = append(steps, step{kind: "iterate"})
			path = path[2:]
			if strings.HasPrefix(path, ".") {
				path = path[1:]
			}
			continue
		}
		if strings.HasPrefix(path, "[") {
			end := strings.Index(path, "]")
			if end < 0 {
				return nil, fmt.Errorf("unclosed '[' in path")
			}
			inner := path[1:end]
			path = path[end+1:]
			if strings.HasPrefix(path, ".") {
				path = path[1:]
			}
			n, err := strconv.Atoi(inner)
			if err != nil {
				return nil, fmt.Errorf("invalid array index: %q", inner)
			}
			steps = append(steps, step{kind: "index", index: n})
			continue
		}
		// field name up to next '.' or '['
		i := strings.IndexAny(path, ".[")
		var field string
		if i < 0 {
			field = path
			path = ""
		} else {
			field = path[:i]
			if path[i] == '.' {
				path = path[i+1:]
			} else {
				path = path[i:]
			}
		}
		if field != "" {
			steps = append(steps, step{kind: "field", field: field})
		}
	}
	return steps, nil
}

func evalPath(path string, root interface{}) ([]interface{}, error) {
	steps, err := parsePath(path)
	if err != nil {
		return nil, err
	}

	current := []interface{}{root}

	for _, s := range steps {
		var next []interface{}
		for _, v := range current {
			results, err := applyStep(s, v)
			if err != nil {
				return nil, err
			}
			next = append(next, results...)
		}
		current = next
	}

	return current, nil
}

func applyStep(s step, v interface{}) ([]interface{}, error) {
	switch s.kind {
	case "field":
		m, ok := v.(map[string]interface{})
		if !ok {
			return []interface{}{nil}, nil
		}
		return []interface{}{m[s.field]}, nil
	case "index":
		arr, ok := v.([]interface{})
		if !ok {
			return nil, fmt.Errorf("not an array")
		}
		idx := s.index
		if idx < 0 {
			idx = len(arr) + idx
		}
		if idx < 0 || idx >= len(arr) {
			return []interface{}{nil}, nil
		}
		return []interface{}{arr[idx]}, nil
	case "iterate":
		arr, ok := v.([]interface{})
		if !ok {
			return nil, fmt.Errorf("not an array, cannot iterate")
		}
		return arr, nil
	}
	return nil, fmt.Errorf("unknown step kind: %s", s.kind)
}

func jsonType(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	}
	return "unknown"
}

func outputResult(result *JQResult, cfg Config) error {
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result, cfg)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *JQResult, cfg Config) error {
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "jq: %s: %s\n", e.Path, e.Msg)
		}
		return nil
	}
	for _, v := range result.Values {
		if cfg.Raw && v.Type == "string" {
			// strip JSON quotes
			var s string
			if err := json.Unmarshal([]byte(v.Raw), &s); err == nil {
				fmt.Fprintln(w, s)
				continue
			}
		}
		fmt.Fprintln(w, v.Raw)
	}
	return nil
}
