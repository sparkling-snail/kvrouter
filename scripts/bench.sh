#!/usr/bin/env bash
# Runs the full comparison against 4 mock vLLM instances:
#   scenario multi_tenant : 16 tenants, uniform traffic
#   scenario hot_tenant   : 50% of conversations hit one tenant
#   policies              : round_robin, least_loaded, prefix_pure (no load bound), prefix,
#                           weighted (prefix:load = 3:2), weighted_load (1:2)
#   scenario failover     : prefix policy; kill -9 one backend mid-run, restart it later
# Results land in $OUT/*.json (default results/); a markdown table is printed at the end.
#   SEED=2 OUT=results/seed2 scripts/bench.sh     # one seed into its own directory
#
# To run against real vLLM instead, skip the mocks and set BACKENDS, e.g.
#   BACKENDS=http://gpu1:8000,http://gpu2:8000 MODEL=meta-llama/Llama-3.1-8B-Instruct NO_MOCK=1 scripts/bench.sh
# The servers must run with VLLM_SERVER_DEV_MODE=1 so caches can be reset between
# policies; scripts/gpu_run.sh sets all of this up on a GPU host.
# Without Go on the host, build bin/ first and set SKIP_BUILD=1.
#
# Written for bash 3.2 (macOS default): no associative arrays, no empty "${arr[@]}" under set -u.
set -euo pipefail
cd "$(dirname "$0")/.."

PORTS="8001 8002 8003 8004"
BACKENDS=${BACKENDS:-$(for p in $PORTS; do printf 'http://127.0.0.1:%s,' "$p"; done | sed 's/,$//')}
MODEL=${MODEL:-mock}
MOCK_ARGS=${MOCK_ARGS:-}
BENCH_ARGS=${BENCH_ARGS:-}
SEED=${SEED:-42}
OUT=${OUT:-results}
mkdir -p bin "$OUT" logs
[[ -n "${SKIP_BUILD:-}" ]] || go build -o bin/ ./cmd/...

pids=""
cleanup() { [[ -n $pids ]] && kill $pids 2>/dev/null || true; }
trap cleanup EXIT

start_mock() { ./bin/mockvllm -port "$1" $MOCK_ARGS >>"logs/mock$1.log" 2>&1 & echo $! >"logs/mock$1.pid"; pids="$pids $!"; }
if [[ -z "${NO_MOCK:-}" ]]; then
  for p in $PORTS; do start_mock "$p"; done
  sleep 0.5
fi

# Cold caches for every policy. vLLM refuses ({"success": false}) while blocks
# are still held, e.g. by requests the previous run's client abandoned.
reset_backends() {
  local b i
  for b in ${BACKENDS//,/ }; do
    for i in 1 2 3 4 5 6 7 8 9 10; do
      curl -fsS -XPOST "$b/reset_prefix_cache" 2>/dev/null | grep -q '"success": *true' && continue 2
      sleep 1
    done
    echo "cache reset failed on $b (vLLM needs VLLM_SERVER_DEV_MODE=1)" >&2; exit 1
  done
}

run() { # scenario label "router args" bench-args...
  local scenario=$1 label=$2 rargs=$3; shift 3
  reset_backends
  ./bin/router -listen :9000 -backends "$BACKENDS" $rargs 2>"logs/router_${scenario}_${label}.log" &
  local rp=$!; pids="$pids $rp"; sleep 0.3
  ./bin/bench -url http://127.0.0.1:9000 -model "$MODEL" -scenario "$scenario" -policy "$label" \
      -seed "$SEED" -scrape "$BACKENDS" -out "$OUT/${scenario}_${label}.json" $BENCH_ARGS "$@"
  curl -fsS 127.0.0.1:9000/metrics > "$OUT/${scenario}_${label}.router.prom" || true
  kill "$rp"; wait "$rp" 2>/dev/null || true
  echo
}

for scenario in multi_tenant hot_tenant; do
  extra="-hot-frac 0"
  [[ $scenario == hot_tenant ]] && extra="-hot-frac 0.5"
  run $scenario round_robin   "-policy round_robin"                           $extra
  run $scenario least_loaded  "-policy least_loaded"                          $extra
  run $scenario prefix_pure   "-policy prefix -load-factor 0"                 $extra
  run $scenario prefix        "-policy prefix"                                $extra
  run $scenario weighted      "-policy weighted -prefix-weight 3 -load-weight 2" $extra
  run $scenario weighted_load "-policy weighted -prefix-weight 1 -load-weight 2" $extra
done

if [[ -z "${NO_MOCK:-}" ]]; then
  # Failover: kill -9 backend 8002 at t=2s (in-flight streams die), restart at t=6s.
  (
    sleep 2;  kill -9 "$(cat logs/mock8002.pid)"; echo ">>> killed :8002" >&2
    sleep 4;  start_mock 8002; echo ">>> restarted :8002" >&2
  ) &
  run failover prefix "-policy prefix" -convs 384
  pids="$pids $(cat logs/mock8002.pid 2>/dev/null || true)"
fi

./bin/report "$OUT" | tee "$OUT/RESULTS.md"
