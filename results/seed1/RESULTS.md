### failover

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| prefix | 24 ms | 327 ms | 1373 ms | 12.5 ms | 51.5 ms | 148.1 | 139.7 (94%) | 2369 | 95.5% | 1.35 | 4/1536 |

- `prefix` router index: predicted hit 92.8% vs actual 95.5%; 1462 requests expected warm, 0.1% stale
- `prefix` route reasons: prefix_hit=1461, hash_affinity=53, load_fallback=17, prefix_spill=1
- `prefix` errors: stream truncated=4

Per-second timeline (requests *started* in that second):

| t (s) | ok | err | TTFT p50 |
|---:|---:|---:|---:|
| 0 | 61 | 0 | 1004 ms |
| 1 | 111 | 4 | 34 ms |
| 2 | 104 | 0 | 141 ms |
| 3 | 197 | 0 | 30 ms |
| 4 | 146 | 0 | 59 ms |
| 5 | 216 | 0 | 18 ms |
| 6 | 159 | 0 | 50 ms |
| 7 | 145 | 0 | 18 ms |
| 8 | 195 | 0 | 17 ms |
| 9 | 187 | 0 | 16 ms |
| 10 | 11 | 0 | 13 ms |

### hot_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 649 ms | 1586 ms | 1964 ms | 12.4 ms | 35.8 ms | 44.5 | 19.1 (43%) | 712 | 78.5% | 1.00 | 0/512 |
| least_loaded | 313 ms | 960 ms | 1625 ms | 12.3 ms | 41.6 ms | 72.1 (1.62x) | 48.1 (67%) | 1153 | 87.1% | 1.29 | 0/512 |
| prefix_pure | 201 ms | 669 ms | 1205 ms | 14.7 ms | 41.4 ms | 96.0 (2.16x) | 82.3 (86%) | 1536 | 94.6% | 2.47 | 0/512 |
| prefix | 32 ms | 440 ms | 1342 ms | 12.4 ms | 40.2 ms | 121.0 (2.72x) | 109.4 (90%) | 1936 | 93.7% | 1.11 | 0/512 |
| weighted | 27 ms | 672 ms | 1635 ms | 12.2 ms | 36.9 ms | 109.6 (2.47x) | 90.4 (82%) | 1754 | 92.3% | 1.09 | 0/512 |
| weighted_load | 125 ms | 1030 ms | 1913 ms | 12.2 ms | 50.2 ms | 78.7 (1.77x) | 58.7 (75%) | 1259 | 89.1% | 1.20 | 0/512 |

- `prefix_pure` router index: predicted hit 87.1% vs actual 94.6%; 457 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 86.3% vs actual 93.7%; 453 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 86.3% vs actual 92.3%; 464 requests expected warm, 0.0% stale
- `weighted_load` router index: predicted hit 82.8% vs actual 89.1%; 464 requests expected warm, 0.4% stale
- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=457, hash_affinity=55
- `prefix` route reasons: prefix_hit=452, hash_affinity=39, load_fallback=20, prefix_spill=1
- `weighted` route reasons: weighted_hit=454, weighted_cold=58
- `weighted_load` route reasons: weighted_hit=434, weighted_cold=78

### multi_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 1451 ms | 2415 ms | 3133 ms | 12.6 ms | 20.2 ms | 26.1 | 6.5 (25%) | 418 | 63.7% | 1.00 | 0/512 |
| least_loaded | 573 ms | 1622 ms | 2645 ms | 12.3 ms | 31.9 ms | 47.1 (1.80x) | 19.9 (42%) | 753 | 78.1% | 1.08 | 0/512 |
| prefix_pure | 18 ms | 564 ms | 1636 ms | 12.3 ms | 39.8 ms | 131.2 (5.03x) | 116.3 (89%) | 2099 | 94.6% | 1.22 | 0/512 |
| prefix | 21 ms | 625 ms | 1632 ms | 12.2 ms | 44.5 ms | 111.1 (4.26x) | 97.6 (88%) | 1777 | 93.6% | 1.22 | 0/512 |
| weighted | 27 ms | 1272 ms | 2552 ms | 12.1 ms | 54.5 ms | 92.8 (3.56x) | 78.5 (85%) | 1485 | 90.7% | 1.14 | 0/512 |
| weighted_load | 253 ms | 1606 ms | 2841 ms | 12.3 ms | 29.8 ms | 63.1 (2.42x) | 42.4 (67%) | 1009 | 84.7% | 1.39 | 0/512 |

- `prefix_pure` router index: predicted hit 88.3% vs actual 94.6%; 464 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 86.9% vs actual 93.6%; 456 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 88.0% vs actual 90.7%; 464 requests expected warm, 0.2% stale
- `weighted_load` router index: predicted hit 84.4% vs actual 84.7%; 464 requests expected warm, 3.7% stale
- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=464, hash_affinity=48
- `prefix` route reasons: prefix_hit=456, hash_affinity=48, load_fallback=8
- `weighted` route reasons: weighted_hit=464, weighted_cold=48
- `weighted_load` route reasons: weighted_hit=442, weighted_cold=70

