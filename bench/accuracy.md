# openstash accuracy eval

Model: `qwen2.5-coder:14b` · N=2 runs/task · full-dump budget=3500 tok · temp 0

Task: pick the correct `{method, path}` from a natural-language description.

| spec | closed-book | full-dump (truncated) | openstash |
|---|--:|--:|--:|
| petstore | 0% | 100% | 100% |
| cursor | 0% | 0% | 100% |
| gitea | 100% | 0% | 100% |
| stripe | 0% | 0% | 100% |
| **mean** | **25%** | **25%** | **100%** |
