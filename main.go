package main

import (
	"fmt"
	"os"

	"github.com/synseqack/aict/internal/tool"
	_ "github.com/synseqack/aict/tools/cat"
	_ "github.com/synseqack/aict/tools/diff"
	_ "github.com/synseqack/aict/tools/find"
	_ "github.com/synseqack/aict/tools/grep"
	_ "github.com/synseqack/aict/tools/ls"
	_ "github.com/synseqack/aict/tools/stat"
	_ "github.com/synseqack/aict/tools/wc"
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
		printUsage()
		return nil
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

func printUsage() {
	fmt.Print(`aict - Your command line, built for AI

Usage: aict <command> [flags] [arguments]

Commands:
`)

	tools := tool.All()
	for name := range tools {
		fmt.Printf("  %s\n", name)
	}

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
`)
}
