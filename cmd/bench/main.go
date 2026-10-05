// Command bench drives an OpenAI-compatible endpoint with a multi-tenant,
// multi-turn chat workload and reports TTFT, throughput and prefix-cache
// hit rate.
//
// Workload shape (the case prefix-aware routing exists for):
//   - T tenants, each with a long system prompt (RAG context, tool specs...)
//   - C conversations; each belongs to a tenant and runs K sequential turns,
//     every turn resending system prompt + full history + a new user message
//   - optional skew: a fraction of conversations all hit tenant 0 (hot key)
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sparkling-snail/kvrouter/internal/prefix"
)

type result struct {
	Start      time.Duration
	TTFT       time.Duration
	E2E        time.Duration
	PromptTok  int
	CachedTok  int
	OutTok     int
	Backend    string
	Reason     string
	Err        string
	AssistText string
	// Router's belief vs reality: bytes of the prompt it expected to be
	// cached (-1 if the router didn't say) and the prompt's length in bytes.
	MatchedChars, PromptChars int
}

// IndexAccuracy compares the router's approximate prefix index with what
// the backend actually reused (usage.prompt_tokens_details.cached_tokens).
// Fractions are of the prompt, so bytes and tokens can be compared.
type IndexAccuracy struct {
	PredictedHitRate float64 `json:"predicted_hit_rate"` // token-weighted, like cache_hit_rate
	PredictedWarm    int     `json:"predicted_warm"`     // requests the router expected to hit
	Stale            int     `json:"stale"`              // ...where the backend reused < half the expected prefix
	StaleRate        float64 `json:"stale_rate"`
}

type Summary struct {
	Scenario      string             `json:"scenario"`
	Policy        string             `json:"policy"`
	Requests      int                `json:"requests"`
	Errors        int                `json:"errors"`
	WallSec       float64            `json:"wall_sec"`
	ReqPerSec     float64            `json:"req_per_sec"`
	OutTokPerSec  float64            `json:"output_tok_per_sec"`
	PromptTokPerS float64            `json:"prompt_tok_per_sec"`
	TTFTms        map[string]float64 `json:"ttft_ms"`
	E2Ems         map[string]float64 `json:"e2e_ms"`
	CacheHitRate  float64            `json:"cache_hit_rate"`
	ScrapedHit    *float64           `json:"scraped_cache_hit_rate,omitempty"`
	Index         *IndexAccuracy     `json:"index_accuracy,omitempty"`
	PerBackend    map[string]int     `json:"per_backend"`
	Imbalance     float64            `json:"imbalance_max_over_mean"`
	Reasons       map[string]int     `json:"route_reasons"`
	Timeline      []bucket           `json:"timeline"`
	Errs          map[string]int     `json:"error_samples,omitempty"`
}

type bucket struct {
	Sec     int     `json:"sec"`
	OK      int     `json:"ok"`
	Err     int     `json:"err"`
	TTFTp50 float64 `json:"ttft_p50_ms"`
}

type msg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

var words = strings.Fields(`alpha bravo charlie delta echo foxtrot golf hotel india juliet kilo lima mike
november oscar papa quebec romeo sierra tango uniform victor whiskey xray yankee zulu cache tensor kernel
block page token batch prefill decode router shard replica latency throughput memory bandwidth scheduler`)

func text(rng *rand.Rand, tokens int) string {
	var b strings.Builder
	for b.Len() < tokens*4 {
		b.WriteString(words[rng.IntN(len(words))])
		b.WriteByte(' ')
	}
	return b.String()
}

func main() {
	var (
		url         = flag.String("url", "http://127.0.0.1:9000", "router (or single server) base URL")
		model       = flag.String("model", "mock", "model name")
		tenants     = flag.Int("tenants", 16, "distinct system prompts")
		sysTokens   = flag.Int("sys-tokens", 3000, "approx tokens per system prompt")
		sharedTok   = flag.Int("shared-tokens", 200, "approx tokens of header shared by ALL tenants (chat template + global instructions)")
		convs       = flag.Int("convs", 128, "conversations")
		turns       = flag.Int("turns", 4, "turns per conversation")
		userTokens  = flag.Int("user-tokens", 60, "approx tokens per user message")
		maxTokens   = flag.Int("max-tokens", 16, "output tokens per request")
		concurrency = flag.Int("concurrency", 48, "concurrent conversations")
		hotFrac     = flag.Float64("hot-frac", 0, "fraction of conversations forced onto tenant 0")
		seed        = flag.Uint64("seed", 42, "workload seed (same seed => identical prompts across runs)")
		scenario    = flag.String("scenario", "default", "label")
		policy      = flag.String("policy", "", "label")
		out         = flag.String("out", "", "write JSON summary here")
		scrape      = flag.String("scrape", "", "comma-separated backend URLs; diff vllm:prefix_cache_* counters")
		timeout     = flag.Duration("timeout", 120*time.Second, "per-request timeout")
	)
	flag.Parse()

	rng := rand.New(rand.NewPCG(*seed, 1))
	// A header shared by every tenant, like a chat template + global policy.
	// Exercises the router's minimum-match rule: matching only this must not
	// count as a cache hit.
	preamble := "You are a helpful assistant deployed on the inference platform. " +
		text(rand.New(rand.NewPCG(7, 7)), *sharedTok)
	sys := make([]string, *tenants)
	for i := range sys {
		sys[i] = preamble + fmt.Sprintf("[tenant %d context]\n", i) + text(rng, *sysTokens)
	}
	type conv struct {
		tenant int
		users  []string
	}
	cs := make([]conv, *convs)
	for i := range cs {
		t := rng.IntN(*tenants)
		if rng.Float64() < *hotFrac {
			t = 0
		}
		cs[i].tenant = t
		for k := 0; k < *turns; k++ {
			cs[i].users = append(cs[i].users, text(rng, *userTokens))
		}
	}

	before := scrapeAll(*scrape)
	client := &http.Client{
		Timeout:   *timeout,
		Transport: &http.Transport{MaxIdleConnsPerHost: 1024, DisableCompression: true},
	}

	work := make(chan int)
	var mu sync.Mutex
	var results []result
	t0 := time.Now()
	var wg sync.WaitGroup
	for w := 0; w < *concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ci := range work {
				c := cs[ci]
				msgs := []msg{{"system", sys[c.tenant]}}
				for _, u := range c.users {
					msgs = append(msgs, msg{"user", u})
					r := do(client, *url, *model, msgs, *maxTokens, t0)
					mu.Lock()
					results = append(results, r)
					mu.Unlock()
					if r.Err != "" {
						break // a real client would surface the error; stop this conversation
					}
					msgs = append(msgs, msg{"assistant", r.AssistText})
				}
			}
		}()
	}
	for i := range cs {
		work <- i
	}
	close(work)
	wg.Wait()
	wall := time.Since(t0)

	s := summarize(results, wall)
	s.Scenario, s.Policy = *scenario, *policy
	if after := scrapeAll(*scrape); before != nil && after != nil {
		q := after[0] - before[0]
		if q > 0 {
			h := (after[1] - before[1]) / q
			s.ScrapedHit = &h
		}
	}
	print(s)
	if *out != "" {
		b, _ := json.MarshalIndent(s, "", "  ")
		if err := os.WriteFile(*out, b, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func do(client *http.Client, url, model string, msgs []msg, maxTokens int, t0 time.Time) result {
	body, _ := json.Marshal(map[string]any{
		"model": model, "messages": msgs, "max_tokens": maxTokens, "stream": true,
		"stream_options": map[string]any{"include_usage": true},
		"temperature":    0, "ignore_eos": true,
	})
	start := time.Now()
	r := result{Start: start.Sub(t0), MatchedChars: -1, PromptChars: len(prefix.ExtractPrompt(body))}
	resp, err := client.Post(url+"/v1/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		r.Err = shortErr(err.Error())
		r.E2E = time.Since(start)
		return r
	}
	defer resp.Body.Close()
	r.Backend = resp.Header.Get("X-Router-Backend")
	if r.Backend == "" {
		r.Backend = resp.Header.Get("X-Mock-Instance")
	}
	r.Reason = resp.Header.Get("X-Router-Reason")
	if v, err := strconv.Atoi(resp.Header.Get("X-Router-Matched-Chars")); err == nil {
		r.MatchedChars = v
	}
	if resp.StatusCode != http.StatusOK {
		r.Err = "http " + strconv.Itoa(resp.StatusCode)
		r.E2E = time.Since(start)
		return r
	}
	var out strings.Builder
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	done := false
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			done = true
			break
		}
		var ch struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				Details          *struct {
					CachedTokens int `json:"cached_tokens"`
				} `json:"prompt_tokens_details"`
			} `json:"usage"`
		}
		if json.Unmarshal([]byte(data), &ch) != nil {
			continue
		}
		for _, c := range ch.Choices {
			if c.Delta.Content != "" {
				if r.TTFT == 0 {
					r.TTFT = time.Since(start)
				}
				out.WriteString(c.Delta.Content)
			}
		}
		if ch.Usage != nil {
			r.PromptTok = ch.Usage.PromptTokens
			r.OutTok = ch.Usage.CompletionTokens
			if ch.Usage.Details != nil {
				r.CachedTok = ch.Usage.Details.CachedTokens
			}
		}
	}
	r.E2E = time.Since(start)
	if err := sc.Err(); err != nil {
		r.Err = shortErr("stream: " + err.Error())
	} else if !done {
		r.Err = "stream truncated"
	}
	r.AssistText = out.String()
	return r
}

var portRe = regexp.MustCompile(`:\d+`)

func shortErr(s string) string {
	if i := strings.LastIndex(s, ": "); i >= 0 && len(s) > 80 {
		s = s[i+2:]
	}
	return s
}

func pct(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	i := int(math.Ceil(p/100*float64(len(xs)))) - 1
	return xs[max(0, min(i, len(xs)-1))]
}

func summarize(rs []result, wall time.Duration) Summary {
	s := Summary{
		Requests: len(rs), WallSec: wall.Seconds(),
		PerBackend: map[string]int{}, Reasons: map[string]int{}, Errs: map[string]int{},
	}
	var ttft, e2e []float64
	var out, prompt, cached int
	var idx *IndexAccuracy
	var predictedTok float64
	var idxPrompt int
	bySec := map[int]*struct {
		ok, err int
		ttft    []float64
	}{}
	for _, r := range rs {
		sec := int(r.Start.Seconds())
		b := bySec[sec]
		if b == nil {
			b = &struct {
				ok, err int
				ttft    []float64
			}{}
			bySec[sec] = b
		}
		if r.Err != "" {
			s.Errors++
			s.Errs[r.Err]++
			b.err++
			continue
		}
		b.ok++
		ms := float64(r.TTFT.Microseconds()) / 1000
		b.ttft = append(b.ttft, ms)
		ttft = append(ttft, ms)
		e2e = append(e2e, float64(r.E2E.Microseconds())/1000)
		out += r.OutTok
		prompt += r.PromptTok
		cached += r.CachedTok
		if r.MatchedChars >= 0 && r.PromptChars > 0 && r.PromptTok > 0 {
			if idx == nil {
				idx = &IndexAccuracy{}
			}
			predicted := min(1, float64(r.MatchedChars)/float64(r.PromptChars))
			actual := float64(r.CachedTok) / float64(r.PromptTok)
			predictedTok += predicted * float64(r.PromptTok)
			idxPrompt += r.PromptTok
			if predicted > 0 {
				idx.PredictedWarm++
				if actual < predicted/2 {
					idx.Stale++
				}
			}
		}
		key := r.Backend
		if m := portRe.FindString(r.Backend); m != "" {
			key = m
		}
		s.PerBackend[key]++
		if r.Reason != "" {
			s.Reasons[r.Reason]++
		}
	}
	sort.Float64s(ttft)
	sort.Float64s(e2e)
	mean := func(xs []float64) float64 {
		t := 0.0
		for _, x := range xs {
			t += x
		}
		return t / math.Max(1, float64(len(xs)))
	}
	s.TTFTms = map[string]float64{"mean": mean(ttft), "p50": pct(ttft, 50), "p90": pct(ttft, 90), "p99": pct(ttft, 99)}
	s.E2Ems = map[string]float64{"mean": mean(e2e), "p50": pct(e2e, 50), "p99": pct(e2e, 99)}
	s.ReqPerSec = float64(len(rs)-s.Errors) / wall.Seconds()
	s.OutTokPerSec = float64(out) / wall.Seconds()
	s.PromptTokPerS = float64(prompt) / wall.Seconds()
	if prompt > 0 {
		s.CacheHitRate = float64(cached) / float64(prompt)
	}
	if idx != nil {
		idx.PredictedHitRate = predictedTok / float64(idxPrompt)
		if idx.PredictedWarm > 0 {
			idx.StaleRate = float64(idx.Stale) / float64(idx.PredictedWarm)
		}
		s.Index = idx
	}
	if len(s.PerBackend) > 0 {
		mx, tot := 0, 0
		for _, n := range s.PerBackend {
			mx = max(mx, n)
			tot += n
		}
		s.Imbalance = float64(mx) / (float64(tot) / float64(len(s.PerBackend)))
	}
	secs := make([]int, 0, len(bySec))
	for k := range bySec {
		secs = append(secs, k)
	}
	sort.Ints(secs)
	for _, k := range secs {
		b := bySec[k]
		sort.Float64s(b.ttft)
		s.Timeline = append(s.Timeline, bucket{Sec: k, OK: b.ok, Err: b.err, TTFTp50: pct(b.ttft, 50)})
	}
	if len(s.Errs) == 0 {
		s.Errs = nil
	}
	return s
}

func print(s Summary) {
	fmt.Printf("== %s / %s ==\n", s.Scenario, s.Policy)
	fmt.Printf("requests=%d errors=%d wall=%.1fs  %.1f req/s  %.0f out tok/s  %.0f prompt tok/s\n",
		s.Requests, s.Errors, s.WallSec, s.ReqPerSec, s.OutTokPerSec, s.PromptTokPerS)
	fmt.Printf("TTFT ms: mean=%.0f p50=%.0f p90=%.0f p99=%.0f   E2E p50=%.0f p99=%.0f\n",
		s.TTFTms["mean"], s.TTFTms["p50"], s.TTFTms["p90"], s.TTFTms["p99"], s.E2Ems["p50"], s.E2Ems["p99"])
	fmt.Printf("cache hit rate=%.1f%%", 100*s.CacheHitRate)
	if s.ScrapedHit != nil {
		fmt.Printf(" (scraped %.1f%%)", 100**s.ScrapedHit)
	}
	fmt.Printf("  imbalance=%.2f  per-backend=%v\n", s.Imbalance, s.PerBackend)
	if s.Index != nil {
		fmt.Printf("router index: predicted hit=%.1f%%  predicted-warm=%d  stale=%d (%.1f%%)\n",
			100*s.Index.PredictedHitRate, s.Index.PredictedWarm, s.Index.Stale, 100*s.Index.StaleRate)
	}
	if len(s.Reasons) > 0 {
		fmt.Printf("route reasons=%v\n", s.Reasons)
	}
	if len(s.Errs) > 0 {
		fmt.Printf("errors=%v\n", s.Errs)
	}
}

// scrapeAll sums vllm:prefix_cache_queries_total and _hits_total across backends.
func scrapeAll(list string) []float64 {
	if list == "" {
		return nil
	}
	var q, h float64
	for _, u := range strings.Split(list, ",") {
		resp, err := http.Get(strings.TrimRight(u, "/") + "/metrics")
		if err != nil {
			return nil
		}
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			line := sc.Text()
			f := strings.Fields(line)
			if len(f) < 2 {
				continue
			}
			v, _ := strconv.ParseFloat(f[len(f)-1], 64)
			switch {
			case strings.HasPrefix(line, "vllm:prefix_cache_queries_total"):
				q += v
			case strings.HasPrefix(line, "vllm:prefix_cache_hits_total"):
				h += v
			}
		}
		resp.Body.Close()
	}
	return []float64{q, h}
}
