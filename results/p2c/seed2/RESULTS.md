### hot_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 499 ms | 1213 ms | 1915 ms | 12.4 ms | 15.1 ms | 54.9 | 27.6 (50%) | 879 | 82.6% | 1.00 | 0/512 |
| random | 473 ms | 1958 ms | 2533 ms | 12.2 ms | 16.1 ms | 48.2 (0.88x) | 24.5 (51%) | 770 | 82.1% | 1.05 | 0/512 |
| p2c | 456 ms | 1223 ms | 1923 ms | 12.4 ms | 13.6 ms | 58.1 (1.06x) | 30.7 (53%) | 929 | 82.4% | 1.18 | 0/512 |
| least_loaded | 32 ms | 845 ms | 1916 ms | 12.4 ms | 13.5 ms | 86.2 (1.57x) | 64.5 (75%) | 1380 | 89.7% | 1.07 | 0/512 |
| prefix | 17 ms | 676 ms | 1632 ms | 12.7 ms | 14.0 ms | 117.4 (2.14x) | 102.3 (87%) | 1878 | 93.4% | 1.45 | 0/512 |

- `prefix` router index: predicted hit 86.5% vs actual 93.4%; 454 requests expected warm, 0.0% stale
- `round_robin` route reasons: round_robin=512
- `random` route reasons: random=512
- `p2c` route reasons: p2c=512
- `least_loaded` route reasons: least_loaded=512
- `prefix` route reasons: prefix_hit=452, hash_affinity=32, load_fallback=26, prefix_spill=2

### multi_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 1227 ms | 2822 ms | 3678 ms | 12.3 ms | 14.7 ms | 28.3 | 9.7 (34%) | 453 | 63.6% | 1.00 | 0/512 |
| random | 911 ms | 3130 ms | 4071 ms | 12.2 ms | 16.0 ms | 27.9 (0.99x) | 8.5 (30%) | 446 | 66.7% | 1.07 | 0/512 |
| p2c | 929 ms | 1748 ms | 2838 ms | 12.4 ms | 13.4 ms | 36.9 (1.30x) | 8.4 (23%) | 591 | 70.4% | 1.11 | 0/512 |
| least_loaded | 304 ms | 1471 ms | 2539 ms | 12.4 ms | 13.4 ms | 58.6 (2.07x) | 36.3 (62%) | 938 | 82.9% | 1.12 | 0/512 |
| prefix | 19 ms | 804 ms | 1380 ms | 12.5 ms | 14.2 ms | 104.4 (3.69x) | 86.0 (82%) | 1670 | 92.3% | 1.20 | 0/512 |

- `prefix` router index: predicted hit 84.7% vs actual 92.3%; 444 requests expected warm, 0.0% stale
- `round_robin` route reasons: round_robin=512
- `random` route reasons: random=512
- `p2c` route reasons: p2c=512
- `least_loaded` route reasons: least_loaded=512
- `prefix` route reasons: prefix_hit=443, hash_affinity=51, load_fallback=17, prefix_spill=1

