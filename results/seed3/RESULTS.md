### failover

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| prefix | 27 ms | 380 ms | 1633 ms | 12.6 ms | 47.0 ms | 131.3 | 120.5 (92%) | 2101 | 94.8% | 1.36 | 4/1524 |

- `prefix` router index: predicted hit 93.0% vs actual 94.8%; 1453 requests expected warm, 0.6% stale
- `prefix` route reasons: prefix_hit=1450, hash_affinity=43, load_fallback=24, prefix_spill=3
- `prefix` errors: stream truncated=4

Per-second timeline (requests *started* in that second):

| t (s) | ok | err | TTFT p50 |
|---:|---:|---:|---:|
| 0 | 54 | 4 | 1266 ms |
| 1 | 63 | 0 | 368 ms |
| 2 | 80 | 0 | 50 ms |
| 3 | 125 | 0 | 38 ms |
| 4 | 154 | 0 | 26 ms |
| 5 | 183 | 0 | 27 ms |
| 6 | 174 | 0 | 26 ms |
| 7 | 135 | 0 | 24 ms |
| 8 | 167 | 0 | 15 ms |
| 9 | 190 | 0 | 26 ms |
| 10 | 172 | 0 | 17 ms |
| 11 | 23 | 0 | 13 ms |

### hot_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 473 ms | 1310 ms | 1816 ms | 12.3 ms | 36.0 ms | 56.0 | 29.1 (52%) | 896 | 82.5% | 1.00 | 0/512 |
| least_loaded | 288 ms | 1094 ms | 1924 ms | 12.2 ms | 48.0 ms | 67.4 (1.20x) | 48.2 (71%) | 1078 | 86.4% | 1.19 | 0/512 |
| prefix_pure | 260 ms | 760 ms | 2109 ms | 15.0 ms | 40.6 ms | 78.0 (1.39x) | 66.6 (85%) | 1248 | 94.6% | 3.00 | 0/512 |
| prefix | 25 ms | 647 ms | 1737 ms | 12.6 ms | 45.1 ms | 105.0 (1.88x) | 90.5 (86%) | 1681 | 92.7% | 1.27 | 0/512 |
| weighted | 24 ms | 639 ms | 1932 ms | 12.2 ms | 33.1 ms | 110.8 (1.98x) | 94.1 (85%) | 1773 | 92.7% | 1.25 | 0/512 |
| weighted_load | 25 ms | 964 ms | 1675 ms | 12.2 ms | 41.7 ms | 88.0 (1.57x) | 66.5 (76%) | 1408 | 90.0% | 1.02 | 0/512 |

- `prefix_pure` router index: predicted hit 87.6% vs actual 94.6%; 460 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 86.3% vs actual 92.7%; 453 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 87.3% vs actual 92.7%; 464 requests expected warm, 0.0% stale
- `weighted_load` router index: predicted hit 84.5% vs actual 90.0%; 464 requests expected warm, 0.2% stale
- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=460, hash_affinity=52
- `prefix` route reasons: prefix_hit=451, hash_affinity=35, load_fallback=24, prefix_spill=2
- `weighted` route reasons: weighted_hit=460, weighted_cold=52
- `weighted_load` route reasons: weighted_hit=443, weighted_cold=69

### multi_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 867 ms | 3651 ms | 5493 ms | 12.4 ms | 27.1 ms | 24.1 | 7.7 (32%) | 385 | 64.4% | 1.00 | 0/512 |
| least_loaded | 555 ms | 1765 ms | 2546 ms | 12.4 ms | 28.3 ms | 45.6 (1.89x) | 20.7 (45%) | 729 | 77.8% | 1.30 | 0/512 |
| prefix_pure | 28 ms | 651 ms | 1777 ms | 12.8 ms | 39.3 ms | 111.7 (4.64x) | 99.2 (89%) | 1787 | 94.6% | 1.50 | 0/512 |
| prefix | 26 ms | 676 ms | 1721 ms | 12.3 ms | 44.5 ms | 106.8 (4.43x) | 90.5 (85%) | 1709 | 92.7% | 1.11 | 0/512 |
| weighted | 49 ms | 1332 ms | 2914 ms | 12.2 ms | 52.1 ms | 81.1 (3.37x) | 68.8 (85%) | 1298 | 89.5% | 1.24 | 0/512 |
| weighted_load | 244 ms | 1280 ms | 2529 ms | 12.3 ms | 51.4 ms | 62.9 (2.61x) | 42.0 (67%) | 1007 | 85.1% | 1.11 | 0/512 |

- `prefix_pure` router index: predicted hit 87.2% vs actual 94.6%; 458 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 86.3% vs actual 92.7%; 453 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 87.4% vs actual 89.5%; 464 requests expected warm, 0.6% stale
- `weighted_load` router index: predicted hit 82.5% vs actual 85.1%; 464 requests expected warm, 2.6% stale
- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=458, hash_affinity=54
- `prefix` route reasons: prefix_hit=452, hash_affinity=42, load_fallback=17, prefix_spill=1
- `weighted` route reasons: weighted_hit=461, weighted_cold=51
- `weighted_load` route reasons: weighted_hit=432, weighted_cold=80

