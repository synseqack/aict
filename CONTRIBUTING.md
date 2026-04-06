# Contributing to aict

Thank you for your interest in contributing to aict.

aict is a CLI tool that outputs XML/JSON, built for AI agents to consume directly. The goal is to replace human-readable CLI output with structured data that AI agents can parse without guessing field positions or parsing formats.

## Project Structure

```
aict/
├── main.go               # Entry point
├── cmd/mcp/              # MCP server implementation
├── internal/             # Shared packages
│   ├── tool/             # Tool registry & schema generation
│   ├── xml/              # Output encoding (XML/JSON/plain)
│   ├── detect/           # Language & MIME detection
│   ├── path/             # Path resolution
│   ├── format/           # Size formatting
│   └── meta/             # Timestamps
└── tools/                # Individual tool implementations
    └── <toolname>/
        └── <toolname>.go # Tool implementation
```

## Adding a New Tool

1. Create a new directory under `tools/<toolname>/`
2. Implement the tool following the pattern:

```go
package toolname

import (
    "encoding/xml"
    "flag"
    "os"

    "github.com/synseqack/aict/internal/detect"
    "github.com/synseqack/aict/internal/meta"
    pathutil "github.com/synseqack/aict/internal/path"
    "github.com/synseqack/aict/internal/tool"
    xmlout "github.com/synseqack/aict/internal/xml"
)

func init() {
    tool.Register("toolname", Run)
}

type Config struct {
    XML     bool
    JSON    bool
    Plain   bool
    Pretty  bool
    // Add tool-specific flags
}

type ToolResult struct {
    XMLName   xml.Name `xml:"toolname"`
    Timestamp int64    `xml:"timestamp,attr"`
    // Add result fields
}

func Run(args []string) error {
    cfg, paths := parseFlags(args)
    // Tool logic
    return outputResult(result, cfg)
}

func parseFlags(args []string) (Config, []string) {
    // Use stdlib flag package
    // Always set XML/JSON/Plain from env var or flags
}

func outputResult(result *ToolResult, cfg Config) error {
    if cfg.JSON {
        return xmlout.WriteJSON(os.Stdout, result)
    }
    if cfg.Plain {
        return writePlain(os.Stdout, result)
    }
    return xmlout.WriteXML(os.Stdout, result, cfg.Pretty)
}
```

3. Register it in `main.go`:

```go
import _ "github.com/synseqack/aict/tools/toolname"
```

## Output Requirements

All tools must:

- Output valid XML with root element named after the tool
- Include `timestamp` attribute (Unix epoch)
- Support `--xml`, `--json`, `--plain` flags
- Support `AICT_XML=1` environment variable
- Return structured errors via `<error>` elements
- Use absolute paths in output
- Include human-readable companions for bytes/sizes

## Testing

Add tests in `<toolname>_test.go`:

```go
package toolname

import (
    "bytes"
    "encoding/xml"
    "os"
    "testing"
)

func TestTool_Basic(t *testing.T) {
    dir := t.TempDir()
    // Create test file
    // Run tool
    // Validate XML output
}

func TestTool_Error(t *testing.T) {
    // Test error handling
}
```

## Code Style

- Use `gofmt` for formatting
- Avoid external dependencies (stdlib only)
- Use lowercase for error messages (no punctuation)
- Export structs for XML marshaling
