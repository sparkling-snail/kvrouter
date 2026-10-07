### 

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
|  | 0 ms | 0 ms | 0 ms | 0.0 ms | 0.0 ms | 0.0 | – | 0 | 0.0% | 0.00 | 0/0 |


### hot_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 98 ms | 475 ms | 1432 ms | 9.6 ms | 160.5 ms | 98.8 | 86.3 (87%) | 6322 | 85.3% | 1.00 | 0/2400 |
| least_loaded | 63 ms | 256 ms | 1481 ms | 9.2 ms | 68.3 ms | 128.4 (1.30x) | 119.9 (93%) | 8219 | 90.3% | 1.02 | 0/2400 |
| prefix_pure | 122 ms | 265 ms | 1433 ms | 10.2 ms | 71.7 ms | 122.5 (1.24x) | 115.7 (94%) | 7842 | 93.8% | 1.56 | 0/2400 |
| prefix | 55 ms | 169 ms | 1294 ms | 9.2 ms | 60.0 ms | 150.4 (1.52x) | 143.4 (95%) | 9626 | 94.0% | 1.01 | 0/2400 |
| weighted | 60 ms | 164 ms | 1480 ms | 9.0 ms | 62.9 ms | 146.6 (1.48x) | 138.8 (95%) | 9384 | 93.2% | 1.01 | 0/2400 |
| weighted_load | 61 ms | 168 ms | 1414 ms | 9.1 ms | 68.5 ms | 132.0 (1.34x) | 122.9 (93%) | 8448 | 90.9% | 1.01 | 0/2400 |

- `prefix_pure` router index: predicted hit 87.0% vs actual 93.8%; 2175 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 87.4% vs actual 94.0%; 2178 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 87.6% vs actual 93.2%; 2327 requests expected warm, 0.0% stale
- `weighted_load` router index: predicted hit 85.9% vs actual 90.9%; 2314 requests expected warm, 0.0% stale
- `round_robin` route reasons: round_robin=2400
- `least_loaded` route reasons: least_loaded=2400
- `prefix_pure` route reasons: prefix_hit=2175, hash_affinity=225
- `prefix` route reasons: prefix_hit=2178, hash_affinity=203, load_fallback=19
- `weighted` route reasons: weighted_hit=2178, weighted_cold=222
- `weighted_load` route reasons: weighted_hit=2129, weighted_cold=271

### multi_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 165 ms | 708 ms | 2975 ms | 10.4 ms | 193.1 ms | 64.6 | 34.5 (53%) | 4135 | 72.2% | 1.00 | 0/2400 |
| least_loaded | 61 ms | 214 ms | 2554 ms | 9.8 ms | 124.9 ms | 107.0 (1.66x) | 88.9 (83%) | 6851 | 86.3% | 1.00 | 0/2400 |
| prefix_pure | 59 ms | 147 ms | 2142 ms | 9.6 ms | 60.3 ms | 140.0 (2.17x) | 132.5 (95%) | 8958 | 94.0% | 1.08 | 0/2400 |
| prefix | 60 ms | 162 ms | 2209 ms | 9.5 ms | 64.6 ms | 141.6 (2.19x) | 134.1 (95%) | 9063 | 93.4% | 1.03 | 0/2400 |
| weighted | 65 ms | 197 ms | 2625 ms | 9.6 ms | 119.6 ms | 122.3 (1.89x) | 115.7 (95%) | 7825 | 90.0% | 1.03 | 0/2400 |
| weighted_load | 61 ms | 202 ms | 2724 ms | 9.9 ms | 121.6 ms | 108.7 (1.68x) | 93.2 (86%) | 6958 | 86.9% | 1.00 | 0/2400 |

- `prefix_pure` router index: predicted hit 74.7% vs actual 94.0%; 1905 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 75.3% vs actual 93.4%; 1949 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 77.1% vs actual 90.0%; 2308 requests expected warm, 0.0% stale
- `weighted_load` router index: predicted hit 46.6% vs actual 86.9%; 2278 requests expected warm, 0.0% stale
- `round_robin` route reasons: round_robin=2400
- `least_loaded` route reasons: least_loaded=2400
- `prefix_pure` route reasons: prefix_hit=1905, hash_affinity=495
- `prefix` route reasons: prefix_hit=1949, hash_affinity=437, load_fallback=14
- `weighted` route reasons: weighted_hit=1952, weighted_cold=448
- `weighted_load` route reasons: weighted_cold=1238, weighted_hit=1162

