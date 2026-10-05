# kvrouter — a KV-cache-aware router for vLLM

A Go reverse proxy that sits in front of several OpenAI-compatible inference
servers (vLLM, SGLang, TGI) and sends each request to the instance most likely
to already hold its prompt prefix in KV cache — without letting cache affinity
turn one instance into a hot spot.

```
                       ┌────────────────────────── kvrouter ───────────────────────────┐
 client ─ POST /v1/chat │ extract prompt → chained block hashes → policy.Pick          │
   (SSE stream)  ─────► │   1. longest cached prefix among backends under load bound    │──► vLLM :8001
                        │   2. else rendezvous hash on first 8 blocks (cold prefix)     │──► vLLM :8002
                        │   3. retry on another backend if it fails before first byte   │──► vLLM :8003
                        │ active /health probes + passive ejection · /metrics · headers │──► vLLM :8004
                        └───────────────────────────────────────────────────────────────┘
```

Stdlib only, no dependencies. ~1,800 lines of Go (router, mock server, load generator) plus ~270 lines of tests.

## Results (4 instances, mock vLLM, mean of 3 seeds)

**Multi-tenant** — 16 tenants with ~3k-token system prompts, 128 conversations × 4 turns, 48 concurrent.

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | vs RR | cache hit | load imbalance |
|---|---:|---:|---:|---:|---:|---:|---:|
| round_robin  | 1038 ms | 3161 ms | 4303 ms | 25.8  | 1.00× | 64.0% | 1.00 |
| least_loaded |  296 ms | 1568 ms | 2820 ms | 53.8  | 2.09× | 81.8% | 1.16 |
| prefix_pure (no load bound) | 22 ms | 637 ms | 1596 ms | 116.1 | 4.50× | 94.6% | 1.51 |
| **prefix** (bounded load)   | **16 ms** | **637 ms** | 1740 ms | **112.5** | **4.36×** | 93.1% | 1.23 |

**Hot tenant** — same, but 50% of conversations belong to one tenant.

| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | vs RR | cache hit | load imbalance |
|---|---:|---:|---:|---:|---:|---:|---:|
| round_robin  | 614 ms | 1480 ms | 1999 ms | 49.1  | 1.00× | 80.4% | 1.00 |
| least_loaded | 112 ms |  904 ms | 1742 ms | 75.7  | 1.54× | 88.4% | 1.13 |
| prefix_pure  | 220 ms |  720 ms | 1336 ms | 90.8  | 1.85× | 94.7% | **2.74** |
| **prefix**   | **18 ms** | **600 ms** | 1672 ms | **110.8** | **2.26×** | 93.1% | 1.38 |

**Failover** — prefix policy, `kill -9` one of four backends ~2 s into the run, restart it ~4 s later.
Across 3 runs: **4 errors in 4,605 requests**, all streams that were mid-flight on the
killed process. Every request that arrived after the crash succeeded (3–14 transparent
retries per run). The backend was ejected on the first broken connection and restored
on its first good probe, then took traffic again with a reset index.
The cost of a crash shows up as a short TTFT bump rather than errors: in some runs p50
TTFT rises to ~300 ms for 1–2 seconds after the kill, because the dead instance's
tenants get re-placed and have to be prefilled cold on their new owners.

Per-seed tables and raw JSON: [`results/seed{1,2,3}/`](results/). Spread across
seeds is small for p50 and hit rate; `prefix_pure` throughput varies most
(101–134 req/s) because its balance depends on where rendezvous hashing happens
to land 16 tenants.

### What the numbers say

- **Why round-robin loses:** with 16 × 3k-token prompts (≈48k tokens) and 32k
  tokens of cache per instance, round-robin makes every instance try to cache
  every tenant, so they thrash. Affinity routing partitions tenants, so the
  *aggregate* cache across the fleet is usable. That's also why TTFT p50 drops
  ~60×: almost every request only prefills its new turn.
- **Least-loaded is the honest baseline**, not round-robin. It already halves
  queueing. Prefix-aware is still 2.1× its throughput.
- **The load bound earns its keep under skew.** Pure affinity pins the hot
  tenant to one instance (2.7× imbalance, p50 TTFT 220 ms). Bounded load
  spills it onto a second instance, which then caches that tenant too —
  replicating the hot prefix — and p50 falls to 18 ms with +22% throughput.
  Under uniform load it costs ~1.5 pts of hit rate and nothing measurable in
  throughput.
- **Trade-off visible in p99:** pure affinity has slightly better p99 in the
  hot-tenant case. Each spill forces one cold 3k-token prefill on the new
  replica. p99 for all prefix policies is dominated by the cold-start burst
  at t=0 (48 cold conversations at once; see the failover timeline in
  `results/seed*/RESULTS.md` — p50 TTFT is ~1 s in second 0 and settles to ~15–25 ms once warm).

> **Caveat:** these are from a simulator, not GPUs. `cmd/mockvllm` models an
> LRU prefix cache with vLLM's 16-token blocks, serialized prefill at ~10k
> tok/s, and batch-dependent decode. The *relative* effects are the point; the
> absolute numbers are not a vLLM benchmark. See "Running against real vLLM".

## Design

### Prefix hashing (`internal/prefix`)
The router hashes the prompt text in 128-byte blocks (≈32 tokens), each hash
chained to the previous one: `h[i] = H(h[i-1] ‖ block[i])`. This mirrors how
vLLM's automatic prefix caching keys KV blocks, so "shares `h[i]`" ⇔ "shares the
first `i+1` blocks". The router doesn't tokenize — it doesn't need the exact
blocks vLLM uses, only *same prefix → same chain*. Chat messages are flattened
in order so turn *n*'s prompt is a strict prefix of turn *n+1*'s.

### Prefix index (`internal/lru`)
Per backend, an LRU set of block hashes approximates what that backend has
cached. Chains are inserted deepest-first so eviction removes leaves before
roots (evicting a root would orphan the entire chain — vLLM also evicts
leaf-first). Lookup is a linear scan of the request's chain per backend:
O(backends × blocks) map lookups.

The index is **approximate and optimistic**: it's updated when the router
sends a request, not from the engine's real cache state. When a backend is
ejected its index is cleared, since a crash or restart loses its KV cache.

### Policy (`internal/router/policy.go`)
1. Compute the load bound `ceil(c · avg_inflight) + slack` (c = 1.25, slack = 2) —
   *consistent hashing with bounded loads* (Mirrokni, Thorup, Zadimoghaddam 2018).
2. Among backends under the bound, pick the one with the longest cached prefix.
   If the overall cache-best backend was over the bound, this is a `prefix_spill`.
3. Matches shorter than the affinity window (8 blocks ≈ 1 KB) don't count.
   Every prompt starts with the same chat-template/global header; without this
   rule, new tenants pile onto whichever backend saw that header first. (There's
   a regression test for this — `TestSharedHeaderDoesNotCountAsHit` — and the
   benchmark includes a 200-token shared header.)
4. Otherwise, rendezvous-hash the first 8 blocks across backends under the bound
   (`hash_affinity`), so a cold prompt's follow-ups land in the same place even
   before the index knows about it. If the rendezvous owner is overloaded, take
   the next one (`load_fallback`).

Every decision's reason is exposed as a metric label and an `X-Router-Reason`
response header, along with `X-Router-Backend` and `X-Router-Matched-Blocks`.

### Failure handling (`internal/router/proxy.go`, `pool.go`)
- **Retries only before the first byte.** Once tokens have streamed to the
  client, replaying on another backend would duplicate output, so mid-stream
  failures are surfaced, not retried (`TestNoRetryAfterFirstByte`).
- **Passive health:** connection refused/reset ejects immediately; HTTP 5xx and
  timeouts eject after 3 in a row (slow ≠ dead).
- **Active health:** `GET /health` every second; restore on first success.
- **Panic routing:** if every backend is marked unhealthy, route to all of them
  anyway (as Envoy does) — the health checker may be what's broken.
- Streaming: 32 KB pooled buffers, flush per read, compression disabled so SSE
  is never buffered, up to 512 idle keep-alive connections per backend.
- Graceful shutdown drains in-flight streams for up to 30 s.

### Observability
`/metrics` (Prometheus text format): route decisions by reason, retries,
mid-stream errors, failed requests by status, per-backend in-flight / health /
served / index size, and an upstream time-to-first-byte histogram. `/debug/backends`
returns the same state as JSON. The benchmark can also scrape vLLM's own
`vllm:prefix_cache_{queries,hits}_total` counters; client-reported and
server-scraped hit rates agree to within 0.25 pts across all 27 runs here.

## Layout

```
cmd/router      the router
cmd/mockvllm    GPU-free vLLM stand-in: prefix cache, prefill contention, SSE, /metrics, fault injection
cmd/bench       multi-tenant multi-turn load generator → TTFT, throughput, cache hit, per-backend balance
cmd/report      results/*.json → markdown tables
internal/prefix block hashing + prompt extraction
internal/lru    bounded hash set (router index and mock cache)
internal/router policies, proxy, health, metrics (+ tests)
scripts/bench.sh   full comparison + failover run
deploy/            Dockerfile, docker-compose for real vLLM
```

## Run it

```bash
go test -race ./...            # unit + HTTP-level failure tests
./scripts/bench.sh             # ~1 min: 2 scenarios × 4 policies + failover, prints tables

# by hand
go build -o bin/ ./cmd/...
for p in 8001 8002 8003 8004; do ./bin/mockvllm -port $p & done
./bin/router -backends http://127.0.0.1:8001,http://127.0.0.1:8002,http://127.0.0.1:8003,http://127.0.0.1:8004
./bin/bench -url http://127.0.0.1:9000
curl -s localhost:9000/metrics | grep route_decisions
```

Fault injection on the mock: `POST /admin/fault?rate=0.2` (random 503s),
`POST /admin/down` / `/admin/up` (fail health checks), `POST /reset` (drop cache).

## Running against real vLLM

`deploy/docker-compose.vllm.yml` starts vLLM instances (one GPU each) behind
the router. **It has not been run as part of this repo's results — that needs
GPUs.** Then:

```bash
BACKENDS=http://gpu-host:8001,http://gpu-host:8002 MODEL=Qwen/Qwen2.5-7B-Instruct NO_MOCK=1 \
BENCH_ARGS="-sys-tokens 2000 -concurrency 32" ./scripts/bench.sh
```

Notes for the real thing:
- Start vLLM with `--enable-prompt-tokens-details` so streamed usage includes
  `cached_tokens`; otherwise rely on the scraped `vllm:prefix_cache_*` counters.
- Reset caches between policies (restart, or vLLM's `/reset_prefix_cache`
  endpoint when the server runs in dev mode).
- Size the workload so total distinct prefixes exceed one instance's KV cache
  but fit the fleet's. That's the regime this router is for. If everything fits
  on every instance, round-robin hit rates converge with prefix routing.

## Limitations and next steps

- **Index is a guess.** The router believes a prefix is cached because it sent
  it there. vLLM evicts on its own schedule, so the index drifts under memory
  pressure. Next step: subscribe to the engine's KV-cache events (vLLM can
  publish block stored/removed events) and make the index exact.
- **Byte blocks, not token blocks.** Fine for affinity, but it can't compute
  exact token savings. Using the model's tokenizer would allow cost-based
  scoring (e.g. `uncached_tokens × prefill_cost + queue_depth × step_time`)
  instead of "longest match under a load bound".
- **Load signal is router-local in-flight count.** With multiple router
  replicas, each sees only its own traffic. Options: scrape
  `vllm:num_requests_waiting` / KV usage from backends, or share the index
  across replicas.
- **No prefill/decode disaggregation awareness**, no LoRA-adapter affinity, no
  per-tenant fairness. Each would be a policy extension.
- Related production systems worth comparing against: the vLLM production-stack
  router, llm-d's inference scheduler, and SGLang's cache-aware router (which
  keeps an approximate radix tree per worker, similar in spirit to the index here).
