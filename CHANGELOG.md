# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.2.0] - 2026-09-01

### Added
- Compact output mode (default): short attribute names (`p`, `a`, `s`, `t`) reduce token usage ~20%
- `--dict` flag shows short-to-long name mapping for any tool
- `--no-compact` flag reverts to verbose long attribute names
- JSON struct tags on all tool result types for compact JSON output
- `basename` tool with `stem` and `extension` attributes
- `dirname` tool for directory portion of paths

### Changed
- All 31 tools now register dictionaries via `xmlout.RegisterDict()`
- Booleans output as `1`/`0` in compact mode, `true`/`false` in verbose
- Updated README with compact mode documentation and tool count (34)

### Fixed
- `grep`: last line of files without trailing newline was silently dropped (`findMatches` broke on `io.EOF` instead of processing remaining data)
- `find` tests referenced non-existent `FindResult.Entries` field (should be `Matches`)

## [2.1.0] - 2026-07-14

### Added
- `aict version` / `--version` / `-V` command; release binaries embed the tag via ldflags
- Token-cost benchmark (`cmd/tokenbench`, `benchmarks/TOKENS.md`, `make bench-tokens`) measuring context-window cost vs GNU coreutils
- Terminal demo GIF in the README (`public/demo.gif`, generated from real captures)
- `doctor` now reports the aict version
- `find`: echoed `<condition>` elements carry `negated="true"` when inverted; `!` accepted as an alias for `-not`

### Changed
- Help output lists tools and flags in sorted, deterministic order
- CI now runs the test suite on Linux, Windows, and macOS, and on every push to master
- Documented dependency policy accurately: tools/internal are stdlib-only; the MCP SDK (used only by `aict mcp`) is the sole external dependency
- Documented the real platform matrix: `df` works on Linux/macOS/Windows; `ps` is Linux-only

### Fixed
- `ls`/`stat` reported `owner="root"` for every file on Linux and macOS (uid/gid extraction never matched `*syscall.Stat_t`)
- `df` returned zero filesystems on macOS (read `/proc/mounts`); now uses `Getfsstat` (#11 follow-up)
- `diff`: hunk `old_count`/`new_count` were always 0, and the Myers backtracking could emit unchanged lines as added+removed pairs (#32)
- `find`: `-not` inverted the entire expression instead of only the next predicate (#30)
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

[Unreleased]: https://github.com/synseqack/aict/compare/v2.1.0...HEAD
[2.1.0]: https://github.com/synseqack/aict/compare/v2.0.1...v2.1.0
[2.0.1]: https://github.com/synseqack/aict/compare/v2.0...v2.0.1
[2.0.0]: https://github.com/synseqack/aict/compare/v1.0.3...v2.0
[1.0.3]: https://github.com/synseqack/aict/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/synseqack/aict/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/synseqack/aict/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/synseqack/aict/releases/tag/v1.0.0
