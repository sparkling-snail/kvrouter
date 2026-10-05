### failover

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| prefix | 21 ms | 583 ms | 1650 ms | 131.3 | 2101 | 94.4% | 1.34 | 2/1530 |

- `prefix` router index: predicted hit 92.4% vs actual 94.4%; 1452 requests expected warm, 0.7% stale
- `prefix` route reasons: prefix_hit=1450, hash_affinity=47, load_fallback=29, prefix_spill=2
- `prefix` errors: stream truncated=2

Per-second timeline (requests *started* in that second):

| t (s) | ok | err | TTFT p50 |
|---:|---:|---:|---:|
| 0 | 56 | 2 | 1266 ms |
| 1 | 65 | 0 | 568 ms |
| 2 | 82 | 0 | 47 ms |
| 3 | 115 | 0 | 28 ms |
| 4 | 209 | 0 | 16 ms |
| 5 | 210 | 0 | 20 ms |
| 6 | 194 | 0 | 17 ms |
| 7 | 135 | 0 | 26 ms |
| 8 | 125 | 0 | 55 ms |
| 9 | 133 | 0 | 18 ms |
| 10 | 165 | 0 | 16 ms |
| 11 | 39 | 0 | 13 ms |

### hot_tenant

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 646 ms | 1429 ms | 1930 ms | 50.9 | 815 | 80.8% | 1.00 | 0/512 |
| least_loaded | 312 ms | 959 ms | 1942 ms | 68.4 (1.34x) | 1094 | 85.9% | 1.17 | 0/512 |
| prefix_pure | 269 ms | 790 ms | 2120 ms | 75.2 (1.48x) | 1204 | 94.4% | 3.00 | 0/512 |
| prefix | 16 ms | 649 ms | 1660 ms | 111.1 (2.18x) | 1778 | 92.8% | 1.38 | 0/512 |
| weighted | 15 ms | 662 ms | 1931 ms | 112.3 (2.21x) | 1797 | 92.5% | 1.36 | 0/512 |
| weighted_load | 25 ms | 969 ms | 1751 ms | 84.6 (1.66x) | 1353 | 89.9% | 1.13 | 0/512 |

- `prefix_pure` router index: predicted hit 87.6% vs actual 94.4%; 460 requests expected warm, 0.2% stale
- `prefix` router index: predicted hit 86.4% vs actual 92.8%; 454 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 87.1% vs actual 92.5%; 464 requests expected warm, 0.0% stale
- `weighted_load` router index: predicted hit 84.3% vs actual 89.9%; 464 requests expected warm, 0.0% stale
- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=460, hash_affinity=52
- `prefix` route reasons: prefix_hit=450, hash_affinity=35, load_fallback=23, prefix_spill=4
- `weighted` route reasons: weighted_hit=458, weighted_cold=54
- `weighted_load` route reasons: weighted_hit=442, weighted_cold=70

### multi_tenant

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 1152 ms | 2535 ms | 3055 ms | 27.5 | 439 | 64.2% | 1.00 | 0/512 |
| least_loaded | 548 ms | 1774 ms | 3143 ms | 45.1 (1.64x) | 721 | 78.3% | 1.12 | 0/512 |
| prefix_pure | 24 ms | 661 ms | 1738 ms | 113.4 (4.13x) | 1815 | 94.6% | 1.50 | 0/512 |
| prefix | 18 ms | 829 ms | 2249 ms | 107.4 (3.91x) | 1719 | 92.7% | 1.14 | 0/512 |
| weighted | 19 ms | 1154 ms | 2546 ms | 97.4 (3.55x) | 1559 | 91.0% | 1.09 | 0/512 |
| weighted_load | 260 ms | 1277 ms | 2522 ms | 61.8 (2.25x) | 989 | 84.8% | 1.20 | 0/512 |

- `prefix_pure` router index: predicted hit 87.6% vs actual 94.6%; 460 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 85.6% vs actual 92.7%; 449 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 87.7% vs actual 91.0%; 464 requests expected warm, 0.0% stale
- `weighted_load` router index: predicted hit 81.7% vs actual 84.8%; 464 requests expected warm, 2.6% stale
- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=460, hash_affinity=52
- `prefix` route reasons: prefix_hit=448, hash_affinity=41, load_fallback=22, prefix_spill=1
- `weighted` route reasons: weighted_hit=462, weighted_cold=50
- `weighted_load` route reasons: weighted_hit=427, weighted_cold=85

