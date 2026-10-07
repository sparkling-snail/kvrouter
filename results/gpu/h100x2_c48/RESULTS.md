### 

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
|  | 0 ms | 0 ms | 0 ms | 0.0 ms | 0.0 ms | 0.0 | – | 0 | 0.0% | 0.00 | 0/0 |


### hot_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 33 ms | 95 ms | 442 ms | 7.2 ms | 60.3 ms | 69.3 | 68.8 (99%) | 4437 | 85.3% | 1.00 | 0/2400 |
| least_loaded | 34 ms | 86 ms | 495 ms | 7.1 ms | 53.4 ms | 78.8 (1.14x) | 78.1 (99%) | 5041 | 90.8% | 1.00 | 0/2400 |
| prefix_pure | 37 ms | 83 ms | 622 ms | 7.9 ms | 50.9 ms | 77.1 (1.11x) | 76.2 (99%) | 4932 | 93.9% | 1.56 | 0/2400 |
| prefix | 32 ms | 73 ms | 386 ms | 7.1 ms | 49.0 ms | 84.9 (1.22x) | 84.3 (99%) | 5431 | 93.9% | 1.01 | 0/2400 |
| weighted | 33 ms | 79 ms | 431 ms | 7.1 ms | 50.0 ms | 83.6 (1.21x) | 82.9 (99%) | 5350 | 93.2% | 1.01 | 0/2400 |
| weighted_load | 33 ms | 91 ms | 430 ms | 7.1 ms | 54.7 ms | 78.8 (1.14x) | 78.2 (99%) | 5046 | 91.0% | 1.00 | 0/2400 |

- `prefix_pure` router index: predicted hit 89.2% vs actual 93.9%; 2224 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 89.3% vs actual 93.9%; 2227 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 89.5% vs actual 93.2%; 2352 requests expected warm, 0.0% stale
- `weighted_load` router index: predicted hit 86.7% vs actual 91.0%; 2352 requests expected warm, 0.0% stale
- `round_robin` route reasons: round_robin=2400
- `least_loaded` route reasons: least_loaded=2400
- `prefix_pure` route reasons: prefix_hit=2224, hash_affinity=176
- `prefix` route reasons: prefix_hit=2227, hash_affinity=161, load_fallback=12
- `weighted` route reasons: weighted_hit=2226, weighted_cold=174
- `weighted_load` route reasons: weighted_hit=2149, weighted_cold=251

### multi_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 38 ms | 139 ms | 1143 ms | 7.6 ms | 66.1 ms | 57.3 | 56.1 (98%) | 3669 | 77.6% | 1.00 | 0/2400 |
| least_loaded | 35 ms | 128 ms | 769 ms | 7.4 ms | 60.2 ms | 70.3 (1.23x) | 69.3 (99%) | 4497 | 87.3% | 1.00 | 0/2400 |
| prefix_pure | 33 ms | 62 ms | 781 ms | 7.5 ms | 48.0 ms | 80.7 (1.41x) | 79.6 (99%) | 5167 | 93.8% | 1.08 | 0/2400 |
| prefix | 33 ms | 75 ms | 771 ms | 7.5 ms | 48.6 ms | 80.7 (1.41x) | 79.5 (99%) | 5162 | 93.4% | 1.00 | 0/2400 |
| weighted | 34 ms | 83 ms | 772 ms | 7.4 ms | 52.4 ms | 77.1 (1.35x) | 76.0 (99%) | 4936 | 91.3% | 1.01 | 0/2400 |
| weighted_load | 35 ms | 129 ms | 790 ms | 7.4 ms | 60.6 ms | 70.0 (1.22x) | 68.9 (98%) | 4478 | 87.1% | 1.00 | 0/2400 |

- `prefix_pure` router index: predicted hit 86.6% vs actual 93.8%; 2159 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 86.3% vs actual 93.4%; 2151 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 87.1% vs actual 91.3%; 2352 requests expected warm, 0.0% stale
- `weighted_load` router index: predicted hit 80.7% vs actual 87.1%; 2352 requests expected warm, 0.0% stale
- `round_robin` route reasons: round_robin=2400
- `least_loaded` route reasons: least_loaded=2400
- `prefix_pure` route reasons: prefix_hit=2159, hash_affinity=241
- `prefix` route reasons: prefix_hit=2151, hash_affinity=237, load_fallback=12
- `weighted` route reasons: weighted_hit=2159, weighted_cold=241
- `weighted_load` route reasons: weighted_hit=1980, weighted_cold=420

