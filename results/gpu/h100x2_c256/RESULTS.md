### 

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
|  | 0 ms | 0 ms | 0 ms | 0.0 ms | 0.0 ms | 0.0 | – | 0 | 0.0% | 0.00 | 0/0 |


### hot_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 282 ms | 1550 ms | 2809 ms | 12.3 ms | 191.2 ms | 84.8 | 40.4 (48%) | 5425 | 78.8% | 1.00 | 0/2400 |
| least_loaded | 218 ms | 673 ms | 2946 ms | 11.5 ms | 181.2 ms | 149.7 (1.77x) | 109.1 (73%) | 9578 | 90.8% | 1.02 | 0/2400 |
| prefix_pure | 545 ms | 1189 ms | 2518 ms | 14.6 ms | 187.8 ms | 123.0 (1.45x) | 46.1 (38%) | 7870 | 92.7% | 1.56 | 0/2400 |
| prefix | 202 ms | 641 ms | 2249 ms | 11.6 ms | 142.1 ms | 176.1 (2.08x) | 147.2 (84%) | 11271 | 93.8% | 1.03 | 0/2400 |
| weighted | 203 ms | 666 ms | 2794 ms | 11.7 ms | 177.3 ms | 162.5 (1.92x) | 138.1 (85%) | 10401 | 92.3% | 1.01 | 0/2400 |
| weighted_load | 221 ms | 725 ms | 2723 ms | 11.4 ms | 181.8 ms | 150.5 (1.77x) | 108.4 (72%) | 9629 | 90.7% | 1.01 | 0/2400 |

- `prefix_pure` router index: predicted hit 79.7% vs actual 92.7%; 2040 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 82.1% vs actual 93.8%; 2107 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 76.8% vs actual 92.3%; 2308 requests expected warm, 0.0% stale
- `weighted_load` router index: predicted hit 69.8% vs actual 90.7%; 2312 requests expected warm, 0.0% stale
- `round_robin` route reasons: round_robin=2400
- `least_loaded` route reasons: least_loaded=2400
- `prefix_pure` route reasons: prefix_hit=2040, hash_affinity=360
- `prefix` route reasons: prefix_hit=2107, hash_affinity=279, load_fallback=14
- `weighted` route reasons: weighted_hit=1958, weighted_cold=442
- `weighted_load` route reasons: weighted_hit=1785, weighted_cold=615

### multi_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 1509 ms | 4537 ms | 5094 ms | 11.7 ms | 196.7 ms | 56.3 | 25.2 (45%) | 3604 | 66.6% | 1.00 | 0/2400 |
| least_loaded | 1221 ms | 2306 ms | 4406 ms | 12.1 ms | 195.3 ms | 59.5 (1.06x) | 6.4 (11%) | 3810 | 62.5% | 1.09 | 0/2400 |
| prefix_pure | 239 ms | 895 ms | 3042 ms | 12.5 ms | 163.5 ms | 149.7 (2.66x) | 105.5 (70%) | 9581 | 93.1% | 1.08 | 0/2400 |
| prefix | 196 ms | 682 ms | 3017 ms | 12.7 ms | 133.2 ms | 166.7 (2.96x) | 142.3 (85%) | 10670 | 93.4% | 1.01 | 0/2400 |
| weighted | 198 ms | 782 ms | 4210 ms | 12.6 ms | 187.7 ms | 130.9 (2.32x) | 99.2 (76%) | 8379 | 88.5% | 1.03 | 0/2400 |
| weighted_load | 1329 ms | 2486 ms | 4270 ms | 12.0 ms | 195.2 ms | 57.3 (1.02x) | 5.5 (10%) | 3670 | 60.5% | 1.10 | 0/2400 |

- `prefix_pure` router index: predicted hit 64.3% vs actual 93.1%; 1690 requests expected warm, 0.0% stale
- `prefix` router index: predicted hit 64.6% vs actual 93.4%; 1694 requests expected warm, 0.0% stale
- `weighted` router index: predicted hit 65.9% vs actual 88.5%; 2333 requests expected warm, 0.0% stale
- `weighted_load` router index: predicted hit 35.6% vs actual 60.5%; 2329 requests expected warm, 0.0% stale
- `round_robin` route reasons: round_robin=2400
- `least_loaded` route reasons: least_loaded=2400
- `prefix_pure` route reasons: prefix_hit=1690, hash_affinity=710
- `prefix` route reasons: prefix_hit=1694, hash_affinity=690, load_fallback=16
- `weighted` route reasons: weighted_hit=1683, weighted_cold=717
- `weighted_load` route reasons: weighted_cold=1551, weighted_hit=849

