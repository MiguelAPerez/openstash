#!/usr/bin/env python3
"""
Mechanical context-cost benchmark for openstash.

Question it answers, with no LLM in the loop:
  "To give an agent everything it needs to correctly call ONE operation,
   how many tokens of context does each access strategy cost?"

Strategies compared per task:
  A. full-dump   - paste the entire spec into context (no tool)
  B. naive-jq    - extract the raw operation, then chase its $refs by hand
                   (what a capable agent does today with grep/jq)
  C. openstash   - search -> show --expand (refs resolved for you)

Outputs are tokenized with tiktoken o200k_base (GPT-4o / modern-LLM proxy).
Also reports the $ref round-trips openstash eliminates, since each one is
a tool call an agent would otherwise have to make.
"""

import glob
import json
import os
import re
import subprocess
import sys

import tiktoken

BIN = os.environ.get("OPENSTASH_BIN", "/tmp/openstash")
STORE = os.environ.get("OPENSTASH_STORE", "/tmp/benchstore")
HERE = os.path.dirname(os.path.abspath(__file__))
ENC = tiktoken.get_encoding("o200k_base")

REF_RE = re.compile(r'"\$ref"\s*:\s*"(#/[^"]+)"')


def toks(s: str) -> int:
    return len(ENC.encode(s))


def run(*args) -> str:
    out = subprocess.run(
        [BIN, "--store", STORE, *args],
        capture_output=True, text=True,
    )
    if out.returncode != 0:
        sys.exit(f"openstash {' '.join(args)} failed (exit {out.returncode}):\n{out.stderr.strip()}")
    return out.stdout


def spec_path(key: str) -> str:
    hits = sorted(glob.glob(os.path.join(STORE, "specs", key, "*", "spec.json")))
    if not hits:
        sys.exit(f"no stored spec for {key}")
    return hits[-1]


def resolve_pointer(doc, pointer):
    # "#/components/schemas/Pet" -> doc["components"]["schemas"]["Pet"]
    node = doc
    for part in pointer.lstrip("#/").split("/"):
        part = part.replace("~1", "/").replace("~0", "~")
        if isinstance(node, list):
            node = node[int(part)]
        else:
            node = node.get(part)
        if node is None:
            return None
    return node


def direct_refs(obj_json: str):
    return list(dict.fromkeys(REF_RE.findall(obj_json)))  # unique, ordered


def closure_refs(doc, start_json: str):
    """All $refs reachable from start (transitive) -> the true minimal slice."""
    seen, frontier, nodes = set(), direct_refs(start_json), {}
    while frontier:
        ptr = frontier.pop()
        if ptr in seen:
            continue
        seen.add(ptr)
        node = resolve_pointer(doc, ptr)
        if node is None:
            continue
        nj = json.dumps(node)
        nodes[ptr] = nj
        for r in direct_refs(nj):
            if r not in seen:
                frontier.append(r)
    return nodes


def measure(task):
    key, query, method, path = task["key"], task["query"], task["method"], task["path"]
    doc = json.load(open(spec_path(key)))

    # A. full dump
    full = toks(json.dumps(doc))

    # B. naive jq: raw operation, plus directly-referenced components (1 hop),
    #    plus the full transitive slice as the agent's true floor.
    path_item = doc["paths"][path]
    mk = next(k for k in path_item if k.lower() == method.lower())
    op = path_item[mk]
    op_json = json.dumps(op, indent=2)
    op_tok = toks(op_json)
    direct = direct_refs(op_json)
    one_hop = op_tok + sum(
        toks(json.dumps(resolve_pointer(doc, r), indent=2)) for r in direct
        if resolve_pointer(doc, r) is not None
    )
    clos = closure_refs(doc, op_json)
    closure_tok = op_tok + sum(toks(v) for v in clos.values())

    # C. openstash
    search_tok = toks(run("search", key, query))
    show_tok = toks(run("show", key, "--method", method, "--path", path))
    expand_tok = toks(run("show", key, "--method", method, "--path", path, "--expand"))

    return {
        "key": key, "method": method, "path": path,
        "full": full,
        "op_raw": op_tok,
        "naive_1hop": one_hop,
        "naive_closure": closure_tok,
        "refs_direct": len(direct),
        "refs_closure": len(clos),
        "os_search": search_tok,
        "os_show": show_tok,
        "os_expand": expand_tok,
    }


def fmt(n):
    return f"{n:,}"


def main():
    tasks = json.load(open(os.path.join(HERE, "tasks.json")))["tasks"]
    rows = [measure(t) for t in tasks]

    lines = []
    lines.append("# openstash context-cost benchmark\n")
    lines.append("Tokens (o200k_base) of context needed to fully understand and call **one operation**.\n")
    lines.append("| spec | operation | A. full-dump | B. naive jq+refs (1-hop / full) | C. openstash show --expand | C vs A | C vs B(1-hop) | ref hops saved |")
    lines.append("|---|---|--:|--:|--:|--:|--:|--:|")
    for r in rows:
        vsA = r["full"] / r["os_expand"]
        vsB = r["naive_1hop"] / r["os_expand"]
        lines.append(
            f"| {r['key']} | `{r['method']} {r['path']}` | {fmt(r['full'])} | "
            f"{fmt(r['naive_1hop'])} / {fmt(r['naive_closure'])} | {fmt(r['os_expand'])} | "
            f"{vsA:.0f}x | {vsB:.1f}x | {r['refs_direct']} direct / {r['refs_closure']} total |"
        )

    lines.append("\n## openstash command ladder (tokens)\n")
    lines.append("| spec | search (discover) | show (one op) | show --expand (ready to call) |")
    lines.append("|---|--:|--:|--:|")
    for r in rows:
        lines.append(f"| {r['key']} | {fmt(r['os_search'])} | {fmt(r['os_show'])} | {fmt(r['os_expand'])} |")

    # cost estimate at a representative input price
    PRICE = 3.0 / 1_000_000  # $/input token, Sonnet-class
    lines.append(f"\n## Cost per lookup at ${PRICE*1e6:.0f}/M input tokens\n")
    lines.append("| spec | full-dump | openstash --expand | saved |")
    lines.append("|---|--:|--:|--:|")
    for r in rows:
        a, c = r["full"] * PRICE, r["os_expand"] * PRICE
        lines.append(f"| {r['key']} | ${a:.4f} | ${c:.4f} | {(1-c/a)*100:.1f}% |")

    md = "\n".join(lines) + "\n"
    open(os.path.join(HERE, "results.md"), "w").write(md)

    with open(os.path.join(HERE, "results.csv"), "w") as f:
        cols = list(rows[0].keys())
        f.write(",".join(cols) + "\n")
        for r in rows:
            f.write(",".join(str(r[c]) for c in cols) + "\n")

    print(md)
    print(f"wrote {HERE}/results.md and results.csv")


if __name__ == "__main__":
    main()
