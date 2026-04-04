# Лабораторная работа: хэш-структуры на Go

Реализованы **файловая хэш-таблица** на диске (бакеты-файлы), **идеальное хэширование (CHD)** для фиксированного набора ключей и **LSH** для поиска близких точек в **ℝ³**. Для сравнения с типичными Java-отчётами: там CPU/allocation **flame graph** часто снимают **async-profiler**; в **Go** стандартный путь — **`pprof`** (`runtime/pprof`, `go tool pprof`), тот же смысл «где горит CPU / память», другой стек и UI.

---

## Flame graph (как у примеров с async-profiler)

Интерактивный **Flame Graph** встроен в веб-UI `pprof`. Для страницы `/flamegraph` внутри `go tool pprof` **нужен Graphviz** (команда `dot`):

- macOS: `brew install graphviz`
- Debian/Ubuntu: `sudo apt install graphviz`

Дальше:

```bash
go run ./cmd/profile -only=hashtable
./scripts/export_pprof_flame_html.sh profiles/hashtable_large_cpu.prof figures/pprof_hashtable_flamegraph.html
```

Откройте **`figures/pprof_hashtable_flamegraph.html`** в браузере — это тот же стиль навигации по стеку, что и у «красивых» flame в Java-примерах (ширина полосы ∝ доля времени).

Без сохранения файла: `go tool pprof -http=:8080 profiles/hashtable_large_cpu.prof` → в меню выбрать **Flame Graph**.

---

## Таблица и графики задержек (диск, нс/op ± σ, «облако»)

Скрипт несколько раз гоняет `BenchmarkGrowthDisk_Set` / `Get`, считает среднее и стандартное отклонение по прогонам, пишет таблицу и рисунки с **полосой ±1σ** и **errorbar** (аналог «облака» на примерах с matplotlib):

```bash
python3 -m venv .venv && .venv/bin/pip install -r scripts/requirements.txt
make bench-disk-cloud
```

Результат: `results/disk_bench_table.md`, `figures/disk_set_latency_cloud.png`, `figures/disk_get_latency_cloud.png`. Число прогонов и длительность: переменные окружения `BENCH_RUNS` (по умолчанию 5) и `BENCHTIME` (по умолчанию `250ms`).

---

## Рост N, память, сравнение LSH / наивный метод

```bash
make charts-mpl
```

Строит `figures/growth_*.png` по `BenchmarkGrowth` (как в методичке — зависимости от **N**). На всех этих графиках — зона ±σ по нескольким прогонам (`GROWTH_BENCH_COUNT`, по умолчанию 3; длительность — `GROWTH_BENCHTIME`, по умолчанию `150ms`). Для быстрой пересборки без разброса: `GROWTH_BENCH_COUNT=1`. Превью: откройте каталог работы, где лежат и `README.md`, и `figures/`.

### Рис. 1–2. Perfect hash: построение индекса

<p align="center"><img src="./figures/growth_ph_build_time.png" alt="Время Build vs N" width="780"/></p>

<p align="center"><img src="./figures/growth_ph_build_mem.png" alt="Память Build vs N" width="780"/></p>

### Рис. 3–4. Perfect hash: поиск

<p align="center"><img src="./figures/growth_ph_get_time.png" alt="Время Get vs N" width="780"/></p>

<p align="center"><img src="./figures/growth_ph_get_mem.png" alt="Память Get vs N" width="780"/></p>

### Рис. 5–8. LSH и наивный перебор

<p align="center"><img src="./figures/growth_lsh_find_time.png" alt="LSH время" width="780"/></p>

<p align="center"><img src="./figures/growth_lsh_find_mem.png" alt="LSH память" width="780"/></p>

<p align="center"><img src="./figures/growth_naive_time.png" alt="Наивный время" width="780"/></p>

<p align="center"><img src="./figures/growth_naive_mem.png" alt="Наивный память" width="780"/></p>

### Рис. 9. LSH vs наивный

<p align="center"><img src="./figures/growth_dup_compare_time.png" alt="LSH vs naive" width="780"/></p>

### Рис. 10. Диск: Set и Get (рост N)

<p align="center"><img src="./figures/growth_disk_time.png" alt="Диск Set Get" width="780"/></p>

### Рис. 11. Диск: задержка Set с разбросом (±σ)

<p align="center"><img src="./figures/disk_set_latency_cloud.png" alt="Disk Set latency cloud" width="780"/></p>

### Рис. 12. Диск: задержка Get с разбросом (±σ)

<p align="center"><img src="./figures/disk_get_latency_cloud.png" alt="Disk Get latency cloud" width="780"/></p>

Таблицы ниже — пример после `BENCH_RUNS=2` и `BENCHTIME=200ms` (см. также `results/disk_bench_table.md`). Для отчёта лучше `BENCH_RUNS=5` и те же графики пересобрать.

### Таблица 1. Диск: Set (нс/op ± σ)

| N | нс/op | B/op | ~ op/с |
|---|-------|------|--------|
| 128 | 78 792 ± 7 003 | 1758 | 12 692 |
| 1024 | 107 298 ± 46 954 | 1960 | 9 320 |
| 8000 | 105 878 ± 48 303 | 2329 | 9 445 |
| 32000 | 94 917 ± 30 453 | 2462 | 10 536 |
| 65536 | 88 985 ± 12 290 | 2421 | 11 238 |

### Таблица 2. Диск: Get (нс/op ± σ)

| N | нс/op | B/op | ~ op/с |
|---|-------|------|--------|
| 128 | 14 703 ± 5 805 | 1471 | 68 016 |
| 1024 | 12 452 ± 1 121 | 1592 | 80 312 |
| 8000 | 15 217 ± 2 186 | 5799 | 65 718 |
| 32000 | 27 992 ± 1 177 | 22690 | 35 725 |
| 65536 | 34 317 ± 30 | 43942 | 29 140 |

---

## Дополнительно

`make profile` — все CPU/heap `.prof`. `make figures-pprof` — столбчатая диаграмма по `pprof -top -cum` → `figures/pprof_hashtable_cpu_top.png`. `go test ./...` — тесты.
