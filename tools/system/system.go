package system

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/user"
	"runtime"
	"strings"

	"github.com/synseqack/aict/internal/meta"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("system", Run)
	tool.RegisterMeta("system", tool.GenerateSchema("system", "Display system information including user, OS, and runtime details", Config{}))
}

type Config struct {
	XML    bool
	JSON   bool
	Plain  bool
	Pretty bool
	Compact bool
}

type SystemResult struct {
	XMLName   xml.Name      `xml:"system"`
	Timestamp int64         `xml:"timestamp,attr"`
	User      UserInfo      `xml:"user"`
	OS        OSInfo        `xml:"os"`
	Runtime   RuntimeInfo   `xml:"runtime"`
	Errors    []SystemError `xml:"error,omitempty"`
}

func (*SystemResult) isSystemResult() {}

type UserInfo struct {
	XMLName  xml.Name `xml:"user"`
	Username string   `xml:"username,attr"`
	UID      string   `xml:"uid,attr"`
	GID      string   `xml:"gid,attr"`
	Home     string   `xml:"home,attr"`
	Shell    string   `xml:"shell,attr"`
	Groups   []string `xml:"group"`
}

type OSInfo struct {
	XMLName   xml.Name `xml:"os"`
	GOOS      string   `xml:"goos,attr"`
	GOARCH    string   `xml:"goarch,attr"`
	Hostname  string   `xml:"hostname,attr"`
	Kernel    string   `xml:"kernel,attr"`
	OSRelease string   `xml:"os_release,attr"`
	Distro    string   `xml:"distro,attr"`
}

type RuntimeInfo struct {
	XMLName      xml.Name `xml:"runtime"`
	Version      string   `xml:"version,attr"`
	NumCPU       int      `xml:"num_cpu,attr"`
	NumGoroutine int      `xml:"num_goroutine,attr"`
}

type SystemError struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Msg     string   `xml:"msg,attr"`
}

func Run(args []string) error {
	cfg, _ := parseFlags(args)

	result := &SystemResult{
		Timestamp: meta.Now(),
	}

	result.User = getUserInfo()

	result.OS = getOSInfo()

	result.Runtime = getRuntimeInfo()

	return outputResult(result, cfg)
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

func getUserInfo() UserInfo {
	info := UserInfo{}

	current, err := user.Current()
	if err != nil {
		return info
	}

	info.Username = current.Username
	info.UID = current.Uid
	info.GID = current.Gid
	info.Home = current.HomeDir
	info.Shell = os.Getenv("SHELL")
	if info.Shell == "" {
		info.Shell = "/bin/sh"
	}

	groups, err := current.GroupIds()
	if err == nil {
		info.Groups = groups
	}

	return info
}

func getOSInfo() OSInfo {
	info := OSInfo{}

	info.GOOS = runtime.GOOS
	info.GOARCH = runtime.GOARCH

	hostname, err := os.Hostname()
	if err == nil {
		info.Hostname = hostname
	}

	if runtime.GOOS == "linux" {
		info.Kernel = getKernelVersion()
		info.OSRelease = getOSRelease()
		info.Distro = parseDistro(info.OSRelease)
	} else if runtime.GOOS == "darwin" {
		info.Kernel = getDarwinVersion()
		info.Distro = "macOS"
	} else if runtime.GOOS == "windows" {
		info.Kernel = "Windows"
		info.Distro = "Windows"
	}

	return info
}

func getKernelVersion() string {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func getOSRelease() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	return string(data)
}

func parseDistro(osRelease string) string {
	for _, line := range strings.Split(osRelease, "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			value := strings.TrimPrefix(line, "PRETTY_NAME=")
			value = strings.Trim(value, `"`)
			return value
		}
	}
	return ""
}

func getDarwinVersion() string {
	data, err := os.ReadFile("/System/Library/CoreServices/SystemVersion.plist")
	if err != nil {
		return ""
	}
	return string(data)
}

func getRuntimeInfo() RuntimeInfo {
	info := RuntimeInfo{}

	info.Version = runtime.Version()
	info.NumCPU = runtime.NumCPU()
	info.NumGoroutine = runtime.NumGoroutine()

	return info
}

func outputResult(result *SystemResult, cfg Config) error {
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

func writePlain(w io.Writer, result *SystemResult) error {
	fmt.Fprintf(w, "User: %s (UID: %s, GID: %s)\n", result.User.Username, result.User.UID, result.User.GID)
	fmt.Fprintf(w, "Home: %s, Shell: %s\n", result.User.Home, result.User.Shell)
	if len(result.User.Groups) > 0 {
		fmt.Fprintf(w, "Groups: %s\n", strings.Join(result.User.Groups, ", "))
	}
	fmt.Fprintf(w, "OS: %s/%s\n", result.OS.GOOS, result.OS.GOARCH)
	fmt.Fprintf(w, "Hostname: %s\n", result.OS.Hostname)
	fmt.Fprintf(w, "Distro: %s\n", result.OS.Distro)
	fmt.Fprintf(w, "Kernel: %s\n", result.OS.Kernel)
	fmt.Fprintf(w, "Go: %s\n", result.Runtime.Version)
	fmt.Fprintf(w, "CPUs: %d, Goroutines: %d\n", result.Runtime.NumCPU, result.Runtime.NumGoroutine)

	return nil
}
