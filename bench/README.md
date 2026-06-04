# openstash benchmarks

Does feeding an agent a **targeted lookup** beat handing it a giant `swagger.json`?
These benchmarks measure it two ways — context cost (mechanical) and task accuracy
(a real model in the loop) — across a 350× spec-size gradient.

## Headline


| Spec     | Spec size | Tokens to call one op | Reduction | Endpoint accuracy |
| -------- | --------- | --------------------- | --------- | ----------------- |
| petstore | 20 KB     | 4,953 → **1,585**     | 3×        | 100% → **100%**   |
| cursor   | 72 KB     | 12,548 → **1,833**    | 7×        | 0% → **100%**     |
| gitea    | 820 KB    | 148,146 → **866**     | 171×      | 0% → **100%**     |
| stripe   | 7.5 MB    | 1,054,928 → **5,106** | **207×**  | 0% → **100%**     |


*Tokens* = full spec vs. `openstash show --expand` for one operation (tiktoken `o200k_base`).
*Accuracy* = a 14B model picking the correct `{method, path}` from a plain-English task,
given a truncated full-spec paste vs. `openstash search` output. Left number → right number.

The advantage **grows with spec size**: negligible on a toy spec, decisive on a real one.
On Stripe the full spec is ~1M tokens — it doesn't fit in context at all, so the naive
approach doesn't just cost more, it *fails*.

## Two benchmarks

### 1. Context cost — `run.py` (no model)

> *To give an agent everything it needs to call one operation, how many tokens does each strategy cost?*

For each task it builds three views of the same operation and tokenizes them:

- **A. full-dump** — the entire `spec.json`. What you'd paste with no tool.
- **B. naive jq + ref-chasing** — pull `paths[path][method]` from the raw spec, then follow
its `$ref`s by hand. Reported at **1-hop** (op + direct schemas) and **full closure**
(every transitively reachable schema), plus the `$ref` count = round-trips a grep-based
agent makes manually.
- **C. openstash** — the real output of `openstash show … --expand`.

**Honest finding:** C ≈ B on raw tokens — openstash isn't magically smaller than careful
`jq`. The wins are that **C ≪ A and scales with spec size**, and C does in *one* call what
B needs many ref-hops for (Stripe's charge op fans out to **854** schema components; naive
full resolution explodes to **262K tokens**, while `--expand` gives the bounded view in one shot).

→ writes `results.md` / `results.csv`

### 2. Accuracy — `accuracy.py` (model in the loop)

> *Does openstash-shaped context make the model answer correctly more often?*

Holds **everything constant except the context** and asks the model to pick the right endpoint
for a natural-language task. Three arms:

- **closed-book** — no spec. The hallucination floor.
- **full-dump** — the raw spec, truncated to a realistic budget (`DUMP_BUDGET` tokens).
On big specs the target op falls outside the window — a deliberate, realistic failure mode.
- **openstash** — the slim `openstash search <query>` surface (~500 tokens).

Same model, same question, same grader (normalized exact-match on `{method, path}`, with
path-template canonicalization so `{owner}`, `:owner`, and a literal value compare equal).
Any accuracy delta is attributable to *how the API info was delivered*.

**What the failures look like** (these are the real story, not the means):

- **closed-book** invents plausible-but-wrong paths — `/pets` not `/pet`, `/agents` missing
the `/v1`, a fabricated `PUT /charges/{id}`.
- **full-dump (truncated)** actively *misleads* on big specs: for gitea it emitted
`/api/v1/repos/...` (grabbed a server prefix from the truncated head); for Stripe the
relevant op was truncated out, so it answered an unrelated `PATCH /v1/accounts/{account_id}`.
- **openstash** gets all four right.

→ writes `accuracy.md`

## Method notes / honesty

- **One variable.** Across arms only the context shape changes — model, prompt, and grader are fixed.
- **Ground truth from the spec**, not vibes — `tasks.json` pins `{method, path}`; grading is auto.
- **Baselines are steelmanned** — full-dump is the naive paste; the `jq` arm resolves refs *for*
the competitor; closed-book exposes the floor.
- **Scope.** Endpoint *selection* only. Param/schema-level correctness (using `show`/`gather`/
`schema`) is a natural next task type and reuses the same `tasks.json`.
- **The harness caught its own bug:** the first Stripe task was self-contradictory ("create"
labeled on the *update* endpoint); it was fixed before the final run.

## Reproduce

Requires the `openstash` binary, the four specs in a store, and `tiktoken`.

```bash
# 1. build the binary
go build -o /tmp/openstash ./cmd/openstash

# 2. populate an isolated store with the four specs
rm -rf /tmp/benchstore
curl -fsSL -o /tmp/petstore.json https://petstore3.swagger.io/api/v3/openapi.json
curl -fsSL -o /tmp/stripe.json   https://raw.githubusercontent.com/stripe/openapi/master/openapi/spec3.json
/tmp/openstash --store /tmp/benchstore add petstore --from /tmp/petstore.json
/tmp/openstash --store /tmp/benchstore add stripe   --from /tmp/stripe.json
# gitea + cursor: add from your own sources, or copy from ~/.openstash/specs

# 3. tokenizer (into a venv)
python3 -m venv .venv && .venv/bin/pip install -r bench/requirements.txt

# 4. run
.venv/bin/python bench/run.py                                # context cost (no model)
.venv/bin/python bench/accuracy.py                         # accuracy (model in loop)
```

Env knobs: `OPENSTASH_BIN`, `OPENSTASH_STORE`, `OLLAMA_HOST`, `OLLAMA_MODEL`,
`BENCH_N`, `DUMP_BUDGET`, `NUM_CTX`.

> Note: a flaky/low-memory Ollama host may choke on the larger full-dump prompts. If runs
> stall, lower `DUMP_BUDGET` (e.g. 3500) and `NUM_CTX` (e.g. 4096).

