### failover

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| prefix | 21 ms | 354 ms | 1367 ms | 137.7 | 2203 | 94.6% | 1.18 | 0/1536 |

- `prefix` route reasons: prefix_hit=1456, hash_affinity=51, load_fallback=26, prefix_spill=3

Per-second timeline (requests *started* in that second):

| t (s) | ok | err | TTFT p50 |
|---:|---:|---:|---:|
| 0 | 64 | 0 | 956 ms |
| 1 | 102 | 0 | 25 ms |
| 2 | 100 | 0 | 319 ms |
| 3 | 69 | 0 | 292 ms |
| 4 | 189 | 0 | 43 ms |
| 5 | 205 | 0 | 20 ms |
| 6 | 141 | 0 | 23 ms |
| 7 | 174 | 0 | 16 ms |
| 8 | 156 | 0 | 19 ms |
| 9 | 184 | 0 | 15 ms |
| 10 | 152 | 0 | 14 ms |

### hot_tenant

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 656 ms | 1560 ms | 2244 ms | 49.5 | 791 | 81.3% | 1.00 | 0/512 |
| least_loaded | 48 ms | 850 ms | 1647 ms | 77.8 (1.57x) | 1245 | 89.1% | 1.12 | 0/512 |
| prefix_pure | 172 ms | 478 ms | 1259 ms | 98.6 (1.99x) | 1578 | 94.8% | 2.75 | 0/512 |
| prefix | 20 ms | 653 ms | 1731 ms | 116.8 (2.36x) | 1869 | 93.5% | 1.49 | 0/512 |

- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=461, hash_affinity=51
- `prefix` route reasons: prefix_hit=451, hash_affinity=32, load_fallback=25, prefix_spill=4

### multi_tenant

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 1194 ms | 2878 ms | 4183 ms | 24.4 | 390 | 61.7% | 1.00 | 0/512 |
| least_loaded | 318 ms | 1493 ms | 2836 ms | 52.9 (2.17x) | 847 | 81.2% | 1.14 | 0/512 |
| prefix_pure | 27 ms | 742 ms | 1675 ms | 101.0 (4.14x) | 1616 | 94.6% | 1.81 | 0/512 |
| prefix | 16 ms | 677 ms | 1643 ms | 115.7 (4.75x) | 1851 | 93.1% | 1.28 | 0/512 |

- `round_robin` route reasons: round_robin=512
- `least_loaded` route reasons: least_loaded=512
- `prefix_pure` route reasons: prefix_hit=456, hash_affinity=56
- `prefix` route reasons: prefix_hit=449, hash_affinity=51, load_fallback=12

