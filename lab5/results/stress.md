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
| `term_common` | `князь` | 1541 | 10 | 0.598 | 0.681 | 0.856 | 2.226 | 1468 |  |
| `term_rare` | `ростовы` | 35 | 10 | 0.272 | 0.304 | 0.375 | 0.810 | 3285 |  |
| `and` | `князь AND андрей` | 457 | 10 | 0.528 | 0.602 | 0.749 | 1.562 | 1660 |  |
| `or` | `пьер OR наташа` | 976 | 10 | 0.502 | 0.573 | 0.712 | 1.382 | 1744 |  |
| `not_nested` | `князь AND NOT (француз AND император)` | 1541 | 10 | 0.606 | 0.681 | 0.845 | 1.403 | 1466 |  |
| `adj` | `пьер ADJ безухов` | 8 | 8 | 0.279 | 0.312 | 0.387 | 1.436 | 3203 |  |
| `near_5` | `андрей NEAR/5 болконский` | 11 | 10 | 0.349 | 0.385 | 0.470 | 1.940 | 2592 |  |
| `near_20` | `наташа NEAR/20 ростова` | 3 | 3 | 0.103 | 0.116 | 0.147 | 0.363 | 8575 |  |
| `negative_only` | `NOT (князь AND андрей)` | 10635 | 10 | 0.468 | 0.552 | 0.772 | 1.604 | 1810 |  |
| `missing` | `несуществующийтерм` | 0 | 0 | 0.002 | 0.003 | 0.003 | 0.017 | 301659 |  |
| `unicode` | `мертвые AND души` | 526 | 10 | 0.524 | 0.578 | 0.689 | 2.061 | 1730 |  |
| `bad_query` | `(князь AND` | 0 | 0 | 0.000 | 0.000 | 0.000 | 0.000 | 0 | unexpected end of query |
