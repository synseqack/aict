package completions

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/synseqack/aict/internal/meta"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("completions", Run)
	tool.RegisterMeta("completions", tool.GenerateSchema("completions", "Generate shell completion scripts for aict", Config{}))
	xmlout.RegisterDict("completions", map[string]string{
		"s": "shell",
		"sc": "script",
	})
}

type Config struct {
	Shell     string `flag:"" desc:"Shell type: bash, zsh, fish"`
	XML       bool
	JSON      bool
	Plain     bool
	Pretty    bool
	Dict      bool
	NoCompact bool
}

type CompletionsResult struct {
	XMLName   xml.Name `xml:"completions" json:"-"`
	Shell     string   `xml:"shell,attr" json:"s"`
	Script    string   `xml:"script,omitempty" json:"sc,omitempty"`
	Timestamp int64    `xml:"timestamp,attr" json:"t"`
	Errors    []CompletionsError `xml:"error,omitempty" json:"errors,omitempty"`
}

func (*CompletionsResult) isCompletionsResult() {}

type CompletionsError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"m"`
}

func Run(args []string) error {
	cfg, _ := parseFlags(args)

	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("completions")
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

	result := &CompletionsResult{
		Shell:     cfg.Shell,
		Timestamp: meta.Now(),
	}

	toolNames := toolNameList()

	var script string
	var err error

	switch cfg.Shell {
	case "", "bash":
		script = generateBash(toolNames)
	case "zsh":
		script = generateZsh(toolNames)
	case "fish":
		script = generateFish(toolNames)
	default:
		result.Errors = append(result.Errors, CompletionsError{
			Code: 1,
			Msg:  fmt.Sprintf("unsupported shell: %q (supported: bash, zsh, fish)", cfg.Shell),
		})
		return outputResult(result, cfg)
	}

	result.Script = script
	_ = err

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
		case "--dict":
			cfg.Dict = true
		case "--no-compact":
			cfg.NoCompact = true
		default:
			if !strings.HasPrefix(arg, "-") {
				cfg.Shell = arg
			} else {
				positional = append(positional, arg)
			}
		}
	}

	if !cfg.XML && !cfg.JSON && !cfg.Plain {
		cfg.Plain = true // completions default to plain (the script text)
	}

	return cfg, positional
}

func toolNameList() []string {
	all := tool.All()
	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func generateBash(names []string) string {
	var sb strings.Builder
	sb.WriteString("_aict() {\n")
	sb.WriteString("    local cur prev words\n")
	sb.WriteString("    cur=\"${COMP_WORDS[COMP_CWORD]}\"\n")
	sb.WriteString("    prev=\"${COMP_WORDS[COMP_CWORD-1]}\"\n")
	sb.WriteString("    words=\"")
	sb.WriteString(strings.Join(names, " "))
	sb.WriteString("\"\n\n")
	sb.WriteString("    if [[ ${COMP_CWORD} -eq 1 ]]; then\n")
	sb.WriteString("        COMPREPLY=($(compgen -W \"${words}\" -- \"${cur}\"))\n")
	sb.WriteString("        return\n")
	sb.WriteString("    fi\n\n")
	sb.WriteString("    COMPREPLY=($(compgen -f -- \"${cur}\"))\n")
	sb.WriteString("}\n\n")
	sb.WriteString("complete -F _aict aict\n")
	return sb.String()
}

func generateZsh(names []string) string {
	var sb strings.Builder
	sb.WriteString("#compdef aict\n\n")
	sb.WriteString("_aict() {\n")
	sb.WriteString("    local -a commands\n")
	sb.WriteString("    commands=(\n")
	for _, name := range names {
		sb.WriteString(fmt.Sprintf("        '%s'\n", name))
	}
	sb.WriteString("    )\n\n")
	sb.WriteString("    _arguments \\\n")
	sb.WriteString("        '1:command:->command'\n\n")
	sb.WriteString("    case $state in\n")
	sb.WriteString("        command)\n")
	sb.WriteString("            _describe 'command' commands\n")
	sb.WriteString("            ;;\n")
	sb.WriteString("    esac\n")
	sb.WriteString("}\n\n")
	sb.WriteString("_aict\n")
	return sb.String()
}

func generateFish(names []string) string {
	var sb strings.Builder
	for _, name := range names {
		sb.WriteString(fmt.Sprintf("complete -c aict -f -n '__fish_use_subcommand' -a '%s'\n", name))
	}
	return sb.String()
}

func outputResult(result *CompletionsResult, cfg Config) error {
	if cfg.JSON {
		return xmlout.WriteJSON(os.Stdout, result)
	}
	if cfg.Plain {
		return writePlain(os.Stdout, result)
	}
	return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}

func writePlain(w io.Writer, result *CompletionsResult) error {
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(w, "completions: %s\n", e.Msg)
		}
		return nil
	}
	_, err := io.WriteString(w, result.Script)
	return err
}
