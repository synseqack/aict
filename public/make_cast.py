#!/usr/bin/env python3
"""Synthesize an asciicast v2 file for the aict README demo.

Outputs are real captures from running aict in this directory; only the
typing rhythm is scripted. Deterministic (seeded) so the cast is
reproducible.
"""

import json
import random
import sys

rng = random.Random(42)

W, H = 112, 30
events = []
t = 0.0

GREEN = "\x1b[32m"
CYAN = "\x1b[36m"
DIM = "\x1b[2m"
BOLD = "\x1b[1m"
RESET = "\x1b[0m"
PROMPT = f"{GREEN}${RESET} "


def emit(text, dt=0.0):
    global t
    t += dt
    events.append([round(t, 4), "o", text])


def type_text(text, base=0.045):
    for ch in text:
        emit(ch, base + rng.uniform(-0.02, 0.03))


def command(cmd, output, hold=2.8, chunk_delay=0.006):
    emit(PROMPT, 0.15)
    type_text(cmd)
    emit("\r\n", 0.35)
    for line in output.splitlines():
        emit(line + "\r\n", chunk_delay)
    global t
    t += hold


def comment(text, hold=1.6):
    emit(PROMPT, 0.1)
    type_text(f"{DIM}{CYAN}# {text}{RESET}", 0.03)
    emit("\r\n", 0.25)
    global t
    t += hold


def main():
    read = lambda p: open(p, encoding="utf-8").read()
    ls_plain = read("cap_ls_plain.txt")
    ls_aict = read("cap_ls_aict.txt")
    grep_aict = read("cap_grep_aict.txt")

    comment("what AI agents get from classic coreutils:")
    command("ls -la src/", ls_plain, hold=1.8)
    comment("which column is the size? is that a dir? what language?", hold=2.0)

    comment("same question, answered for machines:", hold=1.0)
    command("aict grep \"TODO\" src/ -r --pretty", grep_aict, hold=3.4)
    command("aict ls src/ --pretty", ls_aict, hold=4.2)

    comment("every field named · paths absolute · errors structured", hold=1.2)
    comment("github.com/synseqack/aict", hold=3.0)

    header = {
        "version": 2,
        "width": W,
        "height": H,
        "title": "aict — Unix coreutils built for AI agents",
        "env": {"SHELL": "/bin/bash", "TERM": "xterm-256color"},
    }
    out = sys.argv[1] if len(sys.argv) > 1 else "demo.cast"
    with open(out, "w", encoding="utf-8") as f:
        f.write(json.dumps(header) + "\n")
        for ev in events:
            f.write(json.dumps(ev, ensure_ascii=False) + "\n")
    print(f"wrote {out}: {len(events)} events, {t:.1f}s")


if __name__ == "__main__":
    main()
