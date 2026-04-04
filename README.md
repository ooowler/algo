# Лабораторная работа: хэш-структуры на Go

Реализованы три компонента: **файловая хэш-таблица** с бакетами на диске (вставка, изменение, удаление), **идеальное хэширование (CHD)** по фиксированному множеству ключей, **LSH** для поиска близких дублей среди точек в **ℝ³** (построение индекса, добавление точек, полный проход по индексу). Дополнительно: модульные тесты, бенчмарки, замеры через `pprof`, графики зависимостей от размера входа **N** (время и память по данным `go test -benchmem`).

## Запуск

```bash
go test ./... -timeout 300s -count=1
make profile
```

Построение иллюстраций к отчёту (matplotlib, те же данные, что и у бенчмарка `BenchmarkGrowth`):

```bash
python3 -m venv .venv && .venv/bin/pip install -r scripts/requirements.txt
.venv/bin/python scripts/build_charts_matplotlib.py
```

либо `make charts-mpl` (использует `.venv`, если он создан). Результат — файлы `figures/growth_*.png`.

## Структура каталогов

| Каталог / файл | Содержание |
|----------------|------------|
| `hashtable/` | дисковая таблица, файл бакета |
| `perfecthash/` | построение индекса и поиск |
| `lsh/` | индекс, `Add`, `FindDuplicates` |
| `cmd/profile/` | съём CPU и heap-профилей |
| `scripts/build_charts_matplotlib.py` | воспроизведение графиков |
| `figures/` | PNG для раздела ниже |

## Профилирование

```bash
make profile
go tool pprof -top profiles/hashtable_large_cpu.prof
go tool pprof -top profiles/lsh_large_mem.prof
```

Планировщик и блокировки: `make trace-lsh`, затем `go tool trace profiles/lsh_trace_medium.out`.

---

## Результаты замеров (графики)

Зависимости построены по выводу `go test` с `-bench=BenchmarkGrowth` и `-benchmem`: по оси абсцисс — **N**, по оси ординат — время на операцию или аллокации (**KiB/op**). Где нужно, ось **N** логарифмическая.

**Важно для просмотра в редакторе:** откройте **корень этой работы** (каталог, в котором лежат и `README.md`, и папка `figures/`). Иначе превью Markdown может не подставить локальные PNG.

### Рис. 1–2. Perfect hash: построение индекса

<p align="center"><img src="./figures/growth_ph_build_time.png" alt="Время построения CHD vs N" width="780"/></p>

*Рис. 1. Время одной операции `Build` в зависимости от числа ключей.*

<p align="center"><img src="./figures/growth_ph_build_mem.png" alt="Память при Build vs N" width="780"/></p>

*Рис. 2. Средние аллокации на итерацию (`B/op`) при построении индекса.*

### Рис. 3–4. Perfect hash: поиск

<p align="center"><img src="./figures/growth_ph_get_time.png" alt="Время Get vs N" width="780"/></p>

*Рис. 3. Время операции `Get`.*

<p align="center"><img src="./figures/growth_ph_get_mem.png" alt="Память Get vs N" width="780"/></p>

*Рис. 4. Аллокации на операцию `Get` (часто близки к нулю).*

### Рис. 5–8. LSH и наивный полный перебор пар

<p align="center"><img src="./figures/growth_lsh_find_time.png" alt="LSH время" width="780"/></p>

*Рис. 5. Время `FindDuplicates` для LSH.*

<p align="center"><img src="./figures/growth_lsh_find_mem.png" alt="LSH память" width="780"/></p>

*Рис. 6. Память на итерацию LSH.*

<p align="center"><img src="./figures/growth_naive_time.png" alt="Наивный алгоритм время" width="780"/></p>

*Рис. 7. Время наивного скана O(n²).*

<p align="center"><img src="./figures/growth_naive_mem.png" alt="Наивный алгоритм память" width="780"/></p>

*Рис. 8. Память наивного скана.*

### Рис. 9. Сравнение LSH и наивного метода на общих N

<p align="center"><img src="./figures/growth_dup_compare_time.png" alt="LSH vs naive время" width="780"/></p>

*Рис. 9. Сопоставление времени полного прохода (логарифмические оси).*

### Рис. 10. Файловая таблица: Set и Get

<p align="center"><img src="./figures/growth_disk_time.png" alt="Диск Set Get" width="780"/></p>

*Рис. 10. Время операций при росте числа ключей в сценарии бенчмарка (см. `hashtable/bench_growth_test.go`).*

---

## Дополнительно

Сводные бенчмарки: `make bench`, `make bench-scale`, `make bench-growth`. Каталоги `profiles/*.prof` и `results/` при работе появляются локально и в репозиторий обычно не включаются.
