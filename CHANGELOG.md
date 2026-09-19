# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- MCP server: every tool's input schema now advertises its positional
  inputs (`paths`, `pattern`, `old`/`new`, `archive`, …) with the order the
  tool's argv requires. Previously an agent could not discover how to pass a
  path or a search pattern at all.
- MCP server: compact JSON responses carry a `_legend` object mapping the
  short names aict emits (`p`, `fl`, `cs`) back to their long forms, so an
  agent can decode a response without a second round trip. Only the
  abbreviations a response actually uses are included.
- `grep` uses ripgrep when it is on `PATH`: the search is handed to
  `rg --json` and re-emitted in aict's own schema. Which files are searched,
  which are skipped as binary, the match counts, and every emitted field are
  identical to the built-in engine — ripgrep supplies only the matching.
  `aict doctor` reports whether ripgrep was found. Set `AICT_NORG=1` to
  disable the integration; if ripgrep fails outright, `grep` falls back to
  the built-in engine instead of reporting an error.
- `xmlout.Bool`: a boolean type that emits `1`/`0` in compact XML, `true`/`false`
  in verbose XML, and a native unquoted boolean under `--json` in every mode.
- `internal/xml` tests asserting value preservation in compact mode and native
  JSON booleans.
- `cat -n`/`--number`: each output line carries its 1-based number and text in a
  `<line>` element. `content` is unchanged with and without the flag, so an
  existing consumer still finds the text it expects; `--plain` renders GNU's
  `%6d\t` numbering.
- `diff -u`/`-U N`/`--context N`: context lines now flank each change, hunks
  merge when their context windows touch, and hunk headers match GNU diff
  (`-0,0` for a pure insertion at the top of a file). `-U 0` still reproduces
  the old changed-lines-only output. GNU's glued spelling `-U3` is accepted.
- `checksums -c`/`--check` (also `md5sum -c`, `sha1sum -c`, `sha256sum -c`):
  reads a manifest of `HASH  PATH` lines and reports each entry as ok or
  failed. The algorithm is inferred from the digest length, so a manifest
  carries no algorithm label. A missing or malformed line becomes an
  `<error>`; a digest mismatch is reported in-band and exits 0, like every
  other structured failure. `--plain` prints GNU's `path: OK` / `path: FAILED`
  and a `WARNING: N failed, M could not be read` summary.

### Changed
- Boolean attributes in every tool's result structs are typed `xmlout.Bool`
  instead of pre-formatted strings. **`--json` now emits real booleans
  (`"binary": false`) rather than strings (`"binary": "false"`) — a behavior
  change for `--json` consumers.** XML output in both compact and verbose mode
  is otherwise unchanged byte-for-byte, except for empty files, which now
  correctly report `binary="false"` and a text MIME (see Fixed).
- `--no-compact` is now honoured by every tool. It sets `AICT_NOCOMPACT=1`
  once in the subcommand dispatcher rather than being re-parsed in each
  package, so a tool that did not know the flag no longer silently ignores it.
- `aict help` lists `--no-compact` alongside the output modes, and no longer
  describes `--xml` as "default if `AICT_XML=1`" — XML is the default mode
  with or without the variable.

### Fixed
- Compact mode no longer rewrites attribute *values* that happen to read as
  booleans. Previously the shortening pass replaced `="true"` globally, so a
  file named `true` was listed as `name="1"`, a symlink targeting `true` became
  `target="1"`, and grepping for the literal pattern `true` reported `p="1"`.
  Only genuine boolean attributes compact now; values are intact in every mode.
- Empty files are classified as text, not binary: `ls`, `cat`, and `file` now
  report `mime="text/plain; charset=utf-8"` and `binary="false"` for them. A
  zero-byte read returns `(0, io.EOF)`, which was being treated as a read error
  and left the file classified as an opaque octet stream.
- `ls` no longer hangs on non-regular files. A FIFO with no writer blocks
  `os.Open` until one appears, so a single FIFO anywhere in a tree hung the
  whole listing; content detection now runs only on regular files. The same
  guard now covers language detection's shebang sniff, so `find`, `stat`, and
  `file` do not hang on a FIFO either. `cat`, `head`, and `tail` read content
  directly and still block, as GNU does.
- Compact mode shortened the wrong direction. The replacement table is looked
  up by long name, but registered dictionaries are short → long and were used
  as-is, so a tool with a dictionary kept its long attribute names in compact
  output. The table is inverted at the source now; `--pretty` is also honoured
  in compact mode instead of being dropped.
- `find`'s `<condition>` elements and `cat`'s `encoding` attribute had no JSON
  tags, so `--json` emitted Go field names (`Type`, `Value`, `Negated`) that
  the `--dict` legend never advertised. Both now use the documented short
  keys. An empty text file also reported `encoding=""` instead of `utf-8`.
- `file` reported `charset="binary"` for an empty text file; the zero-byte
  read was treated as an error. It now reports `UTF-8`.
- `ls` reported a missing path as `path not found`, while the other ten
  file-reading tools all report `no such file or directory`. An agent
  matching on the message had to special-case one tool; `ls` now matches.
- Documentation drifted from the binary: the tool count read 33 in four
  places (34 is correct), `AICT_JSON=1` was documented in three places and
  never read by any tool, and every guide implied `export AICT_XML=1` was
  required to get XML — XML is the default, and the variable only forces it
  over an earlier `--json`/`--plain`. `AGENTS.md` also documented a
  `RunWithOutput` test helper that does not exist and an import path
  (`internal/xmlout`) that is not the package's path; both now match the
  code.
- `env` and `ls` plain-text renderers had no test coverage at all. Both now
  have format assertions — including the one path that can drop data on the
  floor, `env`'s redacted values.
- `grep -m N` now stops after exactly N matches. The counter was incremented
  after the limit was tested, so `-m 1` reported two matches. This matches the
  flag's documented meaning and ripgrep's `-m`, and makes both grep backends
  agree.
- The MCP server no longer shuffles positional arguments. A tool's inputs
  arrive as a JSON object, which carries no order, and the argv was built by
  walking that object's map — so `diff a b` came out as `b a` on roughly one
  call in five, and `cat f1 f2 f3` listed files in a different order each
  time. Positionals are now emitted in the order each tool's parser expects.
- The MCP server dropped JSON array arguments entirely, so
  `checksums -a md5 -a sha256` was unreachable over MCP. Arrays now expand to
  repeated flags.
- `jq`'s `path` property over MCP emitted a literal `.` as the program, and
  `completions`' `shell` emitted a literal `bash` alongside the requested
  shell. Both now translate to real flags (`-p`) or positionals.
- Twelve flags that aict's parsers accept but whose values are never read are
  no longer advertised in tool input schemas: `du -h`, `df -h`, `grep -E`,
  `ps -a/-f/-p/--sort`, `sort -o`, `tail -f`, `tar -t`, and `wc -a`. Each
  either duplicates behaviour that is always on, or is unimplemented;
  advertising it promised an effect the tool did not have. The flags are still
  accepted on the command line for compatibility. (`cat -n`, `checksums -c`,
  and `diff -U` were on this list too; all three are implemented above and
  advertised again.)

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
