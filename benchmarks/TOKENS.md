# Token cost: aict vs GNU coreutils

The question that actually matters for an AI-agent tool is not "how fast does
it run" but "how many context-window tokens does the agent spend to complete a
task". This benchmark measures that — honestly, including the case where aict
loses.

## Headline result

**aict output costs 1.1–7.8× more tokens per task than terse GNU output.**
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

## Known costs in aict output

- Absolute paths are repeated in `path` and `absolute` attributes on every
  entry — by design (agents never have to resolve a path), but it is the
  single largest token cost, and ratios grow with path length. The fixture
  lives in a temp directory with a ~45-character prefix; shallow repo paths
  produce lower ratios.
- `--json` is slightly more expensive than `--xml` for the same data
  (punctuation-heavy); `--plain` matches classic tool output.

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
