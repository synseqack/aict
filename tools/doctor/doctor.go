package doctor

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/synseqack/aict/internal/meta"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("doctor", Run)
	tool.RegisterMeta("doctor", tool.GenerateSchema("doctor", "Run diagnostics to check aict installation and environment", Config{}))
}

type Config struct {
	XML    bool
	JSON   bool
	Plain  bool
	Pretty bool
	Compact bool
}

type DoctorResult struct {
	XMLName   xml.Name      `xml:"doctor"`
	Timestamp int64         `xml:"timestamp,attr"`
	OS        string        `xml:"os,attr"`
	Arch      string        `xml:"arch,attr"`
	GoVersion string        `xml:"go_version,attr"`
	Checks    []Check       `xml:"check"`
	Summary   string        `xml:"summary,attr"`
	AllPassed bool          `xml:"all_passed,attr"`
	Errors    []DoctorError `xml:"error,omitempty"`
}

func (*DoctorResult) isDoctorResult() {}

type DoctorError struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Msg     string   `xml:"msg,attr"`
}

type Check struct {
	XMLName  xml.Name `xml:"check"`
	Name     string   `xml:"name,attr"`
	Status   string   `xml:"status,attr"`
	Message  string   `xml:"message,attr,omitempty"`
	Severity string   `xml:"severity,attr"`
}

func Run(args []string) error {
	cfg, _ := parseFlags(args)

	result := runDiagnostics()

	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config

	for i := 0; i < len(args); i++ {
		arg := args[i]
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

func runDiagnostics() *DoctorResult {
	result := &DoctorResult{
		Timestamp: meta.Now(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		GoVersion: runtime.Version(),
	}

	checks := []Check{
		checkAICTPath(),
		checkGit(),
		checkGoEnvironment(),
		checkPlatform(),
		checkShellCompletions(),
	}

	result.Checks = checks

	passed := 0
	warnings := 0
	errors := 0

	for _, c := range checks {
		switch c.Status {
		case "pass":
			passed++
		case "warning":
			warnings++
		case "fail":
			errors++
		}
	}

	result.AllPassed = errors == 0
	result.Summary = fmt.Sprintf("%d passed, %d warnings, %d errors", passed, warnings, errors)

	return result
}

func checkAICTPath() Check {
	exePath, err := os.Executable()
	if err != nil {
		return Check{Name: "aict_path", Status: "fail", Message: "cannot determine executable path", Severity: "error"}
	}

	absPath, err := filepath.Abs(exePath)
	if err != nil {
		return Check{Name: "aict_path", Status: "fail", Message: err.Error(), Severity: "error"}
	}

	dir := filepath.Dir(absPath)
	inPath := false

	pathEnv := os.Getenv("PATH")
	paths := strings.Split(pathEnv, string(os.PathListSeparator))
	for _, p := range paths {
		if p == dir {
			inPath = true
			break
		}
	}

	if inPath {
		return Check{Name: "aict_path", Status: "pass", Message: "aict is in PATH", Severity: "info"}
	}
	return Check{Name: "aict_path", Status: "warning", Message: "aict is not in PATH - use 'go install' or add to PATH", Severity: "warning"}
}

func checkGit() Check {
	cmd := exec.Command("git", "version")
	if err := cmd.Run(); err != nil {
		return Check{Name: "git", Status: "fail", Message: "git not found - git subcommands will not work", Severity: "warning"}
	}

	cmd = exec.Command("git", "rev-parse", "--is-inside-work-tree")
	output, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(output)) != "true" {
		return Check{Name: "git", Status: "warning", Message: "not inside a git repository", Severity: "info"}
	}

	return Check{Name: "git", Status: "pass", Message: "git is available and this is a git repository", Severity: "info"}
}

func checkGoEnvironment() Check {
	cmd := exec.Command("go", "version")
	if err := cmd.Run(); err != nil {
		return Check{Name: "go", Status: "fail", Message: "go not found", Severity: "error"}
	}

	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		return Check{Name: "go", Status: "warning", Message: "GOPATH not set", Severity: "warning"}
	}

	return Check{Name: "go", Status: "pass", Message: "Go is available", Severity: "info"}
}

func checkPlatform() Check {
	os := runtime.GOOS

	switch os {
	case "linux":
		return Check{Name: "platform", Status: "pass", Message: "Linux - full support", Severity: "info"}
	case "darwin":
		return Check{Name: "platform", Status: "pass", Message: "macOS - most tools work, ps uses fallback", Severity: "info"}
	case "windows":
		return Check{Name: "platform", Status: "warning", Message: "Windows - subset support (ls, cat, stat, wc, find, diff work; ps, df, uname need adaptation)", Severity: "warning"}
	default:
		return Check{Name: "platform", Status: "warning", Message: fmt.Sprintf("unsupported platform: %s", os), Severity: "warning"}
	}
}

func checkShellCompletions() Check {
	exePath, err := os.Executable()
	if err != nil {
		return Check{Name: "shell_completions", Status: "warning", Message: "cannot determine executable path", Severity: "info"}
	}

	dir := filepath.Dir(exePath)
	completionDir := filepath.Join(dir, "completions")

	if _, err := os.Stat(completionDir); os.IsNotExist(err) {
		return Check{Name: "shell_completions", Status: "warning", Message: "completions directory not found", Severity: "info"}
	}

	return Check{Name: "shell_completions", Status: "pass", Message: "shell completions available", Severity: "info"}
}

func outputResult(result *DoctorResult, cfg Config) error {
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

func writePlain(w io.Writer, result *DoctorResult) error {
	fmt.Fprintf(w, "aict doctor - Diagnostic Report\n")
	fmt.Fprintf(w, "================================\n")
	fmt.Fprintf(w, "OS: %s\n", result.OS)
	fmt.Fprintf(w, "Arch: %s\n", result.Arch)
	fmt.Fprintf(w, "Go: %s\n", result.GoVersion)
	fmt.Fprintf(w, "\n")

	for _, c := range result.Checks {
		icon := "[+]"
		switch c.Status {
		case "pass":
			icon = "[✓]"
		case "warning":
			icon = "[!]"
		case "fail":
			icon = "[✗]"
		}
		fmt.Fprintf(w, "%s %s: %s\n", icon, c.Name, c.Message)
	}

	fmt.Fprintf(w, "\n%s\n", result.Summary)
	return nil
}
