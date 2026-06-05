#!/usr/bin/env python3
"""
Accuracy eval for openstash — model in the loop, context held as the only variable.

Same model, same question ("which endpoint does X?"), three context arms:
  closed-book : no spec at all          -> hallucination floor
  full-dump   : raw spec, truncated to a realistic local context budget
  openstash   : `openstash search` output for the query (slim, targeted)

Task = pick the correct {method, path} for a natural-language description.
Ground truth comes from tasks.json. Grading is normalized exact match.

Runs against an Ollama-compatible endpoint. No paid API.
"""

import glob
import json
import os
import re
import subprocess
import sys
import time
import urllib.request

import tiktoken

BIN = os.environ.get("OPENSTASH_BIN", "/tmp/openstash")
STORE = os.environ.get("OPENSTASH_STORE", "/tmp/benchstore")
HOST = os.environ.get("OLLAMA_HOST", "http://localhost:11434")
MODEL = os.environ.get("OLLAMA_MODEL", "qwen2.5-coder:14b")
N = int(os.environ.get("BENCH_N", "3"))
DUMP_BUDGET = int(os.environ.get("DUMP_BUDGET", "3500"))  # tokens of spec a naive paste fits
NUM_CTX = int(os.environ.get("NUM_CTX", "4096"))
HERE = os.path.dirname(os.path.abspath(__file__))
ENC = tiktoken.get_encoding("o200k_base")

PROMPT = (
    "You are given an API task. Identify the single correct HTTP operation.\n"
    "Task: {task}\n{ctx}"
    "Reply with ONLY a JSON object: {{\"method\": \"...\", \"path\": \"...\"}}. No prose, no markdown."
)
TASK_PHRASING = {
    ("petstore", "/pet", "POST"): "Add a new pet to the store.",
    ("cursor", "/v1/agents", "POST"): "Launch / create a new agent.",
    ("gitea", "/repos/{owner}/{repo}/issues", "GET"): "List the issues in a repository.",
    ("stripe", "/v1/charges/{charge}", "POST"): "Update an existing charge by its id (not create a new one).",
}


def toks(s):
    return len(ENC.encode(s))


def truncate(s, budget):
    ids = ENC.encode(s)
    return ENC.decode(ids[:budget]) if len(ids) > budget else s


def spec_path(key):
    hits = sorted(glob.glob(os.path.join(STORE, "specs", key, "*", "spec.json")))
    if not hits:
        sys.exit(f"no stored spec for {key!r} — run `openstash add` first")
    return hits[-1]


def run(*args):
    result = subprocess.run([BIN, "--store", STORE, *args], capture_output=True, text=True)
    if result.returncode != 0:
        sys.exit(f"openstash {' '.join(args)} failed (exit {result.returncode}):\n{result.stderr.strip()}")
    return result.stdout


def generate(prompt, retries=4):
    body = json.dumps({
        "model": MODEL, "prompt": prompt, "stream": False,
        "options": {"temperature": 0, "num_ctx": NUM_CTX},
    }).encode()
    last = None
    for attempt in range(retries):
        try:
            req = urllib.request.Request(HOST + "/api/generate", data=body,
                                         headers={"Content-Type": "application/json"})
            with urllib.request.urlopen(req, timeout=90) as r:
                return json.load(r)["response"]
        except Exception as e:  # transient endpoint flakiness — back off and retry
            last = e
            time.sleep(2 * (attempt + 1))
    return f"ERR {last}"


def norm_path(p):
    if not p:
        return ""
    p = p.strip().rstrip("/")
    p = re.sub(r"[:{]?[A-Za-z_][A-Za-z0-9_]*[}]?", lambda m: "{P}" if (m.group(0).startswith(("{", ":"))) else m.group(0), p)
    # collapse any templated segment to {P}
    p = re.sub(r"\{[^/]+\}", "{P}", p)
    p = re.sub(r"/:[^/]+", "/{P}", p)
    return p.lower()


def parse(resp):
    m = re.search(r'"method"\s*:\s*"([^"]+)"', resp)
    p = re.search(r'"path"\s*:\s*"([^"]+)"', resp)
    return (m.group(1).upper() if m else "", p.group(1) if p else "")


def grade(pred, gt_method, gt_path):
    pm, pp = pred
    return pm == gt_method.upper() and norm_path(pp) == norm_path(gt_path)


def context_for(arm, task):
    key, q = task["key"], task["query"]
    if arm == "closed-book":
        return ""
    if arm == "full-dump":
        raw = open(spec_path(key)).read()
        return "API spec (may be truncated):\n" + truncate(raw, DUMP_BUDGET) + "\n"
    if arm == "openstash":
        # endpoint-selection task -> the slim `search` surface is the right tool.
        # (params/schema questions would use `show`/`gather`; not this task type.)
        return "Candidate endpoints (from `openstash search`):\n" + run("search", key, q, "--limit", "6") + "\n"
    raise ValueError(arm)


def main():
    tasks = json.load(open(os.path.join(HERE, "tasks.json")))["tasks"]
    arms = ["closed-book", "full-dump", "openstash"]
    results = {a: [] for a in arms}
    rows = []

    print(f"model={MODEL}  N={N}  dump_budget={DUMP_BUDGET}\n")
    for t in tasks:
        key = t["key"]
        phrasing = TASK_PHRASING[(key, t["path"], t["method"])]
        for arm in arms:
            ctx = context_for(arm, t)
            ctx_tok = toks(ctx)
            prompt = PROMPT.format(task=phrasing, ctx=ctx)
            hits = 0
            ex = ("", "")
            for _ in range(N):
                try:
                    resp = generate(prompt)
                except Exception as e:
                    resp = f"ERR {e}"
                pred = parse(resp)
                ex = pred
                if grade(pred, t["method"], t["path"]):
                    hits += 1
            acc = hits / N
            results[arm].append(acc)
            rows.append((key, arm, acc, ctx_tok, ex))
            print(f"  {key:9} {arm:11} acc={acc:.2f}  ctx={ctx_tok:>6} tok  e.g.={ex[0]} {ex[1]}")

    # report
    lines = ["# openstash accuracy eval\n",
             f"Model: `{MODEL}` · N={N} runs/task · full-dump budget={DUMP_BUDGET} tok · temp 0\n",
             "Task: pick the correct `{method, path}` from a natural-language description.\n",
             "| spec | closed-book | full-dump (truncated) | openstash |",
             "|---|--:|--:|--:|"]
    by = {a: {} for a in arms}
    for key, arm, acc, ctx_tok, ex in rows:
        by[arm][key] = acc
    for t in tasks:
        k = t["key"]
        lines.append(f"| {k} | {by['closed-book'][k]:.0%} | {by['full-dump'][k]:.0%} | {by['openstash'][k]:.0%} |")
    lines.append(f"| **mean** | **{sum(results['closed-book'])/len(tasks):.0%}** | "
                 f"**{sum(results['full-dump'])/len(tasks):.0%}** | "
                 f"**{sum(results['openstash'])/len(tasks):.0%}** |")
    md = "\n".join(lines) + "\n"
    open(os.path.join(HERE, "accuracy.md"), "w").write(md)
    print("\n" + md)
    print(f"wrote {HERE}/accuracy.md")


if __name__ == "__main__":
    main()
