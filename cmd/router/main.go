// Command router is a KV-cache-aware HTTP router for OpenAI-compatible
// inference servers (vLLM, SGLang, TGI...).
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sparkling-snail/kvrouter/internal/router"
)

func main() {
	var (
		listen      = flag.String("listen", ":9000", "listen address")
		backends    = flag.String("backends", "http://127.0.0.1:8001,http://127.0.0.1:8002", "comma-separated backend base URLs")
		policyName  = flag.String("policy", "prefix", "routing policy: prefix | round_robin | least_loaded")
		blockChars  = flag.Int("block-chars", 128, "prefix hash block size in bytes (~32 tokens)")
		indexBlocks = flag.Int("index-blocks", 4096, "per-backend LRU capacity of the router's prefix index")
		loadFactor  = flag.Float64("load-factor", 1.25, "bounded-load factor c: a backend may hold at most ceil(c*avg)+slack in-flight; <=0 disables (pure affinity)")
		slack       = flag.Int64("slack", 2, "extra in-flight allowance on top of the load bound")
		affinity    = flag.Int("affinity-blocks", 8, "blocks hashed for cold-prefix placement (rendezvous key)")
		retries     = flag.Int("retries", 2, "max retries on other backends (pre-first-byte only)")
		hdrTimeout  = flag.Duration("header-timeout", 60*time.Second, "max wait for backend response headers")
		healthEvery = flag.Duration("health-interval", time.Second, "active health check interval")
		healthTO    = flag.Duration("health-timeout", 500*time.Millisecond, "active health check timeout")
		failThresh  = flag.Int64("fail-threshold", 3, "consecutive failures before ejecting a backend")
		verbose     = flag.Bool("v", false, "debug logging")
	)
	flag.Parse()

	lvl := slog.LevelInfo
	if *verbose {
		lvl = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})))

	var urls []string
	for _, u := range strings.Split(*backends, ",") {
		if u = strings.TrimSpace(u); u != "" {
			urls = append(urls, u)
		}
	}
	if len(urls) == 0 {
		fmt.Fprintln(os.Stderr, "no backends")
		os.Exit(2)
	}
	pool := router.NewPool(urls, *indexBlocks)
	pool.FailThreshold = *failThresh

	var policy router.Policy
	switch *policyName {
	case "prefix":
		policy = router.NewPrefixAware(pool, *loadFactor, *slack, *affinity)
	case "round_robin", "rr":
		policy = router.NewRoundRobin(pool)
	case "least_loaded", "ll":
		policy = router.NewLeastLoaded(pool)
	default:
		fmt.Fprintln(os.Stderr, "unknown policy:", *policyName)
		os.Exit(2)
	}

	proxy := router.NewProxy(router.Config{
		BlockChars:            *blockChars,
		MaxRetries:            *retries,
		MaxBodyBytes:          32 << 20,
		ResponseHeaderTimeout: *hdrTimeout,
		HealthInterval:        *healthEvery,
		HealthTimeout:         *healthTO,
	}, pool, policy)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go proxy.RunHealthChecks(ctx)

	srv := &http.Server{Addr: *listen, Handler: proxy.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx) // drain in-flight streams
	}()
	slog.Info("router listening", "addr", *listen, "policy", policy.Name(), "backends", urls,
		"load_factor", *loadFactor, "block_chars", *blockChars)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
