### hot_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 616 ms | 1135 ms | 1731 ms | 12.5 ms | 14.6 ms | 52.9 | 21.8 (41%) | 847 | 81.9% | 1.00 | 0/512 |
| random | 541 ms | 1531 ms | 2357 ms | 12.5 ms | 15.5 ms | 45.4 (0.86x) | 22.1 (49%) | 727 | 80.7% | 1.05 | 0/512 |
| p2c | 555 ms | 1252 ms | 1913 ms | 12.5 ms | 13.7 ms | 52.8 (1.00x) | 25.1 (47%) | 845 | 81.7% | 1.12 | 0/512 |
| least_loaded | 249 ms | 879 ms | 1920 ms | 12.4 ms | 13.4 ms | 81.6 (1.54x) | 60.5 (74%) | 1305 | 88.6% | 1.19 | 0/512 |
| prefix | 20 ms | 653 ms | 1649 ms | 12.9 ms | 14.1 ms | 105.6 (2.00x) | 93.7 (89%) | 1690 | 92.8% | 1.43 | 0/512 |

- `prefix` router index: predicted hit 86.1% vs actual 92.8%; 452 requests expected warm, 0.0% stale
- `round_robin` route reasons: round_robin=512
- `random` route reasons: random=512
- `p2c` route reasons: p2c=512
- `least_loaded` route reasons: least_loaded=512
- `prefix` route reasons: prefix_hit=446, hash_affinity=31, load_fallback=29, prefix_spill=6

### multi_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 840 ms | 3745 ms | 4607 ms | 12.0 ms | 16.0 ms | 25.6 | 8.9 (35%) | 410 | 65.0% | 1.00 | 0/512 |
| random | 1146 ms | 3039 ms | 4199 ms | 12.4 ms | 15.1 ms | 25.0 (0.97x) | 7.9 (31%) | 400 | 63.6% | 1.07 | 0/512 |
| p2c | 1000 ms | 2023 ms | 2689 ms | 12.4 ms | 13.4 ms | 32.9 (1.28x) | 5.1 (16%) | 526 | 66.7% | 1.09 | 0/512 |
| least_loaded | 298 ms | 1467 ms | 2534 ms | 12.4 ms | 13.8 ms | 57.9 (2.26x) | 37.8 (65%) | 926 | 83.0% | 1.03 | 0/512 |
| prefix | 17 ms | 803 ms | 1941 ms | 12.5 ms | 14.0 ms | 109.8 (4.28x) | 92.8 (85%) | 1756 | 92.7% | 1.12 | 0/512 |

- `prefix` router index: predicted hit 86.1% vs actual 92.7%; 452 requests expected warm, 0.0% stale
- `round_robin` route reasons: round_robin=512
- `random` route reasons: random=512
- `p2c` route reasons: p2c=512
- `least_loaded` route reasons: least_loaded=512
- `prefix` route reasons: prefix_hit=451, hash_affinity=41, load_fallback=19, prefix_spill=1

