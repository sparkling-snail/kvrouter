package router

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/sparkling-snail/kvrouter/internal/prefix"
)

type Config struct {
	BlockChars            int           // router hash block size, in bytes of prompt text
	MaxRetries            int           // extra attempts on other backends, before first byte only
	MaxBodyBytes          int64         // request size cap
	ResponseHeaderTimeout time.Duration // includes queueing + prefill on the backend
	HealthInterval        time.Duration
	HealthTimeout         time.Duration
}

type Proxy struct {
	cfg     Config
	pool    *Pool
	policy  Policy
	client  *http.Client
	metrics *Metrics
}

var bufPool = sync.Pool{New: func() any { b := make([]byte, 32<<10); return &b }}

// hop-by-hop headers must not be forwarded (RFC 7230 §6.1).
var hopHeaders = map[string]bool{
	"Connection": true, "Keep-Alive": true, "Proxy-Authenticate": true,
	"Proxy-Authorization": true, "Te": true, "Trailer": true,
	"Transfer-Encoding": true, "Upgrade": true,
}

func NewProxy(cfg Config, pool *Pool, policy Policy) *Proxy {
	tr := &http.Transport{
		DialContext: (&net.Dialer{Timeout: 2 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		// Keep plenty of warm connections per backend: under load the
		// router holds one connection per in-flight stream.
		MaxIdleConns:          2048,
		MaxIdleConnsPerHost:   512,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
		DisableCompression:    true, // never buffer SSE behind gzip
	}
	return &Proxy{cfg: cfg, pool: pool, policy: policy, client: &http.Client{Transport: tr}, metrics: NewMetrics()}
}

func (p *Proxy) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/completions", p.serveInference)
	mux.HandleFunc("POST /v1/chat/completions", p.serveInference)
	mux.HandleFunc("GET /v1/models", p.serveInference)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		for _, b := range p.pool.Backends {
			if b.Healthy() {
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	mux.HandleFunc("GET /metrics", p.metrics.Handler(p.pool))
	mux.HandleFunc("GET /debug/backends", p.serveDebug)
	return mux
}

func (p *Proxy) serveDebug(w http.ResponseWriter, _ *http.Request) {
	type row struct {
		URL         string `json:"url"`
		Healthy     bool   `json:"healthy"`
		Inflight    int64  `json:"inflight"`
		Served      int64  `json:"served"`
		IndexBlocks int    `json:"index_blocks"`
	}
	var rows []row
	for _, b := range p.pool.Backends {
		rows = append(rows, row{b.URL, b.Healthy(), b.Inflight(), b.served.Load(), b.index.Len()})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"policy": p.policy.Name(), "backends": rows})
}

func (p *Proxy) serveInference(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, p.cfg.MaxBodyBytes+1))
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if int64(len(body)) > p.cfg.MaxBodyBytes {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	hashes := prefix.BlockHashes(prefix.ExtractPrompt(body), p.cfg.BlockChars)

	tried := make(map[int]bool, 2)
	lastStatus, lastErr := http.StatusServiceUnavailable, errors.New("no backend available")
	for attempt := 0; attempt <= p.cfg.MaxRetries; attempt++ {
		d := p.policy.Pick(hashes, tried)
		if d.Backend == nil {
			break
		}
		tried[d.Backend.ID] = true
		p.metrics.Inc(fmt.Sprintf(`kvrouter_route_decisions_total{policy=%q,reason=%q}`, p.policy.Name(), d.Reason))
		if attempt > 0 {
			p.metrics.Inc(`kvrouter_retries_total`)
		}
		committed, status, err := p.try(w, r, d, hashes, body, attempt)
		if committed {
			return
		}
		lastStatus, lastErr = status, err
		slog.Debug("attempt failed", "backend", d.Backend.URL, "attempt", attempt, "err", err)
	}
	p.metrics.Inc(fmt.Sprintf(`kvrouter_requests_failed_total{status="%d"}`, lastStatus))
	http.Error(w, lastErr.Error(), lastStatus)
}

// try forwards to one backend. committed=true means a response (or a client
// disconnect) has been handled and the caller must not retry. Retries are
// only safe before the first byte goes to the client: once tokens have
// streamed, replaying on another backend would duplicate output.
func (p *Proxy) try(w http.ResponseWriter, r *http.Request, d Decision, hashes []uint64, body []byte, attempt int) (committed bool, status int, err error) {
	b := d.Backend
	b.inflight.Add(1)
	defer b.inflight.Add(-1)

	req, err := http.NewRequestWithContext(r.Context(), r.Method, b.URL+r.URL.RequestURI(), bytes.NewReader(body))
	if err != nil {
		return false, http.StatusInternalServerError, err
	}
	for _, h := range []string{"Content-Type", "Accept", "Authorization"} {
		if v := r.Header.Get(h); v != "" {
			req.Header.Set(h, v)
		}
	}

	start := time.Now()
	resp, err := p.client.Do(req)
	if err != nil {
		if r.Context().Err() != nil { // client went away; nothing to retry for
			p.metrics.Inc(`kvrouter_client_cancelled_total`)
			return true, 499, err
		}
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			p.pool.ReportFailure(b, "timeout") // slow != dead; don't eject on one timeout
		} else {
			p.pool.ReportConnError(b, err)
		}
		return false, http.StatusBadGateway, fmt.Errorf("%s: %w", b.URL, err)
	}
	if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			p.pool.ReportFailure(b, "http "+strconv.Itoa(resp.StatusCode))
		}
		return false, resp.StatusCode, fmt.Errorf("%s returned %d", b.URL, resp.StatusCode)
	}
	defer resp.Body.Close()

	p.pool.ReportSuccess(b)
	p.policy.Observe(b, hashes)
	b.served.Add(1)

	h := w.Header()
	for k, vs := range resp.Header {
		if !hopHeaders[k] {
			h[k] = vs
		}
	}
	h.Set("X-Router-Backend", b.URL)
	h.Set("X-Router-Reason", d.Reason)
	h.Set("X-Router-Matched-Blocks", strconv.Itoa(d.Matched))
	h.Set("X-Router-Attempts", strconv.Itoa(attempt+1))
	w.WriteHeader(resp.StatusCode)

	rc := http.NewResponseController(w)
	bufp := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufp)
	buf := *bufp
	first := true
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if first {
				p.metrics.ObserveTTFB(time.Since(start))
				first = false
			}
			if _, werr := w.Write(buf[:n]); werr != nil {
				p.metrics.Inc(`kvrouter_client_cancelled_total`)
				return true, resp.StatusCode, werr
			}
			_ = rc.Flush() // SSE: push every chunk immediately
		}
		if rerr == io.EOF {
			return true, resp.StatusCode, nil
		}
		if rerr != nil {
			if r.Context().Err() == nil && !errors.Is(rerr, context.Canceled) {
				// Upstream died mid-stream. Can't retry (bytes already sent);
				// count it and let passive health eject the backend.
				p.metrics.Inc(`kvrouter_midstream_errors_total`)
				p.pool.ReportConnError(b, rerr)
			}
			return true, resp.StatusCode, rerr
		}
	}
}

// RunHealthChecks actively probes GET /health on every backend. Combined
// with passive checks in try(), a crashed instance is ejected on the first
// refused connection and restored on its first good probe.
func (p *Proxy) RunHealthChecks(ctx context.Context) {
	client := &http.Client{Timeout: p.cfg.HealthTimeout}
	t := time.NewTicker(p.cfg.HealthInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			var wg sync.WaitGroup
			for _, b := range p.pool.Backends {
				wg.Add(1)
				go func(b *Backend) {
					defer wg.Done()
					resp, err := client.Get(b.URL + "/health")
					if err != nil {
						p.pool.ReportFailure(b, "probe: "+err.Error())
						return
					}
					resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						p.pool.MarkUp(b)
					} else {
						p.pool.ReportFailure(b, "probe status "+strconv.Itoa(resp.StatusCode))
					}
				}(b)
			}
			wg.Wait()
		}
	}
}
