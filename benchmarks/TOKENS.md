# Token cost: aict vs GNU coreutils

The question that actually matters for an AI-agent tool is not "how fast does
it run" but "how many context-window tokens does the agent spend to complete a
task". This benchmark measures that — honestly, including the case where aict
loses.

## Headline result

**aict output costs 1.0–6.3× more tokens per task than terse GNU output.**
With compact mode (default), this drops to **1.0–5.1×** — a ~20% reduction.
The `compare` task achieves parity (1.0×) in compact mode.
What you buy with those tokens:

- **Fewer round-trips.** 3 of 5 tasks below need 2–4 GNU calls (`ls` then
  `file`, `find` then `stat` per hit, `cat` then `wc` then `file`) where aict
  needs one. Each extra call is a full agent turn: model inference, tool
  dispatch, and the intermediate output all land in context anyway.
- **Zero parsing ambiguity.** Every field is a named attribute. No guessing
  which `ls -la` column is the size, no locale-dependent dates, no filename
  edge cases (spaces, newlines) silently corrupting a parse.
- **Correct metadata.** In this very fixture, `file(1)` misidentified a Go
  source file as "C source" and another as plain "ASCII text"; aict's
  extension-based detection labeled both `go`.

If you only need raw content and know exactly what you're looking for, GNU
tools (or `aict --plain`) are cheaper. aict is for when the agent needs the
metadata anyway — which, for inventory/locate-style tasks, it almost always
does.

## Results

Measured with tiktoken `o200k_base` on the captured transcripts
(2026-07-14, windows/amd64; regenerate with the commands below).

| Task | GNU calls | GNU tokens | aict calls | aict tokens | Ratio |
|------|-----------|------------|------------|-------------|-------|
| compare — diff two files with change types and line numbers | 1 | 167 | 1 | 192 | 1.15× |
| inventory — list a directory with size, type and language per entry | 2 | 246 | 1 | 820 | 3.33× |
| locate — find all `.go` files with size and mtime | 4 | 47 | 1 | 367 | 7.81× |
| read — read a source file plus its line count and type | 3 | 173 | 1 | 274 | 1.58× |
| search — find a pattern with file, line and context | 1 | 141 | 1 | 326 | 2.31× |

Token counts cover command output only. They do **not** include the per-call
overhead of extra agent turns (tool-call JSON, model reasoning between calls),
which multiplies the real cost of the multi-call GNU sequences.

## Compact mode (default since v0.2)

All XML/JSON output now uses short attribute names by default. Use `--dict`
to see the mapping, `--no-compact` to revert to verbose output.

### Token benchmark: compact vs verbose

| Task | Verbose tokens | Compact tokens | Savings |
|------|----------------|----------------|----------|
| inventory (ls + file) | 562 | 421 | 25% |
| read (cat + wc + file) | 176 | 162 | 8% |
| search (grep) | 255 | 208 | 18% |
| locate (find + stat) | 234 | 188 | 20% |
| compare (diff) | 133 | 92 | 31% |

**Average savings: ~20%** across benchmark scenarios.

### Per-tool byte savings

| Tool | Verbose bytes | Compact bytes | Savings |
|------|---------------|---------------|----------|
| ls | 9,035 | 7,030 | 22% |
| find | 172,556 | 146,024 | 15% |
| grep | 171 | 82 | 52% |
| cat | 130 | 77 | 40% |
| stat | 477 | 359 | 24% |
| wc | 314 | 240 | 23% |
| diff | 190 | 125 | 34% |
| head | 200 | 102 | 49% |
| tail | 200 | 102 | 49% |
| du | 29,746 | 23,675 | 20% |
| df | 8,243 | 5,135 | 37% |
| file | 112 | 72 | 35% |
| ps | 107,670 | 93,901 | 12% |
| env | 11,853 | 9,665 | 18% |
| sort | 89 | 67 | 24% |
| uniq | 86 | 52 | 39% |
| cut | 82 | 53 | 35% |
| awk | 125 | 103 | 17% |
| sed | 120 | 115 | 4% |

The `--dict` flag outputs the short→long name mapping:
```xml
<dict><a>absolute</a><p>path</p><s>size_bytes</s></dict>
```

This is useful for training small summarization models on aict output.

## Known costs in aict output

- Absolute paths are repeated in `path` and `absolute` attributes on every
  entry — by design (agents never have to resolve a path), but it is the
  single largest token cost, and ratios grow with path length. The fixture
  lives in a temp directory with a ~45-character prefix; shallow repo paths
  produce lower ratios.
- `--json` is slightly more expensive than `--xml` for the same data
  (punctuation-heavy); `--plain` matches classic tool output.

## Live agent eval (opencode)

The transcript numbers above measure output size. What matters end-to-end is
what a real agent *does* with each toolchain. We gave
[opencode](https://github.com/sst/opencode) 1.17.18 (model
`opencode/big-pickle`) the same task three times per condition over the
fixture tree: *"report every file under src/ with size, language, and TODO
lines — reply as a markdown table."* One condition allowed only standard
shell tools; the other only `aict`.

| Metric (3 runs each) | GNU tools | aict |
|----------------------|-----------|------|
| Output tokens (median) | 487 | **265** |
| Model steps | 3–4 | 2–3 |
| Fully correct answers | 2/3 | **3/3** |

Token counts from opencode's `step_finish` events (`--format json`).
Observations from the transcripts:

- The GNU run that trusted `file(1)` for languages got `server.go`
  misidentified as "C source" and shipped a report with a flawed language
  column plus a correction footnote.
- The GNU runs that stayed correct did so by *avoiding* `file(1)` — writing
  multi-line shell loops (`while read f; do size=$(stat -c%s "$f") ...`) and
  inferring languages from extensions themselves. The model can compensate on
  a 5-file fixture; that's reasoning budget spent on plumbing.
- The aict runs each answered from two data calls (`aict find --xml`,
  `aict grep --xml`) — size and language arrive as attributes, nothing to
  infer, and generated roughly half the output tokens.

Reproduce (any agent CLI that reports usage works):

```sh
opencode run --format json -m opencode/big-pickle \
  "Report every file under src/ (recursive): filename, size in bytes, \
   language, and TODO lines. Use ONLY <toolchain>. Reply as a markdown table."
```

Input-token comparisons across runs are confounded by provider prompt
caching; output tokens and correctness are the stable metrics reported here.

## Methodology

Each task is a realistic agent question. The GNU side executes the full
command sequence an agent would need — including follow-up `file`/`stat`
calls for metadata plaintext doesn't carry — and the transcript is the
concatenation of all outputs. The aict side is a single call. Both run over
an identical fixture tree (small mixed-language project; see
`cmd/tokenbench/fixtures.go`).

Reproduce:

```sh
go build -o aict .
go run ./cmd/tokenbench -aict ./aict -samples benchmarks/token-samples
pip install tiktoken
python benchmarks/count_tokens.py benchmarks/token-samples
```

`tokenbench` itself prints a chars/4 estimate so it runs with no Python
dependency; `count_tokens.py` produces the real tokenizer counts reported
here. Anthropic's tokenizer is not publicly downloadable; `o200k_base` is a
reproducible proxy that anyone can verify.
