# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `aict version` / `--version` / `-V` command; release binaries embed the tag via ldflags
- Token-cost benchmark (`cmd/tokenbench`, `benchmarks/TOKENS.md`, `make bench-tokens`) measuring context-window cost vs GNU coreutils
- `doctor` now reports the aict version

### Changed
- Help output lists tools and flags in sorted, deterministic order
- CI now runs the test suite on Linux, Windows, and macOS, and on every push to master
- Documented dependency policy accurately: tools/internal are stdlib-only; the MCP SDK (used only by `aict mcp`) is the sole external dependency

### Fixed
- Test suite passes on Windows (unix-permission and `/proc`-dependent tests are now skipped there)
- `env` redacts secret variable values in XML/JSON output

### Removed
- Scheduled issue-creator workflow

## [2.0.1] - 2026-06-27

### Changed
- docs-site updated for v2.0 (33 tools, `aict mcp`, new tools)

## [2.0.0] - 2026-06-27

### Added
- 5 new tools: `sed`, `awk`, `jq`, `tar`, `completions`
- `git` tool (status, diff, log, ls-files, blame) and `doctor` self-diagnostic
- Benchmark suite (`cmd/bench`, `make bench`) comparing aict against GNU coreutils
- `--workers` flag for parallel grep worker count

### Changed
- MCP server consolidated into the main binary as `aict mcp` (separate `aict-mcp` binary removed)

### Fixed
- `df` on Windows: enumerate drives via `GetLogicalDrives` instead of `/proc/mounts`

## [1.0.3] - 2026-04-06

### Fixed
- Migration and integration guides

## [1.0.2] - 2026-04-06

### Changed
- Consolidated tool documentation into a single generated file

## [1.0.1] - 2026-04-06

### Added
- MCP tool annotations, privacy policy, and usage examples
- Homebrew formula for macOS installation
- CONTRIBUTING.md guide, issue and PR templates
- GitHub Actions CI workflow
- Docker build configuration

## [1.0.0] - 2026-04-06

### Added
- **Phase 0**: Foundation
  - Go module and directory structure
  - Internal packages (xml, detect, path, format, meta)
  - `ls` tool with full XML output

- **Phase 1**: Core Reads
  - `cat` - File read with encoding detection
  - `grep` - Recursive regex search
  - `find` - Filesystem search
  - `stat` - File metadata
  - `wc` - Line/word/char/byte counting
  - `diff` - Myers diff algorithm

- **Phase 2**: Contextual Enrichment
  - `file` - Type detection
  - `head`/`tail` - Partial file read
  - `du`/`df` - Disk usage
  - `realpath`/`basename`/`dirname` - Path utilities
  - `pwd` - Working directory
  - `sort`/`uniq` - Sorting and deduplication
  - `cut`/`tr` - Text processing
  - `env` - Environment with secret redaction
  - `system` - Combined system info
  - `ps` - Process listing
  - `checksums` - Hash computation
  - MCP server (`cmd/mcp`)

### Features
- XML output (default)
- JSON output (`--json`)
- Plain text output (`--plain`)
- `AICT_XML=1` environment variable
- Structured error elements
- Language detection
- MIME type detection

[Unreleased]: https://github.com/synseqack/aict/compare/v2.0.1...HEAD
[2.0.1]: https://github.com/synseqack/aict/compare/v2.0...v2.0.1
[2.0.0]: https://github.com/synseqack/aict/compare/v1.0.3...v2.0
[1.0.3]: https://github.com/synseqack/aict/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/synseqack/aict/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/synseqack/aict/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/synseqack/aict/releases/tag/v1.0.0
