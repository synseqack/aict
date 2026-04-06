package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/synseqack/aict/internal/tool"

	_ "github.com/synseqack/aict/tools/basename"
	_ "github.com/synseqack/aict/tools/cat"
	_ "github.com/synseqack/aict/tools/checksums"
	_ "github.com/synseqack/aict/tools/cut"
	_ "github.com/synseqack/aict/tools/df"
	_ "github.com/synseqack/aict/tools/diff"
	_ "github.com/synseqack/aict/tools/dirname"
	_ "github.com/synseqack/aict/tools/doctor"
	_ "github.com/synseqack/aict/tools/du"
	_ "github.com/synseqack/aict/tools/env"
	_ "github.com/synseqack/aict/tools/file"
	_ "github.com/synseqack/aict/tools/find"
	_ "github.com/synseqack/aict/tools/git"
	_ "github.com/synseqack/aict/tools/grep"
	_ "github.com/synseqack/aict/tools/head"
	_ "github.com/synseqack/aict/tools/ls"
	_ "github.com/synseqack/aict/tools/ps"
	_ "github.com/synseqack/aict/tools/pwd"
	_ "github.com/synseqack/aict/tools/realpath"
	_ "github.com/synseqack/aict/tools/sort"
	_ "github.com/synseqack/aict/tools/stat"
	_ "github.com/synseqack/aict/tools/system"
	_ "github.com/synseqack/aict/tools/tail"
	_ "github.com/synseqack/aict/tools/tr"
	_ "github.com/synseqack/aict/tools/uniq"
	_ "github.com/synseqack/aict/tools/wc"
)

var flagMappings = map[string]map[string]string{
	"ls": {
		"all":       "-a",
		"almostall": "-A",
		"sorttime":  "-t",
		"reverse":   "-r",
		"recursive": "-R",
		"pretty":    "--pretty",
		"compact":   "--compact",
		"help":      "-h",
	},
	"grep": {
		"recursive":        "-r",
		"linenumbers":      "-n",
		"fileswithmatches": "-l",
		"caseinsensitive":  "-i",
		"wordmatch":        "-w",
		"countonly":        "-c",
		"invertmatch":      "-v",
		"extendedregex":    "-E",
		"fixedstrings":     "-F",
		"include":          "--include",
		"excludedir":       "--exclude-dir",
		"maxcount":         "-m",
		"help":             "-h",
	},
	"cat": {
		"linenumbers": "-n",
		"help":        "-h",
	},
	"find": {
		"name":     "-name",
		"type":     "-type",
		"mtime":    "-mtime",
		"maxdepth": "-maxdepth",
		"invert":   "!",
		"or":       "-o",
		"help":     "-h",
	},
	"stat": {
		"help": "-h",
	},
	"wc": {
		"bytes":    "-c",
		"words":    "-w",
		"lines":    "-l",
		"maxlines": "-L",
		"allfiles": "-a",
		"help":     "-h",
	},
	"diff": {
		"brief": "--brief",
		"help":  "-h",
	},
	"head": {
		"lines":   "-n",
		"bytes":   "-c",
		"quiet":   "-q",
		"verbose": "-v",
		"help":    "-h",
	},
	"tail": {
		"lines":   "-n",
		"bytes":   "-c",
		"follow":  "-f",
		"quiet":   "-q",
		"verbose": "-v",
		"help":    "-h",
	},
	"du": {
		"all":       "-a",
		"summarize": "-s",
		"human":     "-h",
		"maxdepth":  "-max-depth",
		"help":      "-h",
	},
	"df": {
		"human":  "-h",
		"inodes": "-i",
		"help":   "-h",
	},
	"env": {
		"ignore": "-i",
		"vars":   "-u",
		"help":   "-h",
	},
	"ps": {
		"all":  "-a",
		"full": "-f",
		"help": "-h",
	},
	"system": {
		"help": "-h",
	},
	"tr": {
		"delete":  "-d",
		"squeeze": "-s",
		"help":    "-h",
	},
	"cut": {
		"bytes":   "-b",
		"chars":   "-c",
		"delimit": "-d",
		"fields":  "-f",
		"help":    "-h",
	},
	"uniq": {
		"count":     "-c",
		"duplicate": "-d",
		"unique":    "-u",
		"help":      "-h",
	},
	"sort": {
		"numeric": "-n",
		"reverse": "-r",
		"unique":  "-u",
		"help":    "-h",
	},
	"pwd": {
		"help": "-h",
	},
	"dirname": {
		"help": "-h",
	},
	"basename": {
		"help": "-h",
	},
	"realpath": {
		"help": "-h",
	},
	"file": {
		"mime": "--mime-type",
		"help": "-h",
	},
	"checksums": {
		"algorithm": "-a",
		"help":      "-h",
	},
	"doctor": {
		"verbose": "-v",
		"fix":     "--fix",
		"help":    "-h",
	},
	"git": {
		"help": "-h",
	},
}

func toBool(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	if s, ok := v.(string); ok {
		return s == "true" || s == "1"
	}
	return false
}

func boolPointer(b bool) *bool {
	return &b
}

func getString(args map[string]interface{}, key, defaultVal string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return defaultVal
}

func getInt(args map[string]interface{}, key string, defaultVal int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	return defaultVal
}

func buildArgs(toolName string, args map[string]interface{}) ([]string, error) {
	mappings, ok := flagMappings[toolName]
	if !ok {
		mappings = make(map[string]string)
	}

	var result []string

	for key, value := range args {
		if key == "" {
			continue
		}

		lowerKey := strings.ToLower(key)
		flag, hasFlag := mappings[lowerKey]
		if !hasFlag || flag == "" {
			continue
		}

		switch v := value.(type) {
		case bool:
			if v {
				result = append(result, flag)
			}
		case string:
			if v != "" {
				result = append(result, flag, v)
			}
		case float64:
			if v != 0 {
				result = append(result, flag, fmt.Sprintf("%d", int(v)))
			}
		}
	}

	for key, value := range args {
		lowerKey := strings.ToLower(key)
		if _, hasFlag := mappings[lowerKey]; hasFlag {
			continue
		}

		switch v := value.(type) {
		case string:
			if v != "" && lowerKey != "help" && lowerKey != "xml" && lowerKey != "json" && lowerKey != "plain" && lowerKey != "pretty" && lowerKey != "compact" {
				result = append(result, v)
			}
		}
	}

	return result, nil
}

func findAICTBinary() string {
	if binaryPath := os.Getenv("AICT_BINARY"); binaryPath != "" {
		return binaryPath
	}

	execPath, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(execPath)
		candidate := filepath.Join(dir, "aict")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	dir, err := os.Getwd()
	if err == nil {
		candidate := filepath.Join(dir, "aict")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return "aict"
}

func runAICT(args []string) (string, error) {
	binaryPath := findAICTBinary()

	aictArgs := append([]string{"--json"}, args...)

	cmd := exec.Command(binaryPath, aictArgs...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(exitErr.Stderr), fmt.Errorf("aict error: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("failed to run aict: %w", err)
	}

	return string(output), nil
}

func parseArgs(args any) map[string]interface{} {
	argsMap := make(map[string]interface{})
	if args == nil {
		return argsMap
	}
	data, err := json.Marshal(args)
	if err == nil {
		_ = json.Unmarshal(data, &argsMap)
	}
	return argsMap
}

func toolHandler(toolName string) func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap := parseArgs(req.Params.Arguments)

		if toBool(argsMap["help"]) {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("usage: aict %s [options]", toolName)},
				},
			}, nil
		}

		aictArgs, err := buildArgs(toolName, argsMap)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("error building args: %v", err)},
				},
			}, nil
		}

		output, err := runAICT(aictArgs)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{Text: err.Error()},
				},
			}, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: output},
			},
		}, nil
	}
}

func main() {
	tools := tool.AllMeta()

	log.Printf("aict MCP server starting with %d tools...", len(tools))

	for name, meta := range tools {
		schemaJSON, err := json.Marshal(meta.InputSchema)
		if err != nil {
			log.Printf("warning: failed to marshal schema for %s: %v", name, err)
			continue
		}

		log.Printf("  - %s: %s", name, meta.Description)
		_ = schemaJSON
	}

	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "aict",
			Version: "1.0.0",
		},
		nil,
	)

	for name, meta := range tools {
		schemaJSON, err := json.Marshal(meta.InputSchema)
		if err != nil {
			log.Printf("warning: failed to marshal schema for %s: %v", name, err)
			continue
		}

		var schemaMap map[string]interface{}
		if err := json.Unmarshal(schemaJSON, &schemaMap); err != nil {
			log.Printf("warning: failed to unmarshal schema for %s: %v", name, err)
			continue
		}

		server.AddTool(&mcp.Tool{
			Name:        name,
			Description: meta.Description,
			InputSchema: schemaMap,
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:    true,
				DestructiveHint: boolPointer(false),
			},
		}, toolHandler(name))
	}

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("MCP server error: %v", err)
	}
}
