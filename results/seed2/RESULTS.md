### failover

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| prefix | 34 ms | 567 ms | 1788 ms | 12.4 ms | 43.3 ms | 117.3 | 103.0 (88%) | 1877 | 93.5% | 1.33 | 0/1536 |

- `prefix` router index: predicted hit 92.0% vs actual 93.5%; 1453 requests expected warm, 1.7% stale
- `prefix` route reasons: prefix_hit=1451, hash_affinity=53, load_fallback=30, prefix_spill=2

Per-second timeline (requests *started* in that second):

| t (s) | ok | err | TTFT p50 |
|---:|---:|---:|---:|
| 0 | 76 | 0 | 830 ms |
| 1 | 65 | 0 | 297 ms |
| 2 | 111 | 0 | 59 ms |
| 3 | 65 | 0 | 329 ms |
| 4 | 143 | 0 | 26 ms |
| 5 | 128 | 0 | 62 ms |
| 6 | 116 | 0 | 22 ms |
| 7 | 89 | 0 | 203 ms |
| 8 | 149 | 0 | 22 ms |
| 9 | 193 | 0 | 22 ms |
| 10 | 131 | 0 | 32 ms |
| 11 | 168 | 0 | 23 ms |
| 12 | 102 | 0 | 14 ms |

### hot_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 617 ms | 1556 ms | 2092 ms | 12.1 ms | 42.1 ms | 48.5 | 22.2 (46%) | 775 | 80.9% | 1.00 | 0/512 |
| least_loaded | 272 ms | 901 ms | 2215 ms | 12.2 ms | 36.5 ms | 77.2 (1.59x) | 55.9 (72%) | 1235 | 88.0% | 1.10 | 0/512 |
| prefix_pure | 163 ms | 422 ms | 1250 ms | 14.9 ms | 41.0 ms | 100.6 (2.07x) | 91.5 (91%) | 1609 | 94.8% | 2.75 | 0/512 |
| prefix | 20 ms | 647 ms | 1719 ms | 12.5 ms | 45.0 ms | 112.0 (2.31x) | 96.3 (86%) | 1792 | 93.4% | 1.41 | 0/512 |
| weighted | 41 ms | 839 ms | 2007 ms | 12.3 ms | 89.6 ms | 95.5 (1.97x) | 83.6 (88%) | 1528 | 92.5% | 1.14 | 0/512 |
| weighted_load | 69 ms | 901 ms | 1938 ms | 12.2 ms | 56.0 ms | 85.9 (1.77x) | 64.2 (75%) | 1374 | 89.8% | 1.38 | 0/512 |

- `prefix_pure` router index: predicted hit 87.8% vs actual 94.8%; 461 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 86.5% vs actual 93.4%; 454 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 87.4% vs actual 92.5%; 464 requests expected warm, 0.0% stale
- `weighted_load` router index: predicted hit 84.7% vs actual 89.8%; 464 requests expected warm, 0.2% stale
- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=461, hash_affinity=51
- `prefix` route reasons: prefix_hit=453, hash_affinity=33, load_fallback=25, prefix_spill=1
- `weighted` route reasons: weighted_hit=461, weighted_cold=51
- `weighted_load` route reasons: weighted_hit=445, weighted_cold=67

### multi_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 1139 ms | 2783 ms | 3482 ms | 12.6 ms | 30.7 ms | 27.8 | 9.4 (34%) | 446 | 66.3% | 1.00 | 0/512 |
| least_loaded | 575 ms | 1889 ms | 2542 ms | 12.3 ms | 28.4 ms | 42.5 (1.53x) | 18.6 (44%) | 680 | 76.1% | 1.16 | 0/512 |
| prefix_pure | 66 ms | 698 ms | 1665 ms | 12.7 ms | 50.0 ms | 100.3 (3.60x) | 87.0 (87%) | 1605 | 94.6% | 1.81 | 0/512 |
| prefix | 24 ms | 616 ms | 1649 ms | 12.4 ms | 47.1 ms | 108.0 (3.88x) | 92.2 (85%) | 1728 | 93.0% | 1.25 | 0/512 |
| weighted | 34 ms | 1138 ms | 2534 ms | 12.1 ms | 55.9 ms | 86.8 (3.12x) | 69.8 (80%) | 1388 | 90.1% | 1.09 | 0/512 |
| weighted_load | 273 ms | 1351 ms | 2622 ms | 12.3 ms | 29.1 ms | 65.3 (2.34x) | 41.0 (63%) | 1044 | 85.6% | 1.09 | 0/512 |

- `prefix_pure` router index: predicted hit 86.7% vs actual 94.6%; 455 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 85.5% vs actual 93.0%; 449 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 86.6% vs actual 90.1%; 464 requests expected warm, 0.6% stale
- `weighted_load` router index: predicted hit 84.1% vs actual 85.6%; 464 requests expected warm, 2.6% stale
- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=455, hash_affinity=57
- `prefix` route reasons: prefix_hit=447, hash_affinity=52, load_fallback=11, prefix_spill=2
- `weighted` route reasons: weighted_hit=456, weighted_cold=56
- `weighted_load` route reasons: weighted_hit=441, weighted_cold=71

