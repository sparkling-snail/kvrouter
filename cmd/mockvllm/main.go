// Command mockvllm is a GPU-free stand-in for a vLLM OpenAI server that
// models the two things a cache-aware router can change:
//
//  1. Automatic prefix caching: an LRU of chained block hashes with a fixed
//     capacity (the KV-cache pool). Cached prompt tokens skip prefill.
//  2. Prefill is compute-bound and contends: prefills on one instance are
//     serialized through a single "GPU", so every uncached token adds
//     queueing delay for everyone behind it. That's where TTFT comes from.
//
// Decode is per-token sleeps that slow down slightly as the batch grows.
// It speaks enough of the OpenAI API (/v1/completions, /v1/chat/completions,
// streaming SSE with usage.prompt_tokens_details.cached_tokens) plus vLLM-
// style /health and /metrics for the router and benchmark to work unchanged
// against real vLLM.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sparkling-snail/kvrouter/internal/lru"
	"github.com/sparkling-snail/kvrouter/internal/prefix"
)

const charsPerToken = 4

type server struct {
	id            string
	blockTokens   int
	cache         *lru.Set
	prefillPerTok time.Duration
	prefillFixed  time.Duration
	decodeStep    time.Duration
	maxSeqs       int

	gpu   sync.Mutex    // serializes prefill compute
	slots chan struct{} // max_num_seqs admission

	running, waiting             atomic.Int64
	promptToks, cachedToks, reqs atomic.Int64
	faultRate                    atomic.Uint64 // float64 bits
	down                         atomic.Bool
}

var vocab = []string{" the", " model", " cache", " token", " is", " fast", " and", " of", " a", " block", " GPU", " prefix", " served", " to", " request"}

func main() {
	port := flag.Int("port", 8001, "listen port")
	id := flag.String("id", "", "instance id (default: port)")
	cacheTokens := flag.Int("cache-tokens", 32768, "KV cache capacity in tokens (shared prefix-cache pool)")
	blockTokens := flag.Int("block-tokens", 16, "KV block size in tokens (vLLM default 16)")
	prefillUs := flag.Float64("prefill-us-per-token", 100, "prefill cost per uncached token, microseconds (~10k tok/s)")
	prefillFixed := flag.Duration("prefill-fixed", 2*time.Millisecond, "fixed per-request prefill overhead")
	decodeStep := flag.Duration("decode-step", 10*time.Millisecond, "base time per decode step")
	maxSeqs := flag.Int("max-num-seqs", 32, "max concurrently running sequences")
	flag.Parse()
	if *id == "" {
		*id = strconv.Itoa(*port)
	}

	s := &server{
		id:            *id,
		blockTokens:   *blockTokens,
		cache:         lru.New(*cacheTokens / *blockTokens),
		prefillPerTok: time.Duration(*prefillUs * float64(time.Microsecond)),
		prefillFixed:  *prefillFixed,
		decodeStep:    *decodeStep,
		maxSeqs:       *maxSeqs,
		slots:         make(chan struct{}, *maxSeqs),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/completions", s.generate(false))
	mux.HandleFunc("POST /v1/chat/completions", s.generate(true))
	mux.HandleFunc("GET /v1/models", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"object": "list", "data": []any{map[string]any{"id": "mock", "object": "model"}}})
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		if s.down.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /metrics", s.metrics)
	reset := func(w http.ResponseWriter, _ *http.Request) {
		s.cache.Reset()
		s.promptToks.Store(0)
		s.cachedToks.Store(0)
		s.reqs.Store(0)
		writeJSON(w, map[string]any{"success": true})
	}
	mux.HandleFunc("POST /reset", reset)
	mux.HandleFunc("POST /reset_prefix_cache", reset) // vLLM's name (VLLM_SERVER_DEV_MODE=1)
	// Fault injection for testing retries / health checks.
	mux.HandleFunc("POST /admin/fault", func(w http.ResponseWriter, r *http.Request) {
		rate, _ := strconv.ParseFloat(r.URL.Query().Get("rate"), 64)
		s.faultRate.Store(math.Float64bits(rate))
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /admin/down", func(w http.ResponseWriter, _ *http.Request) { s.down.Store(true); w.WriteHeader(204) })
	mux.HandleFunc("POST /admin/up", func(w http.ResponseWriter, _ *http.Request) { s.down.Store(false); w.WriteHeader(204) })

	slog.Info("mock vllm listening", "port", *port, "cache_tokens", *cacheTokens, "max_num_seqs", *maxSeqs)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), mux); err != nil {
		slog.Error("listen", "err", err)
		os.Exit(1)
	}
}

type genReq struct {
	MaxTokens int  `json:"max_tokens"`
	Stream    bool `json:"stream"`
}

func (s *server) generate(chat bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.down.Load() || rand.Float64() < math.Float64frombits(s.faultRate.Load()) {
			http.Error(w, "injected fault", http.StatusServiceUnavailable)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var gr genReq
		_ = json.Unmarshal(body, &gr)
		if gr.MaxTokens <= 0 {
			gr.MaxTokens = 16
		}
		prompt := prefix.ExtractPrompt(body)
		promptTokens := max(1, len(prompt)/charsPerToken)
		hashes := prefix.BlockHashes(prompt, s.blockTokens*charsPerToken)
		ctx := r.Context()

		// Admission (max_num_seqs). Waiting here is queueing time.
		s.waiting.Add(1)
		select {
		case s.slots <- struct{}{}:
			s.waiting.Add(-1)
		case <-ctx.Done():
			s.waiting.Add(-1)
			return
		}
		s.running.Add(1)
		defer func() { <-s.slots; s.running.Add(-1) }()

		// Prefix cache lookup at schedule time.
		matched := s.cache.MatchAndInsert(hashes)
		cached := min(matched*s.blockTokens, promptTokens-1) // vLLM always recomputes the last token
		cached = max(cached, 0)
		s.reqs.Add(1)
		s.promptToks.Add(int64(promptTokens))
		s.cachedToks.Add(int64(cached))

		// Prefill: serialized on the "GPU".
		s.gpu.Lock()
		ok := sleepCtx(ctx, s.prefillFixed+time.Duration(promptTokens-cached)*s.prefillPerTok)
		s.gpu.Unlock()
		if !ok {
			return
		}

		id := fmt.Sprintf("cmpl-%s-%d", s.id, rand.Uint64())
		usage := map[string]any{
			"prompt_tokens": promptTokens, "completion_tokens": gr.MaxTokens,
			"total_tokens":          promptTokens + gr.MaxTokens,
			"prompt_tokens_details": map[string]any{"cached_tokens": cached},
		}
		w.Header().Set("X-Mock-Instance", s.id)

		if !gr.Stream {
			var text string
			for i := 0; i < gr.MaxTokens; i++ {
				if i > 0 && !sleepCtx(ctx, s.step()) {
					return
				}
				text += vocab[rand.IntN(len(vocab))]
			}
			if chat {
				writeJSON(w, map[string]any{"id": id, "object": "chat.completion", "model": "mock",
					"choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": text}, "finish_reason": "length"}},
					"usage":   usage})
			} else {
				writeJSON(w, map[string]any{"id": id, "object": "text_completion", "model": "mock",
					"choices": []any{map[string]any{"index": 0, "text": text, "finish_reason": "length"}},
					"usage":   usage})
			}
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		rc := http.NewResponseController(w)
		for i := 0; i < gr.MaxTokens; i++ {
			if i > 0 && !sleepCtx(ctx, s.step()) {
				return
			}
			tok := vocab[rand.IntN(len(vocab))]
			var finish any
			if i == gr.MaxTokens-1 {
				finish = "length"
			}
			var chunk map[string]any
			if chat {
				chunk = map[string]any{"id": id, "object": "chat.completion.chunk", "model": "mock",
					"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"content": tok}, "finish_reason": finish}}}
			} else {
				chunk = map[string]any{"id": id, "object": "text_completion", "model": "mock",
					"choices": []any{map[string]any{"index": 0, "text": tok, "finish_reason": finish}}}
			}
			if !sse(w, chunk) {
				return
			}
			_ = rc.Flush()
		}
		sse(w, map[string]any{"id": id, "object": "chat.completion.chunk", "model": "mock", "choices": []any{}, "usage": usage})
		fmt.Fprint(w, "data: [DONE]\n\n")
		_ = rc.Flush()
	}
}

// step: decode gets slower as the running batch grows (memory-bandwidth bound).
func (s *server) step() time.Duration {
	return time.Duration(float64(s.decodeStep) * (1 + float64(s.running.Load())/float64(2*s.maxSeqs)))
}

func (s *server) metrics(w http.ResponseWriter, _ *http.Request) {
	l := `{model_name="mock",instance="` + s.id + `"}`
	fmt.Fprintf(w, "vllm:num_requests_running%s %d\n", l, s.running.Load())
	fmt.Fprintf(w, "vllm:num_requests_waiting%s %d\n", l, s.waiting.Load())
	fmt.Fprintf(w, "vllm:prefix_cache_queries_total%s %d\n", l, s.promptToks.Load())
	fmt.Fprintf(w, "vllm:prefix_cache_hits_total%s %d\n", l, s.cachedToks.Load())
	fmt.Fprintf(w, "vllm:request_success_total%s %d\n", l, s.reqs.Load())
	fmt.Fprintf(w, "mock_cache_blocks%s %d\n", l, s.cache.Len())
}

func sse(w io.Writer, v any) bool {
	b, _ := json.Marshal(v)
	_, err := fmt.Fprintf(w, "data: %s\n\n", b)
	return err == nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}
