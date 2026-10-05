### failover

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| prefix | 19 ms | 341 ms | 1317 ms | 140.7 | 2252 | 94.8% | 1.38 | 6/1536 |

- `prefix` router index: predicted hit 92.5% vs actual 94.8%; 1455 requests expected warm, 0.6% stale
- `prefix` route reasons: prefix_hit=1454, hash_affinity=53, load_fallback=22, prefix_spill=1
- `prefix` errors: stream truncated=6

Per-second timeline (requests *started* in that second):

| t (s) | ok | err | TTFT p50 |
|---:|---:|---:|---:|
| 0 | 57 | 0 | 1007 ms |
| 1 | 110 | 6 | 24 ms |
| 2 | 113 | 0 | 267 ms |
| 3 | 192 | 0 | 19 ms |
| 4 | 207 | 0 | 17 ms |
| 5 | 150 | 0 | 32 ms |
| 6 | 149 | 0 | 21 ms |
| 7 | 127 | 0 | 20 ms |
| 8 | 155 | 0 | 13 ms |
| 9 | 199 | 0 | 13 ms |
| 10 | 71 | 0 | 13 ms |

### hot_tenant

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 586 ms | 1441 ms | 1814 ms | 46.1 | 738 | 79.2% | 1.00 | 0/512 |
| least_loaded | 270 ms | 1156 ms | 1642 ms | 66.8 (1.45x) | 1068 | 85.9% | 1.30 | 0/512 |
| prefix_pure | 198 ms | 676 ms | 1198 ms | 97.6 (2.11x) | 1561 | 94.6% | 2.47 | 0/512 |
| prefix | 18 ms | 590 ms | 1366 ms | 110.7 (2.40x) | 1772 | 92.9% | 1.13 | 0/512 |
| weighted | 19 ms | 685 ms | 1643 ms | 111.3 (2.41x) | 1781 | 92.7% | 1.20 | 0/512 |
| weighted_load | 26 ms | 909 ms | 1449 ms | 86.5 (1.87x) | 1384 | 89.6% | 1.06 | 0/512 |

- `prefix_pure` router index: predicted hit 87.1% vs actual 94.6%; 457 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 85.6% vs actual 92.9%; 449 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 86.4% vs actual 92.7%; 464 requests expected warm, 0.0% stale
- `weighted_load` router index: predicted hit 83.3% vs actual 89.6%; 464 requests expected warm, 0.0% stale
- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=457, hash_affinity=55
- `prefix` route reasons: prefix_hit=449, hash_affinity=39, load_fallback=24
- `weighted` route reasons: weighted_hit=454, weighted_cold=58
- `weighted_load` route reasons: weighted_hit=436, weighted_cold=76

### multi_tenant

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 1154 ms | 2543 ms | 3185 ms | 26.5 | 424 | 65.0% | 1.00 | 0/512 |
| least_loaded | 365 ms | 1763 ms | 2842 ms | 49.8 (1.88x) | 797 | 80.0% | 1.27 | 0/512 |
| prefix_pure | 14 ms | 526 ms | 1641 ms | 136.2 (5.14x) | 2179 | 94.6% | 1.22 | 0/512 |
| prefix | 15 ms | 561 ms | 1638 ms | 113.5 (4.29x) | 1817 | 93.6% | 1.23 | 0/512 |
| weighted | 18 ms | 1152 ms | 2640 ms | 94.9 (3.58x) | 1519 | 90.9% | 1.20 | 0/512 |
| weighted_load | 258 ms | 2047 ms | 3148 ms | 46.7 (1.76x) | 748 | 79.9% | 1.27 | 0/512 |

- `prefix_pure` router index: predicted hit 88.3% vs actual 94.6%; 464 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 86.9% vs actual 93.6%; 456 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 88.0% vs actual 90.9%; 464 requests expected warm, 0.2% stale
- `weighted_load` router index: predicted hit 84.5% vs actual 79.9%; 464 requests expected warm, 8.6% stale
- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=464, hash_affinity=48
- `prefix` route reasons: prefix_hit=456, hash_affinity=48, load_fallback=8
- `weighted` route reasons: weighted_hit=464, weighted_cold=48
- `weighted_load` route reasons: weighted_hit=443, weighted_cold=69

