# Лабораторная работа 5: inverted index search

## Состав

| Часть | Значение |
|---|---|
| Индекс | координатный inverted index: `term -> docID, tf, positions` |
| Boolean | `AND`, `OR`, `NOT`, скобки, неявный `AND` перед `NOT` |
| Positional | `ADJ`, `NEAR`, `NEAR/N` |
| Skip | `advanceTo` прыгает по posting list через `sqrt(len(list))` step |
| Ranking | BM25 по положительным термам запроса |
| Disk | один mmap-файл: postings + JSON metadata tail |
| Compression | docID delta + position delta + block bitpacking по 128 `uint32` |
| UI | title, score, snippet, elapsed time, L0 matches, регулируемый top-K |
| Stress | `cmd/stress`: latency, p95, QPS, размеры корпуса/индекса, CPU profile |

## Корпус

Загружен реальный публичный корпус русской классики из Lib.ru/az.lib.ru. Важно: `data/russian_classics.jsonl` хранит не исходные строки книг, а поисковые документы-фрагменты примерно по 900 рун. Поэтому строк в JSONL меньше, чем в HTML-исходниках: сейчас `11092` фрагмента против `89366` строк исходных HTML. Только 4 тома `Войны и мира` занимают `24898` строк HTML.

| Автор | Произведения |
|---|---|
| Л. Н. Толстой | `Война и мир` 4 тома, `Анна Каренина`, `Воскресение` |
| А. С. Пушкин | `Евгений Онегин`, `Дубровский`, `Пиковая дама`, `Капитанская дочка` |
| Ф. М. Достоевский | `Преступление и наказание`, `Идиот`, `Бесы`, `Братья Карамазовы` |
| Н. В. Гоголь | `Мертвые души`, `Тарас Бульба`, `Шинель`, `Ревизор` |
| А. П. Чехов | `Палата No 6`, `Дама с собачкой`, `Чайка`, `Дядя Ваня`, `Вишневый сад` |

Итоговые файлы:

```text
data/src_utf8/*.html
data/russian_classics.jsonl
data/russian_classics.lab5
```

## Метрики индекса

| Метрика | Значение |
|---|---:|
| HTML source files | 27 |
| HTML source lines | 89366 |
| War and Peace source lines | 24898 |
| JSONL documents/fragments | 11092 |
| JSONL corpus | 27893738 bytes / 26.60 MiB |
| mmap index | 46005657 bytes / 43.87 MiB |
| Terms | 143693 |
| Avg doc length | 218.62 |
| Raw postings bytes | 23438576 |
| Compressed postings bytes | 8323778 |
| Compression ratio | 0.355 |

## Sanity-check

| Проверка | Ожидание | Факт |
|---|---|---|
| `AND` | меньше L0, чем одиночный term | `князь AND андрей`: 457 docs |
| `OR` | объединяет posting lists | `пьер OR наташа`: 976 docs |
| `NOT` в связке | не строит лишний universe для `A AND NOT B` | `князь AND NOT (...)`: 1541 docs, ~0.69 ms |
| `ADJ` | строгая соседняя позиция | `пьер ADJ безухов`: 8 docs |
| `NEAR/N` | окно по координатам | `андрей NEAR/5 болконский`: 11 docs |
| Missing term | пустая выдача без падения | `несуществующийтерм`: 0 docs |
| Bad query | возвращает ошибку парсинга | `(князь AND`: `unexpected end of query` |
| Unicode | русский текст и snippets без битого UTF-8 | `мертвые AND души`: 526 docs |

## Stress

`cmd/stress` прогонял каждый запрос 1000 раз по mmap-индексу.

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
| `bad_query` | `(князь AND` | 0 | 0 | 0.000 | 0.000 | 0.000 | 0.000 | 0 | `unexpected end of query` |

Полный отчет: `results/stress.md`.

Итог по скорости: на корпусе `11092` документов и `143693` термов все рабочие сценарии поиска укладываются примерно в `0.12-0.68 ms` в среднем. Даже отрицательный запрос с `10635` L0 matches идет за `0.552 ms` avg / `0.772 ms` p95, а частые Boolean-запросы держатся около `1.4-1.7k QPS`. Сжатые postings занимают `35.5%` от сырого размера.

## Профиль CPU

`results/profile_cpu.txt` после stress-прогона:

| flat | flat% | cum | cum% | function |
|---:|---:|---:|---:|---|
| 620ms | 11.90% | 620ms | 11.90% | `runtime.decoderune` |
| 440ms | 8.45% | 440ms | 8.45% | `unicode.lookupCaseRange` |
| 360ms | 6.91% | 950ms | 18.23% | `runtime.stringtoslicerune` |
| 220ms | 4.22% | 220ms | 4.22% | `runtime.memclrNoHeapPointers` |
| 190ms | 3.65% | 280ms | 5.37% | `sort.partition_func` |
| 150ms | 2.88% | 150ms | 2.88% | `runtime.encoderune` |
| 140ms | 2.69% | 650ms | 12.48% | `lab5/search.decodePostings` |
| 130ms | 2.50% | 1090ms | 20.92% | `strings.Map` |

<p align="center"><img src="./figures/pprof_search_flamegraph.svg" width="920"/></p>

Вывод: на текущем корпусе bottleneck не в mmap и не в bitpacking decode, а в Unicode lowercase/snippet/ranking path.

## UI

HTTP-интерфейс показывает title, score, snippet, время поиска, L0 matches и top-K. Справа есть кликабельные примеры для частого/редкого терма, `AND`, `OR`, `NOT`, `ADJ`, `NEAR/N`, Unicode-запроса и пустой выдачи.

## Запуск

```bash
cd /Users/evgeniyforbes/study/2sem/algo/lab5
make report
make server
```

UI:

```text
http://127.0.0.1:18080
```

## Команды

Поиск:

```bash
go run ./cmd/lab5 search -index data/russian_classics.lab5 -q 'князь AND NOT (француз AND император)' -limit 10
go run ./cmd/lab5 search -index data/russian_classics.lab5 -q 'андрей NEAR/5 болконский' -limit 10
go run ./cmd/lab5 search -index data/russian_classics.lab5 -q 'пьер ADJ безухов' -limit 10
```

Stress + profile:

```bash
make stress
make profile
make flamegraph
```

API:

```bash
curl -fsSL 'http://127.0.0.1:18080/api/search?q=%D0%BA%D0%BD%D1%8F%D0%B7%D1%8C%20AND%20NOT%20%28%D1%84%D1%80%D0%B0%D0%BD%D1%86%D1%83%D0%B7%20AND%20%D0%B8%D0%BC%D0%BF%D0%B5%D1%80%D0%B0%D1%82%D0%BE%D1%80%29&limit=5'
```

`total_matches` в API - L0 hits после boolean/positional этапа до обрезки top-K.
