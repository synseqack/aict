#!/usr/bin/env python3
"""Count real tokenizer tokens for tokenbench sample captures.

Usage:
    go build -o aict . && go run ./cmd/tokenbench -aict ./aict -samples benchmarks/token-samples
    python benchmarks/count_tokens.py benchmarks/token-samples

Requires: pip install tiktoken

Counts each <scenario>-{gnu,aict}.txt pair with the o200k_base encoding
(GPT-4o family). Anthropic's tokenizer is not publicly downloadable, but
counts track closely across modern BPE tokenizers; o200k_base is a fair,
reproducible proxy anyone can verify.
"""

import sys
from pathlib import Path

try:
    import tiktoken
except ImportError:
    sys.exit("count_tokens.py: pip install tiktoken first")


def main() -> None:
    samples = Path(sys.argv[1] if len(sys.argv) > 1 else "benchmarks/token-samples")
    if not samples.is_dir():
        sys.exit(f"count_tokens.py: samples directory not found: {samples}\n"
                 "  generate it first: go run ./cmd/tokenbench -aict ./aict -samples benchmarks/token-samples")

    enc = tiktoken.get_encoding("o200k_base")

    scenarios = sorted({p.name.rsplit("-", 1)[0] for p in samples.glob("*-gnu.txt")})
    if not scenarios:
        sys.exit(f"count_tokens.py: no *-gnu.txt samples in {samples}")

    print("| Task | GNU tokens | aict tokens | Ratio |")
    print("|------|------------|-------------|-------|")
    for name in scenarios:
        gnu = len(enc.encode((samples / f"{name}-gnu.txt").read_text(encoding="utf-8", errors="replace")))
        aict = len(enc.encode((samples / f"{name}-aict.txt").read_text(encoding="utf-8", errors="replace")))
        print(f"| {name} | {gnu} | {aict} | {aict / max(gnu, 1):.2f}x |")

    print("\nEncoding: o200k_base (tiktoken). See benchmarks/TOKENS.md for methodology.")


if __name__ == "__main__":
    main()
