#!/usr/bin/env bash
# How wrong does the router's approximate prefix index get under KV-cache
# memory pressure? Runs the multi-tenant workload (prefix policy) against mocks
# with a small cache, once with the default index size and once with the index
# shrunk to the cache's capacity, for each seed.
#
#   scripts/index_drift.sh            # -> results/index_drift/*.json
#
# 12288 cache tokens / 32 tokens per router block (128 bytes) = 384 blocks.
set -euo pipefail
cd "$(dirname "$0")/.."

CACHE_TOKENS=${CACHE_TOKENS:-12288}
SEEDS=${SEEDS:-"1 2 3"}
OUT=results/index_drift
PORTS="8101 8102 8103 8104"
BACKENDS=$(for p in $PORTS; do printf 'http://127.0.0.1:%s,' "$p"; done | sed 's/,$//')
mkdir -p bin logs "$OUT"
go build -o bin/ ./cmd/...

pids=""
cleanup() { [[ -n $pids ]] && kill $pids 2>/dev/null || true; }
trap cleanup EXIT
for p in $PORTS; do
  ./bin/mockvllm -port "$p" -cache-tokens "$CACHE_TOKENS" >"logs/drift_mock$p.log" 2>&1 & pids="$pids $!"
done
sleep 0.5

for seed in $SEEDS; do
  for ib in 4096 $((CACHE_TOKENS / 32)); do
    for p in $PORTS; do curl -fsS -XPOST "127.0.0.1:$p/reset" >/dev/null; done
    ./bin/router -listen :9100 -backends "$BACKENDS" -index-blocks "$ib" 2>"logs/drift_router.log" &
    rp=$!; pids="$pids $rp"; sleep 0.3
    ./bin/bench -url http://127.0.0.1:9100 -seed "$seed" -scenario "drift_idx$ib" -policy "seed$seed" \
        -out "$OUT/idx${ib}_seed${seed}.json" | grep -E "==|cache hit|router index"
    kill "$rp"; wait "$rp" 2>/dev/null || true
  done
done
