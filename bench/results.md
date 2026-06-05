# openstash context-cost benchmark

Tokens (o200k_base) of context needed to fully understand and call **one operation**.

| spec | operation | A. full-dump | B. naive jq+refs (1-hop / full) | C. openstash show --expand | C vs A | C vs B(1-hop) | ref hops saved |
|---|---|--:|--:|--:|--:|--:|--:|
| petstore | `POST /pet` | 4,953 | 589 / 609 | 1,585 | 3x | 0.4x | 1 direct / 3 total |
| cursor | `POST /v1/agents` | 12,548 | 1,868 / 4,989 | 1,833 | 7x | 1.0x | 8 direct / 25 total |
| gitea | `GET /repos/{owner}/{repo}/issues` | 148,146 | 819 / 3,328 | 866 | 171x | 0.9x | 2 direct / 9 total |
| stripe | `POST /v1/charges/{charge}` | 1,054,928 | 5,230 / 261,878 | 5,106 | 207x | 1.0x | 2 direct / 854 total |

## openstash command ladder (tokens)

| spec | search (discover) | show (one op) | show --expand (ready to call) |
|---|--:|--:|--:|
| petstore | 458 | 355 | 1,585 |
| cursor | 442 | 259 | 1,833 |
| gitea | 415 | 791 | 866 |
| stripe | 648 | 388 | 5,106 |

## Cost per lookup at $3/M input tokens

| spec | full-dump | openstash --expand | saved |
|---|--:|--:|--:|
| petstore | $0.0149 | $0.0048 | 68.0% |
| cursor | $0.0376 | $0.0055 | 85.4% |
| gitea | $0.4444 | $0.0026 | 99.4% |
| stripe | $3.1648 | $0.0153 | 99.5% |
