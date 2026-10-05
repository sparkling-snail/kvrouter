"""Average results/seed*/ across seeds -> markdown tables + docs/results.png.

    .venv/bin/python scripts/summarize.py            # prints tables, writes the chart

The README's result tables are this script's output, so they can be regenerated
after re-running scripts/bench.sh for each seed.
"""

import glob
import json
import os
import statistics
import sys
from collections import defaultdict

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
POLICIES = ["round_robin", "least_loaded", "prefix_pure", "prefix", "weighted", "weighted_load"]
LABELS = {
    "round_robin": "round_robin",
    "least_loaded": "least_loaded",
    "prefix_pure": "prefix_pure (no load bound)",
    "prefix": "**prefix** (bounded load)",
    "weighted": "weighted 3:2 (prefix:load)",
    "weighted_load": "weighted 1:2",
}
SCENARIOS = {"multi_tenant": "Multi-tenant", "hot_tenant": "Hot tenant"}


def load(pattern):
    runs = defaultdict(list)  # (scenario, policy) -> [summary, ...]
    for path in sorted(glob.glob(os.path.join(ROOT, pattern))):
        with open(path) as f:
            s = json.load(f)
        runs[(s["scenario"], s["policy"])].append(s)
    return runs


def mean(xs):
    return statistics.fmean(xs) if xs else float("nan")


def table(runs, scenario):
    base = mean([s["req_per_sec"] for s in runs[(scenario, "round_robin")]])
    rows = [
        "| policy | TTFT p50 | TTFT p90 | TTFT p99 | req/s | vs RR | cache hit | load imbalance |",
        "|---|---:|---:|---:|---:|---:|---:|---:|",
    ]
    for p in POLICIES:
        rs = runs.get((scenario, p))
        if not rs:
            continue
        rps = mean([s["req_per_sec"] for s in rs])
        rows.append(
            "| {} | {:.0f} ms | {:.0f} ms | {:.0f} ms | {:.1f} | {:.2f}× | {:.1f}% | {:.2f} |".format(
                LABELS[p],
                mean([s["ttft_ms"]["p50"] for s in rs]),
                mean([s["ttft_ms"]["p90"] for s in rs]),
                mean([s["ttft_ms"]["p99"] for s in rs]),
                rps,
                rps / base,
                100 * mean([s["cache_hit_rate"] for s in rs]),
                mean([s["imbalance_max_over_mean"] for s in rs]),
            )
        )
    return "\n".join(rows)


def drift_table(runs):
    rows = [
        "| router index size | predicted hit | actual hit | expected-warm requests that were stale |",
        "|---|---:|---:|---:|",
    ]
    for scenario in sorted({k[0] for k in runs}, key=lambda s: -int(s.removeprefix("drift_idx"))):
        rs = [s for (sc, _), ss in runs.items() if sc == scenario for s in ss]
        idx = [s["index_accuracy"] for s in rs]
        stale = [i["stale_rate"] for i in idx]
        rows.append(
            "| {} blocks | {:.1f}% | {:.1f}% | {:.1f}% (range {:.0f}–{:.0f}%) |".format(
                scenario.removeprefix("drift_idx"),
                100 * mean([i["predicted_hit_rate"] for i in idx]),
                100 * mean([s["cache_hit_rate"] for s in rs]),
                100 * mean(stale),
                100 * min(stale),
                100 * max(stale),
            )
        )
    return "\n".join(rows)


def chart(runs, out):
    import matplotlib

    matplotlib.use("Agg")
    import matplotlib.pyplot as plt

    policies = [p for p in POLICIES if any((sc, p) in runs for sc in SCENARIOS)]
    names = [p.replace("_", "\n", 1) if p.startswith("weighted_") else p for p in policies]
    colors = ["#9aa0a6" if not p.startswith(("prefix", "weighted")) else "#4c78a8" for p in policies]
    colors = [("#e45756" if p == "prefix" else c) for p, c in zip(policies, colors)]

    fig, axes = plt.subplots(2, 2, figsize=(11, 6.5), constrained_layout=True)
    for col, (sc, title) in enumerate(SCENARIOS.items()):
        for row, (metric, ylabel) in enumerate(
            [(lambda s: s["req_per_sec"], "throughput (req/s)"), (lambda s: s["ttft_ms"]["p50"], "TTFT p50 (ms, log)")]
        ):
            ax = axes[row][col]
            vals = [mean([metric(s) for s in runs.get((sc, p), [])]) for p in policies]
            errs = [statistics.pstdev([metric(s) for s in runs.get((sc, p), [])] or [0]) for p in policies]
            ax.bar(names, vals, yerr=errs, color=colors, capsize=3)
            ax.set_ylabel(ylabel)
            if row == 1:
                ax.set_yscale("log")
                plain = matplotlib.ticker.FuncFormatter(lambda v, _: f"{v:g}")
                ax.yaxis.set_major_formatter(plain)
                ax.yaxis.set_minor_formatter(plain)
                ax.tick_params(axis="y", which="minor", labelsize=7)
            else:
                ax.set_title(title)
            ax.tick_params(axis="x", labelsize=8)
            ax.spines[["top", "right"]].set_visible(False)
            ax.grid(axis="y", alpha=0.3)
    fig.suptitle("4 mock vLLM instances, mean of 3 seeds (error bars: std dev)", fontsize=10)
    os.makedirs(os.path.dirname(out), exist_ok=True)
    fig.savefig(out, dpi=130)
    print(f"wrote {os.path.relpath(out, ROOT)}", file=sys.stderr)


def main():
    runs = load("results/seed*/*.json")
    for sc, title in SCENARIOS.items():
        print(f"**{title}**\n\n{table(runs, sc)}\n")
    drift = load("results/index_drift/*.json")
    if drift:
        print(f"**Index drift** (12k-token KV cache per instance, prefix policy)\n\n{drift_table(drift)}\n")
    chart(runs, os.path.join(ROOT, "docs", "results.png"))


if __name__ == "__main__":
    main()
