// Command report renders results/*.json from cmd/bench as markdown tables.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type summary struct {
	Scenario     string             `json:"scenario"`
	Policy       string             `json:"policy"`
	Requests     int                `json:"requests"`
	Errors       int                `json:"errors"`
	ReqPerSec    float64            `json:"req_per_sec"`
	OutTokPerSec float64            `json:"output_tok_per_sec"`
	TTFT         map[string]float64 `json:"ttft_ms"`
	ITL          map[string]float64 `json:"itl_ms"`
	Goodput      *struct {
		TTFT      float64 `json:"slo_ttft_ms"`
		TPOT      float64 `json:"slo_tpot_ms"`
		Rate      float64 `json:"rate"`
		ReqPerSec float64 `json:"req_per_sec"`
	} `json:"goodput"`
	CacheHit     float64            `json:"cache_hit_rate"`
	Imbalance    float64            `json:"imbalance_max_over_mean"`
	Index        *struct {
		Predicted float64 `json:"predicted_hit_rate"`
		Warm      int     `json:"predicted_warm"`
		StaleRate float64 `json:"stale_rate"`
	} `json:"index_accuracy"`
	PerBackend map[string]int `json:"per_backend"`
	Reasons    map[string]int `json:"route_reasons"`
	Timeline   []struct {
		Sec  int     `json:"sec"`
		OK   int     `json:"ok"`
		Err  int     `json:"err"`
		TTFT float64 `json:"ttft_p50_ms"`
	} `json:"timeline"`
	Errs map[string]int `json:"error_samples"`
}

var order = map[string]int{"round_robin": 0, "least_loaded": 1, "prefix_pure": 2, "prefix": 3, "weighted": 4, "weighted_load": 5}

func main() {
	dir := "results"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	by := map[string][]summary{}
	var scen []string
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var s summary
		if json.Unmarshal(b, &s) != nil {
			continue
		}
		if _, ok := by[s.Scenario]; !ok {
			scen = append(scen, s.Scenario)
		}
		by[s.Scenario] = append(by[s.Scenario], s)
	}
	sort.Strings(scen)
	for _, sc := range scen {
		rs := by[sc]
		sort.Slice(rs, func(i, j int) bool { return order[rs[i].Policy] < order[rs[j].Policy] })
		var base *summary
		for i := range rs {
			if rs[i].Policy == "round_robin" {
				base = &rs[i]
			}
		}
		fmt.Printf("### %s\n\n", sc)
		for _, r := range rs {
			if g := r.Goodput; g != nil {
				fmt.Printf("Goodput SLO: TTFT <= %.0f ms, TPOT <= %.0f ms (0 = none).\n\n", g.TTFT, g.TPOT)
				break
			}
		}
		fmt.Println("| policy | TTFT p50 | TTFT p90 | TTFT p99 | ITL p50 | ITL p99 | req/s | goodput | out tok/s | cache hit | load imbalance | errors |")
		fmt.Println("|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|")
		for _, r := range rs {
			rel := ""
			if base != nil && r.Policy != "round_robin" && base.ReqPerSec > 0 {
				rel = fmt.Sprintf(" (%.2fx)", r.ReqPerSec/base.ReqPerSec)
			}
			good := "–"
			if g := r.Goodput; g != nil {
				good = fmt.Sprintf("%.1f (%.0f%%)", g.ReqPerSec, 100*g.Rate)
			}
			fmt.Printf("| %s | %.0f ms | %.0f ms | %.0f ms | %.1f ms | %.1f ms | %.1f%s | %s | %.0f | %.1f%% | %.2f | %d/%d |\n",
				r.Policy, r.TTFT["p50"], r.TTFT["p90"], r.TTFT["p99"], r.ITL["p50"], r.ITL["p99"], r.ReqPerSec, rel, good,
				r.OutTokPerSec, 100*r.CacheHit, r.Imbalance, r.Errors, r.Requests)
		}
		fmt.Println()
		for _, r := range rs {
			if r.Index != nil && r.Index.Warm > 0 {
				fmt.Printf("- `%s` router index: predicted hit %.1f%% vs actual %.1f%%; %d requests expected warm, %.1f%% stale\n",
					r.Policy, 100*r.Index.Predicted, 100*r.CacheHit, r.Index.Warm, 100*r.Index.StaleRate)
			}
		}
		for _, r := range rs {
			if len(r.Reasons) > 0 {
				fmt.Printf("- `%s` route reasons: %s\n", r.Policy, kv(r.Reasons))
			}
			if len(r.Errs) > 0 {
				fmt.Printf("- `%s` errors: %s\n", r.Policy, kv(r.Errs))
			}
		}
		if sc == "failover" && len(rs) > 0 {
			fmt.Print("\nPer-second timeline (requests *started* in that second):\n\n")
			fmt.Println("| t (s) | ok | err | TTFT p50 |")
			fmt.Println("|---:|---:|---:|---:|")
			for _, t := range rs[0].Timeline {
				fmt.Printf("| %d | %d | %d | %.0f ms |\n", t.Sec, t.OK, t.Err, t.TTFT)
			}
		}
		fmt.Println()
	}
}

func kv(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return m[keys[i]] > m[keys[j]] })
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	return strings.Join(parts, ", ")
}
