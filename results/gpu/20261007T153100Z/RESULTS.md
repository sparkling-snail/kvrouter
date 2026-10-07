### 

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
|  | 0 ms | 0 ms | 0 ms | 0.0 ms | 0.0 ms | 0.0 | – | 0 | 0.0% | 0.00 | 0/0 |


### hot_tenant

Goodput SLO: TTFT <= 1000 ms, TPOT <= 50 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 33 ms | 90 ms | 372 ms | 7.2 ms | 59.4 ms | 69.7 | 69.7 (100%) | 4461 | 85.7% | 1.00 | 0/2400 |
| least_loaded | 33 ms | 89 ms | 333 ms | 7.1 ms | 53.6 ms | 78.4 (1.12x) | 78.4 (100%) | 5016 | 90.7% | 1.02 | 0/2400 |
| prefix_pure | 37 ms | 82 ms | 546 ms | 7.9 ms | 51.1 ms | 77.0 (1.10x) | 77.0 (100%) | 4929 | 93.9% | 1.56 | 0/2400 |
| prefix | 32 ms | 74 ms | 416 ms | 7.2 ms | 49.6 ms | 84.5 (1.21x) | 84.5 (100%) | 5409 | 93.9% | 1.03 | 0/2400 |
| weighted | 33 ms | 79 ms | 444 ms | 7.1 ms | 50.0 ms | 83.6 (1.20x) | 83.6 (100%) | 5349 | 93.3% | 1.00 | 0/2400 |
| weighted_load | 33 ms | 83 ms | 436 ms | 7.1 ms | 54.2 ms | 78.8 (1.13x) | 78.8 (100%) | 5040 | 90.9% | 1.02 | 0/2400 |

- `prefix_pure` router index: predicted hit 89.1% vs actual 93.9%; 2224 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 89.2% vs actual 93.9%; 2226 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 89.6% vs actual 93.3%; 2352 requests expected warm, 0.0% stale
- `weighted_load` router index: predicted hit 87.0% vs actual 90.9%; 2355 requests expected warm, 0.0% stale
- `round_robin` route reasons: round_robin=2400
- `least_loaded` route reasons: least_loaded=2400
- `prefix_pure` route reasons: prefix_hit=2224, hash_affinity=176
- `prefix` route reasons: prefix_hit=2226, hash_affinity=162, load_fallback=12
- `weighted` route reasons: weighted_hit=2227, weighted_cold=173
- `weighted_load` route reasons: weighted_hit=2155, weighted_cold=245

### multi_tenant

Goodput SLO: TTFT <= 1000 ms, TPOT <= 50 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 37 ms | 131 ms | 1138 ms | 7.6 ms | 64.5 ms | 57.1 | 56.5 (99%) | 3656 | 77.2% | 1.00 | 0/2400 |
| least_loaded | 36 ms | 130 ms | 776 ms | 7.4 ms | 61.1 ms | 69.9 (1.22x) | 69.5 (99%) | 4473 | 86.8% | 1.01 | 0/2400 |
| prefix_pure | 33 ms | 60 ms | 779 ms | 7.5 ms | 46.9 ms | 80.8 (1.41x) | 80.5 (100%) | 5169 | 93.8% | 1.08 | 0/2400 |
| prefix | 33 ms | 74 ms | 780 ms | 7.5 ms | 47.7 ms | 80.4 (1.41x) | 80.0 (100%) | 5147 | 93.3% | 1.00 | 0/2400 |
| weighted | 34 ms | 87 ms | 781 ms | 7.4 ms | 53.9 ms | 77.0 (1.35x) | 76.7 (100%) | 4931 | 91.2% | 1.01 | 0/2400 |
| weighted_load | 35 ms | 129 ms | 781 ms | 7.4 ms | 60.7 ms | 70.3 (1.23x) | 69.9 (99%) | 4501 | 87.4% | 1.00 | 0/2400 |

- `prefix_pure` router index: predicted hit 86.6% vs actual 93.8%; 2158 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 86.2% vs actual 93.3%; 2150 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 86.9% vs actual 91.2%; 2352 requests expected warm, 0.0% stale
- `weighted_load` router index: predicted hit 80.2% vs actual 87.4%; 2352 requests expected warm, 0.0% stale
- `round_robin` route reasons: round_robin=2400
- `least_loaded` route reasons: least_loaded=2400
- `prefix_pure` route reasons: prefix_hit=2158, hash_affinity=242
- `prefix` route reasons: prefix_hit=2150, hash_affinity=236, load_fallback=14
- `weighted` route reasons: weighted_hit=2154, weighted_cold=246
- `weighted_load` route reasons: weighted_hit=1967, weighted_cold=433

