# Installation

## Option 1: Go Install (Recommended)

```bash
go install github.com/synseqack/aict@latest
```

This places `aict` in your `$GOPATH/bin` directory. The MCP server is built in: `aict mcp`.

## Option 2: Build from Source

```bash
git clone https://github.com/synseqack/aict
cd aict
go build -o aict .
```

## Option 3: Docker

```bash
docker build -t aict .
docker run --rm -v "$(pwd)":/work -w /work aict ls .
```

## Verify Installation

```bash
aict --help
aict ls .
```

You should see a list of available commands and an XML directory listing.

## Shell Completion

```bash
# Bash
source completions/aict.bash

# Zsh
source completions/aict.zsh
```

Add the appropriate line to your `.bashrc` or `.zshrc` for persistent completion.
