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

	dict := map[string]string{
		"t":  "timestamp",
		"u":  "user",
		"un": "username",
		"uid":"uid",
		"gid":"gid",
		"h":  "home",
		"sh": "shell",
		"grp":"group",
		"os": "os",
		"goos":"goos",
		"arc":"goarch",
		"hn": "hostname",
		"kern":"kernel",
		"rel":"os_release",
		"dist":"distro",
		"r":  "runtime",
		"ver":"version",
		"ncpu":"num_cpu",
		"ngr": "num_goroutine",
		"e":  "error",
		"c":  "code",
		"msg":"msg",
	}
	xmlout.RegisterDict("system", dict)
}

type Config struct {
	XML    bool
	JSON   bool
	Plain  bool
	Pretty bool
	NoCompact bool
	Dict   bool
}

type SystemResult struct {
	XMLName   xml.Name      `xml:"system" json:"-"`
	Timestamp int64         `xml:"timestamp,attr" json:"t"`
	User      UserInfo      `xml:"user" json:"u"`
	OS        OSInfo        `xml:"os" json:"os"`
	Runtime   RuntimeInfo   `xml:"runtime" json:"r"`
	Errors    []SystemError `xml:"error,omitempty" json:"e"`
}

func (*SystemResult) isSystemResult() {}

type UserInfo struct {
	XMLName  xml.Name `xml:"user" json:"-"`
	Username string   `xml:"username,attr" json:"un"`
	UID      string   `xml:"uid,attr" json:"uid"`
	GID      string   `xml:"gid,attr" json:"gid"`
	Home     string   `xml:"home,attr" json:"h"`
	Shell    string   `xml:"shell,attr" json:"sh"`
	Groups   []string `xml:"group" json:"grp"`
}

type OSInfo struct {
	XMLName   xml.Name `xml:"os" json:"-"`
	GOOS      string   `xml:"goos,attr" json:"goos"`
	GOARCH    string   `xml:"goarch,attr" json:"arc"`
	Hostname  string   `xml:"hostname,attr" json:"hn"`
	Kernel    string   `xml:"kernel,attr" json:"kern"`
	OSRelease string   `xml:"os_release,attr" json:"rel"`
	Distro    string   `xml:"distro,attr" json:"dist"`
}

type RuntimeInfo struct {
	XMLName      xml.Name `xml:"runtime" json:"-"`
	Version      string   `xml:"version,attr" json:"ver"`
	NumCPU       int      `xml:"num_cpu,attr" json:"ncpu"`
	NumGoroutine int      `xml:"num_goroutine,attr" json:"ngr"`
}

type SystemError struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr" json:"c"`
	Msg     string   `xml:"msg,attr" json:"msg"`
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
	if cfg.Dict {
		dict := xmlout.GetRegisteredDict("system")
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
