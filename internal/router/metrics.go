package router

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

// Metrics is a tiny Prometheus text-format registry. Stdlib only, so the
// repo builds anywhere with zero dependencies.
type Metrics struct {
	mu       sync.Mutex
	counters map[string]float64
	ttfb     *histogram
}

func NewMetrics() *Metrics {
	return &Metrics{
		counters: map[string]float64{},
		ttfb:     newHistogram([]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}),
	}
}

// Inc increments a counter identified by its full series name, e.g.
// `kvrouter_route_decisions_total{reason="prefix_hit"}`.
func (m *Metrics) Inc(series string) {
	m.mu.Lock()
	m.counters[series]++
	m.mu.Unlock()
}

func (m *Metrics) ObserveTTFB(d time.Duration) { m.ttfb.observe(d.Seconds()) }

func (m *Metrics) Handler(p *Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		m.mu.Lock()
		keys := make([]string, 0, len(m.counters))
		for k := range m.counters {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(w, "%s %g\n", k, m.counters[k])
		}
		m.mu.Unlock()

		for _, b := range p.Backends {
			l := fmt.Sprintf(`{backend=%q}`, b.URL)
			fmt.Fprintf(w, "kvrouter_backend_inflight%s %d\n", l, b.Inflight())
			fmt.Fprintf(w, "kvrouter_backend_healthy%s %d\n", l, boolInt(b.Healthy()))
			fmt.Fprintf(w, "kvrouter_backend_requests_total%s %d\n", l, b.served.Load())
			fmt.Fprintf(w, "kvrouter_backend_index_blocks%s %d\n", l, b.index.Len())
		}
		m.ttfb.write(w, "kvrouter_upstream_ttfb_seconds")
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

type histogram struct {
	mu      sync.Mutex
	bounds  []float64
	buckets []uint64
	sum     float64
	count   uint64
}

func newHistogram(bounds []float64) *histogram {
	return &histogram{bounds: bounds, buckets: make([]uint64, len(bounds))}
}

func (h *histogram) observe(v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, b := range h.bounds {
		if v <= b {
			h.buckets[i]++
		}
	}
	h.sum += v
	h.count++
}

func (h *histogram) write(w io.Writer, name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, b := range h.bounds {
		fmt.Fprintf(w, "%s_bucket{le=\"%g\"} %d\n", name, b, h.buckets[i])
	}
	fmt.Fprintf(w, "%s_bucket{le=\"+Inf\"} %d\n", name, h.count)
	fmt.Fprintf(w, "%s_sum %g\n%s_count %d\n", name, h.sum, name, h.count)
}
