#!/usr/bin/env bash
# Runs the policy comparison (scripts/bench.sh) against real vLLM on one GPU host:
# preflight checks, start $NUM_GPUS vLLM servers, wait for health, size the
# workload from the servers' KV-cache capacity, run every policy with cold
# caches, then record what was run next to the results.
#
#   git clone <repo> && cd kvrouter
#   HF_TOKEN=hf_... scripts/gpu_run.sh                  # 4 GPUs, Qwen2.5-7B-Instruct
#   NUM_GPUS=2 MODEL=meta-llama/Llama-3.1-8B-Instruct scripts/gpu_run.sh
#
# Knobs (env): NUM_GPUS MODEL VLLM_VERSION SEED OUT SYS_TOKENS TENANTS CONVS
#   MAX_TOKENS CONCURRENCY SLO_TTFT SLO_TPOT KEEP_UP=1 (leave servers running)
# Needs: NVIDIA driver, docker + compose plugin, nvidia-container-toolkit, curl.
# Go is optional; without it the binaries are built in a golang container.
set -euo pipefail
cd "$(dirname "$0")/.."

NUM_GPUS=${NUM_GPUS:-4}
export MODEL=${MODEL:-Qwen/Qwen2.5-7B-Instruct}
export VLLM_VERSION=${VLLM_VERSION:-v0.31.0}
SEED=${SEED:-1}
OUT=${OUT:-results/gpu/$(date -u +%Y%m%dT%H%M%SZ)}
SYS_TOKENS=${SYS_TOKENS:-3000}
MAX_TOKENS=${MAX_TOKENS:-64}
CONCURRENCY=${CONCURRENCY:-48}
SLO_TTFT=${SLO_TTFT:-1s}
SLO_TPOT=${SLO_TPOT:-50ms}
COMPOSE="docker compose -f deploy/docker-compose.vllm.yml"

die() { echo "gpu_run: $*" >&2; exit 1; }
log() { echo ">>> $*" >&2; }

# ---------------------------------------------------------------- preflight
(( NUM_GPUS >= 2 && NUM_GPUS <= 4 )) || die "NUM_GPUS must be 2-4 (compose defines 4 servers)"
command -v nvidia-smi >/dev/null || die "nvidia-smi not found: no NVIDIA driver on this host"
have=$(nvidia-smi -L | grep -c '^GPU' || true)
(( have >= NUM_GPUS )) || die "need $NUM_GPUS GPUs, nvidia-smi sees $have"
docker info 2>/dev/null | grep -qi nvidia || die "docker has no nvidia runtime (install nvidia-container-toolkit)"
$COMPOSE version >/dev/null 2>&1 || die "docker compose plugin missing"
[[ -n "${HF_TOKEN:-}" ]] || log "HF_TOKEN unset: fine for ungated models, fails for gated ones (e.g. Llama)"
nvidia-smi --query-gpu=index,name,memory.total,driver_version --format=csv,noheader >&2

# ---------------------------------------------------------------- build
if command -v go >/dev/null; then
  go build -o bin/ ./cmd/...
else
  log "no Go on host; building in golang:1.24"
  docker run --rm -u "$(id -u):$(id -g)" -e GOCACHE=/tmp/gocache -e GOPATH=/tmp/go -e CGO_ENABLED=0 \
    -v "$PWD":/src -w /src golang:1.24 go build -buildvcs=false -o bin/ ./cmd/...
fi

# ---------------------------------------------------------------- servers
services=""; backends=""
for (( i = 0; i < NUM_GPUS; i++ )); do
  services="$services vllm-$i"
  backends="$backends,http://127.0.0.1:$((8001 + i))"
done
backends=${backends#,}

[[ -n "${KEEP_UP:-}" ]] || trap '$COMPOSE down >/dev/null 2>&1 || true' EXIT
log "starting$services ($MODEL, vllm $VLLM_VERSION)"
$COMPOSE up -d $services

# First start downloads the image and weights; allow 30 min.
deadline=$(( $(date +%s) + 1800 ))
for b in ${backends//,/ }; do
  until curl -fsS "$b/health" >/dev/null 2>&1; do
    if $COMPOSE ps --status exited --services | grep -q .; then
      $COMPOSE logs --tail 40 >&2; die "a vLLM server exited during startup"
    fi
    (( $(date +%s) < deadline )) || { $COMPOSE logs --tail 40 >&2; die "$b not healthy after 30 min"; }
    sleep 10
  done
  log "$b healthy"
done

# ---------------------------------------------------------------- workload size
# Size the workload so distinct prefixes overflow one server's KV cache but fit
# the fleet's: about half the fleet's capacity, and at least 1.5 servers' worth.
# That's the regime prefix routing is for. If every tenant fits on every
# server, round-robin hit rates converge with prefix routing.
kv=$($COMPOSE logs vllm-0 2>&1 | grep -o 'KV cache size: [0-9,]* tokens' | head -1 | tr -dc 0-9)
[[ -n "$kv" ]] || die "couldn't read 'KV cache size' from vllm-0's log; set TENANTS by hand"
target=$(( kv * NUM_GPUS / 2 )); (( target * 2 >= kv * 3 )) || target=$(( kv * 3 / 2 ))
TENANTS=${TENANTS:-$(( (target + SYS_TOKENS - 1) / SYS_TOKENS ))}
CONVS=${CONVS:-$(( TENANTS * 6 > 128 ? TENANTS * 6 : 128 ))}
log "KV cache: $kv tokens/server → $TENANTS tenants × $SYS_TOKENS tokens, $CONVS conversations"

# ---------------------------------------------------------------- record what ran
mkdir -p "$OUT"
versions=""
for b in ${backends//,/ }; do versions="$versions,$(curl -fsS "$b/version" || echo '{}')"; done
gpus=$(nvidia-smi --query-gpu=name,memory.total,driver_version --format=csv,noheader |
  head -n "$NUM_GPUS" | sed 's/.*/"&"/' | paste -sd, -)
BENCH_ARGS="-tenants $TENANTS -sys-tokens $SYS_TOKENS -convs $CONVS -max-tokens $MAX_TOKENS \
-concurrency $CONCURRENCY -slo-ttft $SLO_TTFT -slo-tpot $SLO_TPOT -timeout 300s"
cat >"$OUT/meta.json" <<EOF
{
  "date": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "git": "$(git rev-parse --short HEAD 2>/dev/null)$(git diff --quiet 2>/dev/null || echo -dirty)",
  "model": "$MODEL",
  "vllm_image": "vllm/vllm-openai:$VLLM_VERSION",
  "vllm_version": [${versions#,}],
  "gpus": [$gpus],
  "num_servers": $NUM_GPUS,
  "kv_cache_tokens_per_server": $kv,
  "seed": $SEED,
  "bench_args": "$BENCH_ARGS"
}
EOF

# ---------------------------------------------------------------- run
log "running all policies → $OUT"
NO_MOCK=1 SKIP_BUILD=1 BACKENDS="$backends" SEED="$SEED" OUT="$OUT" BENCH_ARGS="$BENCH_ARGS" \
  ./scripts/bench.sh
$COMPOSE logs --no-color 2>&1 | gzip >"$OUT/vllm.log.gz"

log "done. Copy the results to your machine with:"
src=$OUT; [[ $src == /* ]] || src=$PWD/$OUT
echo "  rsync -avz $(whoami)@$(hostname -f 2>/dev/null || hostname):$src/ results/gpu/$(basename "$OUT")/"
