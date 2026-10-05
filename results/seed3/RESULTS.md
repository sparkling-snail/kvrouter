### failover

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| prefix | 17 ms | 351 ms | 1483 ms | 146.8 | 2348 | 95.1% | 1.27 | 1/1533 |

- `prefix` route reasons: prefix_hit=1455, hash_affinity=51, load_fallback=24, prefix_spill=2
- `prefix` errors: stream truncated=1

Per-second timeline (requests *started* in that second):

| t (s) | ok | err | TTFT p50 |
|---:|---:|---:|---:|
| 0 | 58 | 1 | 1295 ms |
| 1 | 59 | 0 | 380 ms |
| 2 | 98 | 0 | 49 ms |
| 3 | 153 | 0 | 46 ms |
| 4 | 170 | 0 | 23 ms |
| 5 | 195 | 0 | 24 ms |
| 6 | 159 | 0 | 15 ms |
| 7 | 192 | 0 | 15 ms |
| 8 | 207 | 0 | 14 ms |
| 9 | 213 | 0 | 13 ms |
| 10 | 28 | 0 | 13 ms |

### hot_tenant

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 527 ms | 1555 ms | 2023 ms | 51.7 | 827 | 80.5% | 1.00 | 0/512 |
| least_loaded | 37 ms | 874 ms | 1932 ms | 77.1 (1.49x) | 1233 | 88.7% | 1.17 | 0/512 |
| prefix_pure | 281 ms | 1005 ms | 1535 ms | 77.9 (1.51x) | 1246 | 94.6% | 3.00 | 0/512 |
| prefix | 16 ms | 596 ms | 1908 ms | 101.7 (1.97x) | 1627 | 92.7% | 1.54 | 0/512 |

- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=460, hash_affinity=52
- `prefix` route reasons: prefix_hit=450, hash_affinity=33, load_fallback=28, prefix_spill=1

### multi_tenant

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 958 ms | 3759 ms | 4561 ms | 26.8 | 428 | 63.6% | 1.00 | 0/512 |
| least_loaded | 314 ms | 1472 ms | 2821 ms | 54.4 (2.03x) | 870 | 81.1% | 1.05 | 0/512 |
| prefix_pure | 22 ms | 660 ms | 1754 ms | 113.2 (4.23x) | 1811 | 94.6% | 1.50 | 0/512 |
| prefix | 16 ms | 673 ms | 1939 ms | 109.0 (4.07x) | 1744 | 92.5% | 1.18 | 0/512 |

- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=459, hash_affinity=53
- `prefix` route reasons: prefix_hit=447, hash_affinity=40, load_fallback=24, prefix_spill=1

