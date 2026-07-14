# Contributing

Thank you for your interest in contributing to aict.

## Project Structure

```
aict/
├── main.go       # Entry point
├── cmd/mcp/               # MCP server implementation
├── internal/              # Shared packages
│   ├── xml/               # Output encoding (XML/JSON/plain)
│   ├── detect/            # Language & MIME detection
│   ├── path/              # Path resolution
│   ├── format/            # Size formatting
│   └── meta/              # Timestamps
└── tools/                 # Individual tool implementations
    └── <toolname>/
        └── <toolname>.go  # Tool implementation
```

## Adding a New Tool

1. Create a new directory under `tools/<toolname>/`
2. Implement the tool following the pattern in `CONTRIBUTING.md`
3. Register it in `main.go`:
   ```go
   import _ "github.com/synseqack/aict/tools/toolname"
   ```
4. Write tests in `<toolname>_test.go`

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

```bash
go test ./...
go test ./tools/toolname/
```

## Code Style

- Use `gofmt` for formatting
- Avoid external dependencies (stdlib only)
- Use lowercase for error messages (no punctuation)
- Export structs for XML marshaling

## Dependencies

**ALLOWED** — Go standard library only.

**FORBIDDEN** — no external dependencies (`github.com/...` imports, `go get`, cgo).

Exception: the MCP server (`cmd/mcp`) uses `github.com/modelcontextprotocol/go-sdk`.
