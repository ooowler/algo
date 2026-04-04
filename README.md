# Лабораторная работа: хэш-структуры на Go

Реализованы **файловая хэш-таблица** на диске (бакеты-файлы), **идеальное хэширование (CHD)** для фиксированного набора ключей и **LSH** для поиска близких точек в **ℝ³**.

---

## Зависимость от N, память, сравнение LSH и наивного метода

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

### Рис. 11–12. Диск: задержка Set и Get (нс/op ± σ)

<p align="center"><img src="./figures/disk_set_latency_cloud.png" alt="Disk Set latency" width="780"/></p>

<p align="center"><img src="./figures/disk_get_latency_cloud.png" alt="Disk Get latency" width="780"/></p>

---

## Таблицы: дисковая таблица (нс/op ± σ)

### Таблица 1. Set

| N | нс/op | B/op | ~ op/с |
|---|-------|------|--------|
| 128 | 78 792 ± 7 003 | 1758 | 12 692 |
| 1024 | 107 298 ± 46 954 | 1960 | 9 320 |
| 8000 | 105 878 ± 48 303 | 2329 | 9 445 |
| 32000 | 94 917 ± 30 453 | 2462 | 10 536 |
| 65536 | 88 985 ± 12 290 | 2421 | 11 238 |

### Таблица 2. Get

| N | нс/op | B/op | ~ op/с |
|---|-------|------|--------|
| 128 | 14 703 ± 5 805 | 1471 | 68 016 |
| 1024 | 12 452 ± 1 121 | 1592 | 80 312 |
| 8000 | 15 217 ± 2 186 | 5799 | 65 718 |
| 32000 | 27 992 ± 1 177 | 22690 | 35 725 |
| 65536 | 34 317 ± 30 | 43942 | 29 140 |

Текстовая версия таблиц: `results/disk_bench_table.md`.
