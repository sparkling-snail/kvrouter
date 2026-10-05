#!/usr/bin/env bash
# Runs the full comparison against 4 mock vLLM instances:
#   scenario multi_tenant : 16 tenants, uniform traffic
#   scenario hot_tenant   : 50% of conversations hit one tenant
#   policies              : round_robin, least_loaded, prefix_pure (no load bound), prefix
#   scenario failover     : prefix policy; kill -9 one backend mid-run, restart it later
# Results land in results/*.json; a markdown table is printed at the end.
#
# To run against real vLLM instead, skip the mocks and set BACKENDS, e.g.
#   BACKENDS=http://gpu1:8000,http://gpu2:8000 MODEL=meta-llama/Llama-3.1-8B-Instruct NO_MOCK=1 scripts/bench.sh
set -euo pipefail
cd "$(dirname "$0")/.."

PORTS=(8001 8002 8003 8004)
BACKENDS=${BACKENDS:-$(printf 'http://127.0.0.1:%s,' "${PORTS[@]}" | sed 's/,$//')}
MODEL=${MODEL:-mock}
MOCK_ARGS=${MOCK_ARGS:-}
BENCH_ARGS=${BENCH_ARGS:-}
mkdir -p bin results logs
go build -o bin/ ./cmd/...

pids=()
cleanup() { kill "${pids[@]}" 2>/dev/null || true; }
trap cleanup EXIT

declare -A MOCK_PID
start_mock() { ./bin/mockvllm -port "$1" $MOCK_ARGS >"logs/mock$1.log" 2>&1 & MOCK_PID[$1]=$!; pids+=($!); }
if [[ -z "${NO_MOCK:-}" ]]; then
  for p in "${PORTS[@]}"; do start_mock "$p"; done
  sleep 0.5
fi

reset_backends() {
  [[ -n "${NO_MOCK:-}" ]] && return   # real vLLM: restart servers or POST /reset_prefix_cache yourself
  for p in "${PORTS[@]}"; do curl -fsS -XPOST "127.0.0.1:$p/reset" >/dev/null; done
}

run() { # scenario policy router-args... -- bench-args...
  local scenario=$1 label=$2; shift 2
  local rargs=() bargs=()
  while [[ $# -gt 0 && $1 != -- ]]; do rargs+=("$1"); shift; done
  [[ $# -gt 0 ]] && shift
  bargs=("$@")
  reset_backends
  ./bin/router -listen :9000 -backends "$BACKENDS" "${rargs[@]}" 2>"logs/router_${scenario}_${label}.log" &
  local rp=$!; pids+=($rp); sleep 0.3
  ./bin/bench -url http://127.0.0.1:9000 -model "$MODEL" -scenario "$scenario" -policy "$label" \
      -scrape "$BACKENDS" -out "results/${scenario}_${label}.json" $BENCH_ARGS "${bargs[@]}"
  curl -fsS 127.0.0.1:9000/metrics > "results/${scenario}_${label}.router.prom" || true
  kill "$rp"; wait "$rp" 2>/dev/null || true
  echo
}

for scenario in multi_tenant hot_tenant; do
  extra=()
  [[ $scenario == hot_tenant ]] && extra=(-hot-frac 0.5)
  run $scenario round_robin  -policy round_robin                   -- "${extra[@]}"
  run $scenario least_loaded -policy least_loaded                  -- "${extra[@]}"
  run $scenario prefix_pure  -policy prefix -load-factor 0         -- "${extra[@]}"
  run $scenario prefix       -policy prefix                        -- "${extra[@]}"
done

if [[ -z "${NO_MOCK:-}" ]]; then
  # Failover: kill -9 backend 8002 at t=2s (in-flight streams die), restart at t=6s.
  (
    sleep 2;  kill -9 "${MOCK_PID[8002]}"; echo ">>> killed :8002" >&2
    sleep 4;  ./bin/mockvllm -port 8002 $MOCK_ARGS >>logs/mock8002.log 2>&1 & echo $! > logs/mock8002.pid; echo ">>> restarted :8002" >&2
  ) &
  run failover prefix -policy prefix -- -convs 384
  pids+=("$(cat logs/mock8002.pid 2>/dev/null || true)")
fi

go run ./cmd/report results | tee results/RESULTS.md
