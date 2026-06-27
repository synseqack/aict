package ps

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/synseqack/aict/internal/meta"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
	tool.Register("ps", Run)
	tool.RegisterMeta("ps", tool.GenerateSchema("ps", "List running processes with details", Config{}))
}

type Config struct {
	All    bool   `flag:"" desc:"Show all processes"`
	Full   bool   `flag:"" desc:"Show full command details"`
	PID    string `flag:"" desc:"Filter by PID"`
	SortBy string `flag:"" desc:"Sort by field (e.g., pid, cpu, mem)"`
	XML    bool
	JSON   bool
	Plain  bool
	Pretty bool
	Compact bool
}

type PsResult struct {
	XMLName   xml.Name  `xml:"ps"`
	Timestamp int64     `xml:"timestamp,attr"`
	Processes []Process `xml:"process,omitempty"`
	Errors    []PsError `xml:"error,omitempty"`
}

func (*PsResult) isPsResult() {}

type Process struct {
	XMLName   xml.Name `xml:"process"`
	PID       int      `xml:"pid,attr"`
	PPID      int      `xml:"ppid,attr"`
	User      string   `xml:"user,attr"`
	UID       string   `xml:"uid,attr"`
	State     string   `xml:"state,attr"`
	StateDesc string   `xml:"state_desc,attr"`
	CPUPct    float64  `xml:"cpu_pct,attr"`
	MemPct    float64  `xml:"mem_pct,attr"`
	VSZKB     int64    `xml:"vsz_kb,attr"`
	RSSKB     int64    `xml:"rss_kb,attr"`
	Started   string   `xml:"started,attr"`
	Command   string   `xml:"command,attr"`
	Args      string   `xml:"args,attr"`
	Exe       string   `xml:"exe,attr"`
}

type PsError struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Msg     string   `xml:"msg,attr"`
}

func Run(args []string) error {
	cfg, _ := parseFlags(args)

	result := &PsResult{
		Timestamp: meta.Now(),
	}

	processes, err := getProcesses(cfg)
	if err != nil {
		result.Errors = append(result.Errors, PsError{Code: 1, Msg: err.Error()})
	}

	result.Processes = processes

	return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
	var cfg Config

	var positional []string

	for _, arg := range args {
		switch arg {
		case "-a", "-A", "aux", "-ef":
			cfg.All = true
			cfg.Full = true
		case "-p":
		case "--sort":
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

func getProcesses(cfg Config) ([]Process, error) {
	procDir := "/proc"
	entries, err := os.ReadDir(procDir)
	if err != nil {
		return nil, err
	}

	var processes []Process

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		pid, err := strconv.Atoi(name)
		if err != nil {
			continue
		}

		proc, err := getProcessInfo(pid, cfg)
		if err != nil {
			continue
		}

		processes = append(processes, proc)
	}

	sort.Slice(processes, func(i, j int) bool {
		return processes[i].PID < processes[j].PID
	})

	return processes, nil
}

func getProcessInfo(pid int, cfg Config) (Process, error) {
	proc := Process{PID: pid}

	statPath := filepath.Join("/proc", strconv.Itoa(pid), "stat")
	data, err := os.ReadFile(statPath)
	if err != nil {
		return proc, err
	}

	content := string(data)

	openIdx := strings.Index(content, "(")
	closeIdx := strings.LastIndex(content, ")")
	if openIdx == -1 || closeIdx == -1 || closeIdx < openIdx {
		return proc, nil
	}

	afterComm := strings.TrimSpace(content[closeIdx+1:])
	fields := strings.Fields(afterComm)
	if len(fields) >= 21 {
		proc.State = fields[0]
		proc.StateDesc = getStateDescription(proc.State)
		proc.PPID, _ = strconv.Atoi(fields[1])
		vsz, _ := strconv.ParseInt(fields[20], 10, 64)
		proc.VSZKB = vsz / 1024
	}

	cmdlinePath := filepath.Join("/proc", strconv.Itoa(pid), "cmdline")
	cmdline, err := os.ReadFile(cmdlinePath)
	if err == nil {
		args := strings.Split(string(cmdline), "\x00")
		var filtered []string
		for _, a := range args {
			if a != "" {
				filtered = append(filtered, a)
			}
		}
		if len(filtered) > 0 {
			proc.Command = filepath.Base(filtered[0])
			proc.Args = strings.Join(filtered, " ")
		}
	}

	statusPath := filepath.Join("/proc", strconv.Itoa(pid), "status")
	statusData, err := os.ReadFile(statusPath)
	if err == nil {
		for _, line := range strings.Split(string(statusData), "\n") {
			if strings.HasPrefix(line, "Uid:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					proc.UID = fields[1]
					proc.User = getUsername(fields[1])
				}
			} else if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					rss, _ := strconv.ParseInt(fields[1], 10, 64)
					proc.RSSKB = rss
				}
			}
		}
	}

	exePath := filepath.Join("/proc", strconv.Itoa(pid), "exe")
	exe, err := os.Readlink(exePath)
	if err == nil {
		proc.Exe = exe
	}

	return proc, nil
}

func getStateDescription(state string) string {
	switch state {
	case "R":
		return "running"
	case "S":
		return "sleeping"
	case "D":
		return "disk sleep"
	case "Z":
		return "zombie"
	case "T":
		return "stopped"
	case "t":
		return "tracing stop"
	case "X":
		return "dead"
	case "I", "i":
		return "idle"
	default:
		return "unknown"
	}
}

func getUsername(uid string) string {
	u, err := user.LookupId(uid)
	if err != nil {
		return uid
	}
	return u.Username
}

func outputResult(result *PsResult, cfg Config) error {
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

func writePlain(w io.Writer, result *PsResult) error {
	fmt.Fprintf(w, "%-10s %-6s %-10s %s %s %s %s\n", "USER", "PID", "CPU", "MEM", "VSZ", "RSS", "COMMAND")
	for _, p := range result.Processes {
		fmt.Fprintf(w, "%-10s %-6d %-10s %s %6d %6d %s\n",
			p.User, p.PID, "-", "-", p.VSZKB, p.RSSKB, p.Command)
	}
	return nil
}
