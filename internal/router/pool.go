package router

import (
	"log/slog"
	"strings"
	"sync/atomic"

	"github.com/sparkling-snail/kvrouter/internal/lru"
)

// Backend is one vLLM instance.
type Backend struct {
	ID  int
	URL string

	inflight    atomic.Int64 // requests currently proxied (incl. streaming)
	healthy     atomic.Bool
	consecFails atomic.Int64
	served      atomic.Int64

	// index approximates which prefix blocks this backend has cached.
	index *lru.Set
}

func (b *Backend) Inflight() int64 { return b.inflight.Load() }
func (b *Backend) Healthy() bool   { return b.healthy.Load() }

// Pool owns backend state and the health state machine.
type Pool struct {
	Backends      []*Backend
	FailThreshold int64 // consecutive soft failures before ejection
}

func NewPool(urls []string, indexBlocks int) *Pool {
	p := &Pool{FailThreshold: 3}
	for i, u := range urls {
		b := &Backend{ID: i, URL: strings.TrimRight(u, "/"), index: lru.New(indexBlocks)}
		b.healthy.Store(true)
		p.Backends = append(p.Backends, b)
	}
	return p
}

// candidates returns healthy, non-excluded backends. If none are healthy it
// "panic routes" to every non-excluded backend (as Envoy does) rather than
// failing everything because health checks might be the broken part.
func (p *Pool) candidates(exclude map[int]bool) (out []*Backend, panicMode bool) {
	for _, b := range p.Backends {
		if !exclude[b.ID] && b.Healthy() {
			out = append(out, b)
		}
	}
	if len(out) > 0 {
		return out, false
	}
	for _, b := range p.Backends {
		if !exclude[b.ID] {
			out = append(out, b)
		}
	}
	return out, len(out) > 0
}

func (p *Pool) MarkDown(b *Backend, why string) {
	if b.healthy.Swap(false) {
		// A backend that went away most likely lost its KV cache (restart,
		// OOM, node drain). Forget what we thought it had so we don't keep
		// steering "hits" to a cold instance when it returns.
		b.index.Reset()
		slog.Warn("backend ejected", "backend", b.URL, "reason", why)
	}
}

// MarkUp is called on a good health probe. It only clears the failure streak
// when restoring an ejected backend: a backend that answers /health but fails
// real requests must not have its streak wiped every probe interval.
func (p *Pool) MarkUp(b *Backend) {
	if !b.healthy.Swap(true) {
		b.consecFails.Store(0)
		slog.Info("backend restored", "backend", b.URL)
	}
}

// ReportConnError: TCP-level failure. Eject immediately (passive health check).
func (p *Pool) ReportConnError(b *Backend, err error) { p.MarkDown(b, "conn error: "+err.Error()) }

// ReportFailure: HTTP 5xx or failed probe. Eject after N in a row.
func (p *Pool) ReportFailure(b *Backend, why string) {
	if b.consecFails.Add(1) >= p.FailThreshold {
		p.MarkDown(b, why)
	}
}

func (p *Pool) ReportSuccess(b *Backend) { b.consecFails.Store(0) }
