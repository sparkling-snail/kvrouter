### failover

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| prefix | 21 ms | 323 ms | 1565 ms | 142.6 | 2281 | 95.0% | 1.29 | 6/1533 |

- `prefix` router index: predicted hit 92.3% vs actual 95.0%; 1448 requests expected warm, 0.4% stale
- `prefix` route reasons: prefix_hit=1447, hash_affinity=50, load_fallback=29, prefix_spill=1
- `prefix` errors: stream truncated=6

Per-second timeline (requests *started* in that second):

| t (s) | ok | err | TTFT p50 |
|---:|---:|---:|---:|
| 0 | 58 | 0 | 1004 ms |
| 1 | 119 | 6 | 23 ms |
| 2 | 107 | 0 | 57 ms |
| 3 | 63 | 0 | 191 ms |
| 4 | 205 | 0 | 28 ms |
| 5 | 216 | 0 | 17 ms |
| 6 | 222 | 0 | 17 ms |
| 7 | 143 | 0 | 34 ms |
| 8 | 185 | 0 | 19 ms |
| 9 | 179 | 0 | 14 ms |
| 10 | 30 | 0 | 14 ms |

### hot_tenant

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 510 ms | 1245 ms | 1816 ms | 55.3 | 885 | 82.8% | 1.00 | 0/512 |
| least_loaded | 262 ms | 924 ms | 2049 ms | 78.8 (1.42x) | 1260 | 88.2% | 1.17 | 0/512 |
| prefix_pure | 167 ms | 393 ms | 1268 ms | 100.1 (1.81x) | 1601 | 94.8% | 2.75 | 0/512 |
| prefix | 19 ms | 657 ms | 1728 ms | 119.1 (2.15x) | 1905 | 93.4% | 1.44 | 0/512 |
| weighted | 20 ms | 656 ms | 1901 ms | 115.9 (2.09x) | 1854 | 92.9% | 1.12 | 0/512 |
| weighted_load | 18 ms | 970 ms | 1921 ms | 82.4 (1.49x) | 1319 | 89.5% | 1.14 | 0/512 |

- `prefix_pure` router index: predicted hit 87.8% vs actual 94.8%; 461 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 86.5% vs actual 93.4%; 454 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 87.5% vs actual 92.9%; 464 requests expected warm, 0.0% stale
- `weighted_load` router index: predicted hit 84.1% vs actual 89.5%; 464 requests expected warm, 0.4% stale
- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=461, hash_affinity=51
- `prefix` route reasons: prefix_hit=453, hash_affinity=32, load_fallback=26, prefix_spill=1
- `weighted` route reasons: weighted_hit=461, weighted_cold=51
- `weighted_load` route reasons: weighted_hit=441, weighted_cold=71

### multi_tenant

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 954 ms | 2377 ms | 4021 ms | 27.9 | 446 | 67.5% | 1.00 | 0/512 |
| least_loaded | 435 ms | 1516 ms | 2841 ms | 48.2 (1.73x) | 771 | 79.6% | 1.08 | 0/512 |
| prefix_pure | 35 ms | 695 ms | 1668 ms | 102.3 (3.67x) | 1636 | 94.6% | 1.81 | 0/512 |
| prefix | 17 ms | 576 ms | 1646 ms | 117.9 (4.23x) | 1886 | 93.1% | 1.28 | 0/512 |
| weighted | 22 ms | 1141 ms | 2533 ms | 87.7 (3.15x) | 1403 | 89.9% | 1.10 | 0/512 |
| weighted_load | 186 ms | 1288 ms | 2250 ms | 62.7 (2.25x) | 1004 | 84.7% | 1.11 | 0/512 |

- `prefix_pure` router index: predicted hit 86.9% vs actual 94.6%; 456 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 85.6% vs actual 93.1%; 449 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 87.0% vs actual 89.9%; 464 requests expected warm, 0.6% stale
- `weighted_load` router index: predicted hit 82.0% vs actual 84.7%; 464 requests expected warm, 2.4% stale
- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=456, hash_affinity=56
- `prefix` route reasons: prefix_hit=449, hash_affinity=51, load_fallback=12
- `weighted` route reasons: weighted_hit=458, weighted_cold=54
- `weighted_load` route reasons: weighted_hit=428, weighted_cold=84

