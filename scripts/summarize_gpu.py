"""Real-GPU load sweep (results/gpu/<prefix>_c<concurrency>/) -> markdown tables + chart.

    .venv/bin/python scripts/summarize_gpu.py                 # results/gpu/h100x2_c*
    .venv/bin/python scripts/summarize_gpu.py results/gpu/l4x4

Prints goodput by concurrency for each scenario, a detail table at the highest
concurrency, and writes docs/gpu_goodput.png.
"""

import glob
import json
import os
import re
import sys

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
# Reference categorical palette (dataviz skill), fixed slot per policy; validated
# for CVD separation. Three slots are < 3:1 on white, so markers differ too and
# the README carries the same numbers as a table.
STYLE = {
    "prefix": ("#2a78d6", "o"),
    "prefix_pure": ("#eb6834", "s"),
    "weighted": ("#1baf7a", "^"),
    "weighted_load": ("#eda100", "v"),
    "least_loaded": ("#e87ba4", "D"),
    "round_robin": ("#008300", "X"),
}


def load(prefix):
    runs = {}  # concurrency -> {(scenario, policy): summary}
    meta = None
    for d in sorted(glob.glob(prefix + "_c*")):
        m = re.search(r"_c(\d+)$", d)
        if not m:
            continue
        c = int(m.group(1))
        runs[c] = {}
        for path in glob.glob(os.path.join(d, "*.json")):
            with open(path) as f:
                s = json.load(f)
            if "scenario" in s:
                runs[c][(s["scenario"], s["policy"])] = s
            elif meta is None:
                meta = s
    return runs, meta


def sweep_table(runs, scenario):
    cs = sorted(runs)
    head = "| policy | " + " | ".join(f"{c} concurrent" for c in cs) + " |"
    rows = [head, "|---|" + "---:|" * len(cs)]
    for p in POLICIES:
        cells = []
        for c in cs:
            s = runs[c].get((scenario, p))
            g = s and s.get("goodput")
            cells.append(f"{g['req_per_sec']:.0f} ({100 * g['rate']:.0f}%)" if g else "–")
        rows.append(f"| {LABELS[p]} | " + " | ".join(cells) + " |")
    return "\n".join(rows)


def detail_table(rs):
    rows = [
        "| policy | TTFT p50 | TTFT p90 | ITL p99 | TPOT p90 | req/s | goodput | cache hit | load imbalance |",
        "|---|---:|---:|---:|---:|---:|---:|---:|---:|",
    ]
    for p in POLICIES:
        s = rs.get(p)
        if not s:
            continue
        g = s["goodput"]
        rows.append(
            "| {} | {:.0f} ms | {:.0f} ms | {:.0f} ms | {:.1f} ms | {:.1f} | {:.1f} ({:.0f}%) | {:.1f}% | {:.2f} |".format(
                LABELS[p],
                s["ttft_ms"]["p50"],
                s["ttft_ms"]["p90"],
                s["itl_ms"]["p99"],
                s["tpot_ms"]["p90"],
                s["req_per_sec"],
                g["req_per_sec"],
                100 * g["rate"],
                100 * s["cache_hit_rate"],
                s["imbalance_max_over_mean"],
            )
        )
    return "\n".join(rows)


def chart(runs, meta, out):
    import matplotlib

    matplotlib.use("Agg")
    import matplotlib.pyplot as plt

    cs = sorted(runs)
    ymax = max(s["goodput"]["req_per_sec"] for r in runs.values() for s in r.values()) * 1.12
    fig, axes = plt.subplots(1, 2, figsize=(11, 4.4), sharey=True, constrained_layout=True)
    fig.patch.set_facecolor("#fcfcfb")
    for ax, (sc, title) in zip(axes, SCENARIOS.items()):
        ax.set_facecolor("#fcfcfb")
        for p in POLICIES:
            ys = [runs[c][(sc, p)]["goodput"]["req_per_sec"] for c in cs if (sc, p) in runs[c]]
            color, marker = STYLE[p]
            hero = p == "prefix"
            ax.plot(cs, ys, color=color, marker=marker, markersize=8 if hero else 7,
                    linewidth=2.5 if hero else 2, label=p, zorder=3 if hero else 2,
                    markeredgecolor="#fcfcfb", markeredgewidth=1.5)
            if hero:
                ax.annotate(f"prefix {ys[-1]:.0f}", (cs[-1], ys[-1]), xytext=(8, 0),
                            textcoords="offset points", va="center", fontsize=9, color="#21211f")
        ax.set_xscale("log", base=2)
        ax.set_xticks(cs, [str(c) for c in cs])
        ax.minorticks_off()
        ax.set_xlim(cs[0] / 1.25, cs[-1] * 1.5)
        ax.set_ylim(0, ymax)
        ax.set_title(title, fontsize=11, color="#21211f")
        ax.set_xlabel("concurrent conversations", color="#55544f")
        ax.tick_params(colors="#55544f", labelsize=9)
        ax.spines[["top", "right"]].set_visible(False)
        ax.spines[["left", "bottom"]].set_color("#c8c7c0")
        ax.grid(axis="y", color="#e6e5df", linewidth=0.8)
    axes[0].set_ylabel("goodput (req/s meeting SLO)", color="#55544f")
    axes[1].legend(frameon=False, fontsize=9, loc="upper left")
    slo = runs[cs[0]][("multi_tenant", "prefix")]["goodput"]
    gpus = (meta or {}).get("gpus", ["?"])
    fig.suptitle(
        f"{len(gpus)}× {gpus[0].split(',')[0]}, {(meta or {}).get('model', '?')}, vLLM "
        f"{(meta or {}).get('vllm_version', [{}])[0].get('version', '?')} · "
        f"SLO: TTFT ≤ {slo['slo_ttft_ms']:.0f} ms, TPOT ≤ {slo['slo_tpot_ms']:.0f} ms",
        fontsize=10, color="#21211f",
    )
    os.makedirs(os.path.dirname(out), exist_ok=True)
    fig.savefig(out, dpi=130, facecolor=fig.get_facecolor())
    print(f"wrote {os.path.relpath(out, ROOT)}", file=sys.stderr)


def main():
    prefix = sys.argv[1] if len(sys.argv) > 1 else os.path.join(ROOT, "results", "gpu", "h100x2")
    runs, meta = load(prefix)
    if not runs:
        sys.exit(f"no {prefix}_c*/ directories")
    for sc, title in SCENARIOS.items():
        print(f"**{title}**: goodput, req/s (share of requests meeting the SLO)\n\n{sweep_table(runs, sc)}\n")
    top = max(runs)
    for sc, title in SCENARIOS.items():
        rs = {p: s for (scn, p), s in runs[top].items() if scn == sc}
        print(f"**{title} at {top} concurrent**\n\n{detail_table(rs)}\n")
    chart(runs, meta, os.path.join(ROOT, "docs", "gpu_goodput.png"))


if __name__ == "__main__":
    main()
