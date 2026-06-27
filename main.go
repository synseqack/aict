package main

import (
	"fmt"
	"os"

	mcpserver "github.com/synseqack/aict/cmd/mcp"
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

	_ "github.com/synseqack/aict/tools/awk"
	_ "github.com/synseqack/aict/tools/completions"
	_ "github.com/synseqack/aict/tools/jq"
	_ "github.com/synseqack/aict/tools/sed"
	_ "github.com/synseqack/aict/tools/tar"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "aict: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	toolName := args[0]
	subArgs := args[1:]

	if toolName == "help" || toolName == "--help" || toolName == "-h" {
		if len(subArgs) > 0 {
			printToolHelp(subArgs[0])
		} else {
			printUsage()
		}
		return nil
	}

	if toolName == "mcp" {
		return mcpserver.Serve()
	}

	tools := tool.All()
	fn, ok := tools[toolName]
	if !ok {
		fmt.Fprintf(os.Stderr, "aict: unknown command: %s\n", toolName)
		fmt.Fprintf(os.Stderr, "Run 'aict help' for usage.\n")
		return fmt.Errorf("unknown command: %s", toolName)
	}

	return fn(subArgs)
}

func printToolHelp(name string) {
	allMeta := tool.AllMeta()
	m, ok := allMeta[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "aict: unknown command: %s\n", name)
		printUsage()
		return
	}

	fmt.Printf("aict %s — %s\n\n", name, m.Description)

	props, ok := m.InputSchema["properties"].(map[string]interface{})
	if !ok || len(props) == 0 {
		fmt.Println("No flags available.")
		return
	}

	fmt.Println("Flags:")
	for flag, raw := range props {
		info, _ := raw.(map[string]interface{})
		desc, _ := info["description"].(string)
		typ, _ := info["type"].(string)
		if typ == "" {
			typ = "string"
		}
		fmt.Printf("  --%-20s (%s) %s\n", flag, typ, desc)
	}
}

func printUsage() {
	meta := tool.AllMeta()
	tools := tool.All()

	fmt.Print(`aict - Your command line, built for AI

Usage: aict <command> [flags] [arguments]

Commands:
`)

	for name := range tools {
		if m, ok := meta[name]; ok {
			fmt.Printf("  %-12s %s\n", name, m.Description)
		} else {
			fmt.Printf("  %s\n", name)
		}
	}

	fmt.Printf("  %-12s %s\n", "mcp", "Start MCP server (stdio transport)")

	fmt.Print(`
Output modes:
  --xml         XML output (default if AICT_XML=1)
  --json        JSON output
  --plain       Plain text output

Examples:
  aict ls src/
  aict grep "func" . -r
  aict cat main.go
  aict find . -name "*.go"
  aict mcp
`)
}
