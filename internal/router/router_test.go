package router

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sparkling-snail/kvrouter/internal/prefix"
)

func urls(n int) []string {
	var u []string
	for i := 0; i < n; i++ {
		u = append(u, fmt.Sprintf("http://b%d", i))
	}
	return u
}

func chain(seed string, blocks int) []uint64 {
	return prefix.BlockHashes(strings.Repeat(seed, blocks*128/len(seed)+1), 128)[:blocks]
}

func TestPrefixAffinityIsSticky(t *testing.T) {
	pool := NewPool(urls(4), 1000)
	pa := NewPrefixAware(pool, 1.25, 2, 8)
	h := chain("tenant-A system prompt ", 20)

	d := pa.Pick(h, nil)
	if d.Reason != "hash_affinity" {
		t.Fatalf("cold prefix should use rendezvous, got %s", d.Reason)
	}
	pa.Observe(d.Backend, h)
	// Next turn of the same conversation: longer prompt, same prefix.
	h2 := append(append([]uint64{}, h...), chain("turn two", 5)...)
	d2 := pa.Pick(h2, nil)
	if d2.Backend != d.Backend || d2.Reason != "prefix_hit" || d2.Matched != 20 {
		t.Fatalf("want prefix_hit on %s with 20 blocks, got %s on %s (%d)", d.Backend.URL, d2.Reason, d2.Backend.URL, d2.Matched)
	}
}

func TestBoundedLoadSpills(t *testing.T) {
	pool := NewPool(urls(4), 1000)
	pa := NewPrefixAware(pool, 1.25, 0, 8)
	h := chain("hot tenant ", 20)
	owner := pa.Pick(h, nil).Backend
	pa.Observe(owner, h)

	owner.inflight.Store(10) // owner is hot, the others idle
	d := pa.Pick(h, nil)
	if d.Backend == owner {
		t.Fatalf("overloaded owner should not get the request")
	}
	if d.Reason != "load_fallback" {
		t.Fatalf("want load_fallback, got %s", d.Reason)
	}
	// Once the spill target has the prefix too, it is a (shorter or equal) hit.
	pa.Observe(d.Backend, h[:10])
	d2 := pa.Pick(h, nil)
	if d2.Reason != "prefix_spill" || d2.Backend != d.Backend {
		t.Fatalf("want prefix_spill to replica, got %s on %s", d2.Reason, d2.Backend.URL)
	}

	// With the bound disabled, pure affinity hammers the owner regardless.
	pure := NewPrefixAware(pool, 0, 0, 8)
	if got := pure.Pick(h, nil).Backend; got != owner {
		t.Fatalf("pure affinity should pick owner")
	}
}

func TestWeightedTradesCacheForLoad(t *testing.T) {
	pool := NewPool(urls(4), 1000)
	h := chain("tenant ", 20)
	owner := pool.Backends[2]
	owner.index.Insert(h)

	// Equal load: the cached backend wins.
	w := NewWeighted(pool, 3, 2, 8)
	if d := w.Pick(h, nil); d.Backend != owner || d.Reason != "weighted_hit" || d.Matched != 20 {
		t.Fatalf("want weighted_hit on owner, got %s on %s (%d)", d.Reason, d.Backend.URL, d.Matched)
	}
	// Owner is the busiest. With prefix weight > load weight a full match
	// still outscores an idle cold backend (3*1+2*0 > 3*0+2*1)...
	owner.inflight.Store(10)
	if d := w.Pick(h, nil); d.Backend != owner {
		t.Fatalf("prefix-heavy weights should stay on owner, got %s", d.Backend.URL)
	}
	// ...and with load weight > prefix weight it moves.
	w = NewWeighted(pool, 1, 2, 8)
	if d := w.Pick(h, nil); d.Backend == owner || d.Reason != "weighted_cold" {
		t.Fatalf("load-heavy weights should leave busy owner, got %s on %s", d.Reason, d.Backend.URL)
	}
}

// Every prompt starts with the same chat-template header. Matching only that
// header must not drag new tenants onto the backend that happens to hold it.
func TestSharedHeaderDoesNotCountAsHit(t *testing.T) {
	pool := NewPool(urls(4), 10000)
	pa := NewPrefixAware(pool, 0, 0, 8)
	header := strings.Repeat("<|system|> global safety policy. ", 12) // ~3 blocks
	counts := map[int]int{}
	for i := 0; i < 200; i++ {
		h := prefix.BlockHashes(header+strings.Repeat(fmt.Sprintf("tenant %d docs. ", i), 100), 128)
		d := pa.Pick(h, nil)
		if d.Reason == "prefix_hit" {
			t.Fatalf("tenant %d: header-only match treated as hit (matched %d)", i, d.Matched)
		}
		pa.Observe(d.Backend, h)
		counts[d.Backend.ID]++
	}
	for id, n := range counts {
		if n > 80 {
			t.Fatalf("new tenants piled onto backend %d: %v", id, counts)
		}
	}
}

func TestUnhealthyAndPanicMode(t *testing.T) {
	pool := NewPool(urls(3), 100)
	pa := NewPrefixAware(pool, 1.25, 2, 8)
	h := chain("x", 10)
	owner := pa.Pick(h, nil).Backend
	pa.Observe(owner, h)

	pool.MarkDown(owner, "test")
	if owner.index.Len() != 0 {
		t.Fatalf("ejecting a backend must clear its index")
	}
	if d := pa.Pick(h, nil); d.Backend == owner {
		t.Fatalf("picked unhealthy backend")
	}
	for _, b := range pool.Backends {
		pool.MarkDown(b, "test")
	}
	if d := pa.Pick(h, nil); d.Backend == nil || !strings.HasSuffix(d.Reason, "_panic") {
		t.Fatalf("all-down should panic-route, got %+v", d)
	}
}

// A backend whose /health is fine but whose requests keep failing must still
// be ejected: good probes between failures must not reset the streak.
func TestHealthyProbeDoesNotHideRequestFailures(t *testing.T) {
	pool := NewPool(urls(2), 100)
	b := pool.Backends[0]
	for i := int64(0); i < pool.FailThreshold; i++ {
		pool.MarkUp(b) // probe succeeds between every failed request
		pool.ReportFailure(b, "http 503")
	}
	if b.Healthy() {
		t.Fatalf("backend failing every request should be ejected despite passing probes")
	}
	pool.MarkUp(b)
	if !b.Healthy() || b.consecFails.Load() != 0 {
		t.Fatalf("restore should mark healthy and clear the streak")
	}
}

func TestRendezvousSpreadsTenants(t *testing.T) {
	pool := NewPool(urls(4), 100)
	pa := NewPrefixAware(pool, 0, 0, 8)
	counts := map[int]int{}
	for i := 0; i < 400; i++ {
		counts[pa.Pick(chain(fmt.Sprintf("tenant %d ", i), 8), nil).Backend.ID]++
	}
	for id, n := range counts {
		if n < 60 || n > 140 {
			t.Fatalf("rendezvous skewed: backend %d got %d/400 (%v)", id, n, counts)
		}
	}
}

// ---- proxy-level behaviour with real HTTP servers ----

func newTestProxy(t *testing.T, backends ...string) *Proxy {
	pool := NewPool(backends, 1000)
	return NewProxy(Config{BlockChars: 128, MaxRetries: 2, MaxBodyBytes: 1 << 20,
		ResponseHeaderTimeout: 5 * time.Second, HealthInterval: time.Hour, HealthTimeout: time.Second},
		pool, NewRoundRobin(pool))
}

func TestPowerOfTwoAvoidsBusiestBackend(t *testing.T) {
	pool := NewPool(urls(4), 1000)
	for i, b := range pool.Backends {
		b.inflight.Store(int64(i)) // b3 is the busiest
	}
	p2 := NewPowerOfTwo(pool)
	seen := map[int]int{}
	for i := 0; i < 2000; i++ {
		seen[p2.Pick(nil, nil).Backend.ID]++
	}
	if seen[3] != 0 {
		t.Fatalf("busiest backend can never win a two-way comparison, got %d picks", seen[3])
	}
	// b0 wins every pair it is in: 3 of the 6 distinct pairs.
	if seen[0] < 800 || seen[0] > 1200 {
		t.Fatalf("least loaded backend should win ~half the picks, got %d/2000", seen[0])
	}
	// Two backends: two choices = least loaded.
	two := NewPool(urls(2), 1000)
	two.Backends[0].inflight.Store(5)
	if got := NewPowerOfTwo(two).Pick(nil, nil).Backend; got != two.Backends[1] {
		t.Fatalf("with 2 backends p2c should always pick the idler one")
	}
}

func TestRandomSpreadsUniformly(t *testing.T) {
	pool := NewPool(urls(4), 1000)
	pool.Backends[0].inflight.Store(100) // load is ignored
	r := NewRandom(pool)
	seen := map[int]int{}
	for i := 0; i < 4000; i++ {
		seen[r.Pick(nil, nil).Backend.ID]++
	}
	for id := 0; id < 4; id++ {
		if seen[id] < 800 || seen[id] > 1200 {
			t.Fatalf("backend %d got %d/4000 picks, want ~1000", id, seen[id])
		}
	}
}

func TestRetryOn503ThenSucceeds(t *testing.T) {
	var badHits atomic.Int64
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		badHits.Add(1)
		w.WriteHeader(503)
	}))
	defer bad.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "data: ok\n\n")
	}))
	defer good.Close()

	p := newTestProxy(t, bad.URL, good.URL)
	front := httptest.NewServer(p.Handler())
	defer front.Close()

	for i := 0; i < 4; i++ {
		resp, err := http.Post(front.URL+"/v1/completions", "application/json", strings.NewReader(`{"prompt":"hi"}`))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 || string(b) != "data: ok\n\n" {
			t.Fatalf("req %d: status %d body %q", i, resp.StatusCode, b)
		}
	}
	// 3 consecutive 503s eject the bad backend; after that it gets no traffic.
	if p.pool.Backends[0].Healthy() {
		t.Fatalf("bad backend should be ejected after %d failures", p.pool.FailThreshold)
	}
	if n := badHits.Load(); n > 3 {
		t.Fatalf("bad backend hit %d times after ejection threshold", n)
	}
}

func TestConnRefusedEjectsImmediately(t *testing.T) {
	dead := httptest.NewServer(http.NotFoundHandler())
	deadURL := dead.URL
	dead.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, "ok") }))
	defer good.Close()

	p := newTestProxy(t, deadURL, good.URL)
	front := httptest.NewServer(p.Handler())
	defer front.Close()
	resp, err := http.Post(front.URL+"/v1/completions", "application/json", strings.NewReader(`{"prompt":"x"}`))
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("expected transparent failover, got %v %v", resp, err)
	}
	if p.pool.Backends[0].Healthy() {
		t.Fatalf("connection refused should eject on first failure")
	}
}

func TestNoRetryAfterFirstByte(t *testing.T) {
	var hits atomic.Int64
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		io.WriteString(w, "data: partial\n\n")
		w.(http.Flusher).Flush()
		// Die mid-stream.
		hj, _ := w.(http.Hijacker)
		c, _, _ := hj.Hijack()
		c.Close()
	})
	a, b := httptest.NewServer(h), httptest.NewServer(h)
	defer a.Close()
	defer b.Close()

	p := newTestProxy(t, a.URL, b.URL)
	front := httptest.NewServer(p.Handler())
	defer front.Close()
	resp, err := http.Post(front.URL+"/v1/completions", "application/json", strings.NewReader(`{"prompt":"x","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	io.ReadAll(resp.Body)
	resp.Body.Close()
	if hits.Load() != 1 {
		t.Fatalf("mid-stream failure must not be retried (would duplicate tokens); backends hit %d times", hits.Load())
	}
}
