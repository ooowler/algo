# Лабораторная работа 5: inverted index search

## Состав

| Часть | Реализация |
|---|---|
| Индекс | координатный inverted index: `term -> docID, tf, positions` |
| Boolean | `AND`, `OR`, `NOT`, скобки, неявный `AND` перед `NOT` |
| Positional | `ADJ`, `NEAR`, `NEAR/N` |
| Ускорение | merge по отсортированным posting list-ам + skip step в `advanceTo` |
| Ranking | BM25 по положительным термам запроса |
| Disk | один mmap-файл с JSON-метаданными и сжатыми postings |
| Compression | delta encoding + PForDelta-style block bitpacking по 128 чисел |
| UI | HTTP-страница + `/api/search`, выводит title, score, snippet |

## Быстрый запуск

```bash
make test
make demo
go run ./cmd/lab5 search -index data/demo.lab5 -q 'history AND NOT (russia AND china)'
make server
```

UI после `make server`: `http://127.0.0.1:18080`.

## Загрузка документов

Поддерживаются:

- JSONL: одна строка - `{"id":1,"title":"...","text":"..."}`
- директория с `.txt`/`.md`, где имя файла становится заголовком

```bash
go run ./cmd/lab5 build -docs data/docs.jsonl -index data/index.lab5
go run ./cmd/server -index data/index.lab5
```

Так можно загрузить подготовленные страницы Wikipedia после конвертации в JSONL.

## Запросы

Примеры:

```text
search AND index
history AND NOT (russia AND china)
history NOT (russia AND china)
quick ADJ brown
russia NEAR/3 china
alpha OR gamma OR omega
```

`ANT` принят как алиас `AND`, `EDGE` принят как алиас `ADJ`, потому что в голосовой расшифровке эти операторы могли быть распознаны так.

## Формат mmap-индекса

Файл начинается с `LAB5IDX1`, затем идет offset JSON-метаданных. Секция postings хранится до JSON:

1. `docID` как delta от прошлого `docID`;
2. `tf` отдельной последовательностью;
3. позиции как delta внутри документа;
4. каждая последовательность пакуется блоками до 128 `uint32`;
5. ширина блока равна `bits.Len32(max(block))`, значения пишутся bitpacking-ом.

При поиске mmap-ится весь файл, а декодируются только posting list-ы термов текущего запроса.

## Бенчмарки и профиль

```bash
make bench
make profile
```

Что смотреть на защите:

- `BenchmarkQueriesMemory`: операции в памяти, включая `AND/OR/NOT/ADJ/NEAR`;
- `BenchmarkQueriesDiskMmap`: те же запросы через mmap-индекс;
- `BenchmarkCompressionBaseline`: `raw_B`, `compressed_B`, `ratio`;
- `results/profile_cpu.txt`, `results/profile_mem.txt` после `make profile`.

Бенчмарки используют синтетический корпус с разной частотностью термов, чтобы не сравнивать только один популярный term с редким.
