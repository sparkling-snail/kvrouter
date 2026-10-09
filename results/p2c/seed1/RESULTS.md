### hot_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 658 ms | 1472 ms | 1783 ms | 12.6 ms | 14.7 ms | 45.9 | 16.2 (35%) | 734 | 78.8% | 1.00 | 0/512 |
| random | 627 ms | 2073 ms | 3117 ms | 12.4 ms | 15.8 ms | 41.8 (0.91x) | 18.4 (44%) | 669 | 80.0% | 1.07 | 0/512 |
| p2c | 577 ms | 1211 ms | 1897 ms | 12.5 ms | 13.5 ms | 52.9 (1.15x) | 22.6 (43%) | 846 | 80.1% | 1.17 | 0/512 |
| least_loaded | 280 ms | 923 ms | 1630 ms | 12.4 ms | 13.4 ms | 68.2 (1.49x) | 45.2 (66%) | 1091 | 87.1% | 1.09 | 0/512 |
| prefix | 19 ms | 589 ms | 1365 ms | 12.6 ms | 14.0 ms | 114.2 (2.49x) | 98.1 (86%) | 1827 | 93.0% | 1.16 | 0/512 |

- `prefix` router index: predicted hit 85.8% vs actual 93.0%; 450 requests expected warm, 0.0% stale
- `round_robin` route reasons: round_robin=512
- `random` route reasons: random=512
- `p2c` route reasons: p2c=512
- `least_loaded` route reasons: least_loaded=512
- `prefix` route reasons: prefix_hit=449, hash_affinity=39, load_fallback=23, prefix_spill=1

### multi_tenant

Goodput SLO: TTFT <= 500 ms, TPOT <= 25 ms (0 = none).

| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| round_robin | 1107 ms | 2969 ms | 3737 ms | 12.3 ms | 14.9 ms | 27.3 | 7.9 (29%) | 436 | 64.3% | 1.00 | 0/512 |
| random | 797 ms | 3164 ms | 4468 ms | 12.2 ms | 15.6 ms | 29.2 (1.07x) | 9.9 (34%) | 468 | 67.2% | 1.09 | 0/512 |
| p2c | 952 ms | 1804 ms | 2844 ms | 12.4 ms | 13.3 ms | 35.8 (1.31x) | 7.3 (21%) | 572 | 69.9% | 1.09 | 0/512 |
| least_loaded | 267 ms | 1520 ms | 2539 ms | 12.4 ms | 13.4 ms | 57.4 (2.10x) | 34.4 (60%) | 918 | 83.0% | 1.30 | 0/512 |
| prefix | 18 ms | 648 ms | 1640 ms | 12.4 ms | 14.2 ms | 104.1 (3.82x) | 89.5 (86%) | 1666 | 92.7% | 1.04 | 0/512 |

- `prefix` router index: predicted hit 85.9% vs actual 92.7%; 451 requests expected warm, 0.0% stale
- `round_robin` route reasons: round_robin=512
- `random` route reasons: random=512
- `p2c` route reasons: p2c=512
- `least_loaded` route reasons: least_loaded=512
- `prefix` route reasons: prefix_hit=450, hash_affinity=45, load_fallback=16, prefix_spill=1

