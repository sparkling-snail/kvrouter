### failover

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| prefix | 19 ms | 324 ms | 1351 ms | 152.3 | 2437 | 95.3% | 1.29 | 3/1536 |

- `prefix` route reasons: prefix_hit=1456, hash_affinity=54, load_fallback=21, prefix_spill=2
- `prefix` errors: stream truncated=3

Per-second timeline (requests *started* in that second):

| t (s) | ok | err | TTFT p50 |
|---:|---:|---:|---:|
| 0 | 62 | 0 | 1002 ms |
| 1 | 104 | 3 | 23 ms |
| 2 | 116 | 0 | 51 ms |
| 3 | 159 | 0 | 22 ms |
| 4 | 182 | 0 | 22 ms |
| 5 | 215 | 0 | 18 ms |
| 6 | 185 | 0 | 20 ms |
| 7 | 188 | 0 | 15 ms |
| 8 | 215 | 0 | 15 ms |
| 9 | 107 | 0 | 14 ms |

### hot_tenant

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 660 ms | 1324 ms | 1730 ms | 46.2 | 739 | 79.4% | 1.00 | 0/512 |
| least_loaded | 251 ms | 988 ms | 1649 ms | 72.3 (1.56x) | 1157 | 87.4% | 1.09 | 0/512 |
| prefix_pure | 209 ms | 677 ms | 1213 ms | 96.0 (2.08x) | 1536 | 94.6% | 2.47 | 0/512 |
| prefix | 17 ms | 552 ms | 1375 ms | 113.8 (2.46x) | 1821 | 93.0% | 1.12 | 0/512 |

- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=457, hash_affinity=55
- `prefix` route reasons: prefix_hit=446, hash_affinity=38, load_fallback=25, prefix_spill=3

### multi_tenant

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 961 ms | 2845 ms | 4166 ms | 26.2 | 420 | 66.8% | 1.00 | 0/512 |
| least_loaded | 255 ms | 1740 ms | 2802 ms | 54.1 (2.06x) | 865 | 83.2% | 1.30 | 0/512 |
| prefix_pure | 17 ms | 508 ms | 1359 ms | 134.1 (5.11x) | 2146 | 94.6% | 1.22 | 0/512 |
| prefix | 16 ms | 561 ms | 1636 ms | 112.9 (4.30x) | 1806 | 93.6% | 1.22 | 0/512 |

- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=464, hash_affinity=48
- `prefix` route reasons: prefix_hit=456, hash_affinity=48, load_fallback=8

