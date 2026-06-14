# Lab5 stress report

| Metric | Value |
|---|---:|
| Index | `data/russian_classics.lab5` |
| Index file size | 46005657 bytes / 43.87 MiB |
| Corpus | `data/russian_classics.jsonl` |
| Corpus file size | 27893738 bytes / 26.60 MiB |
| Source HTML files | 27 |
| Source HTML lines | 89366 |
| War and Peace source lines | 24898 |
| Docs | 11092 |
| Terms | 143693 |
| Avg doc length | 218.62 |
| Raw postings bytes | 23438576 |
| Compressed postings bytes | 8323778 |
| Compression ratio | 0.355 |
| Rounds per query | 1000 |
| Go runtime | go1.25.2 |

| Case | Query | L0 matches | Returned | min ms | avg ms | p95 ms | max ms | ~QPS | Error |
|---|---|---:|---:|---:|---:|---:|---:|---:|---|
| `term_common` | `князь` | 1541 | 10 | 0.593 | 0.656 | 0.791 | 1.979 | 1523 |  |
| `term_rare` | `ростовы` | 35 | 10 | 0.271 | 0.296 | 0.359 | 0.452 | 3377 |  |
| `and` | `князь AND андрей` | 457 | 10 | 0.531 | 0.581 | 0.695 | 1.251 | 1720 |  |
| `or` | `пьер OR наташа` | 976 | 10 | 0.499 | 0.555 | 0.670 | 1.683 | 1800 |  |
| `not_nested` | `князь AND NOT (француз AND император)` | 1541 | 10 | 0.607 | 0.668 | 0.779 | 1.599 | 1496 |  |
| `adj` | `пьер ADJ безухов` | 8 | 8 | 0.278 | 0.317 | 0.381 | 1.503 | 3145 |  |
| `near_5` | `андрей NEAR/5 болконский` | 11 | 10 | 0.347 | 0.390 | 0.475 | 1.005 | 2560 |  |
| `near_20` | `наташа NEAR/20 ростова` | 3 | 3 | 0.102 | 0.115 | 0.141 | 0.429 | 8680 |  |
| `negative_only` | `NOT (князь AND андрей)` | 10635 | 10 | 0.467 | 0.548 | 0.766 | 1.820 | 1823 |  |
| `missing` | `несуществующийтерм` | 0 | 0 | 0.002 | 0.003 | 0.003 | 0.031 | 309693 |  |
| `unicode` | `мертвые AND души` | 526 | 10 | 0.521 | 0.589 | 0.720 | 3.394 | 1696 |  |
| `bad_query` | `(князь AND` | 0 | 0 | 0.000 | 0.000 | 0.000 | 0.000 | 0 | unexpected end of query |
