package router

import (
	"math"
	"math/rand/v2"
	"sync/atomic"

	"github.com/sparkling-snail/kvrouter/internal/prefix"
)

// Decision is a routing choice plus why it was made (exported as a metric
// label and a response header — the "why did my request go there?" question
// is the first thing you ask when debugging a router).
type Decision struct {
	Backend *Backend
	Reason  string
	Matched int // prefix blocks the router believes are cached there
}

type Policy interface {
	Name() string
	Pick(hashes []uint64, exclude map[int]bool) Decision
	// Observe is called once a backend accepted the request, so the router
	// can record that this prefix is now (about to be) cached there.
	Observe(b *Backend, hashes []uint64)
}

// ---------------------------------------------------------------- round robin

type RoundRobin struct {
	pool *Pool
	next atomic.Uint64
}

func NewRoundRobin(p *Pool) *RoundRobin          { return &RoundRobin{pool: p} }
func (r *RoundRobin) Name() string               { return "round_robin" }
func (r *RoundRobin) Observe(*Backend, []uint64) {}

func (r *RoundRobin) Pick(_ []uint64, exclude map[int]bool) Decision {
	cands, panicMode := r.pool.candidates(exclude)
	if len(cands) == 0 {
		return Decision{}
	}
	b := cands[int(r.next.Add(1)-1)%len(cands)]
	return Decision{Backend: b, Reason: withPanic("round_robin", panicMode)}
}

// -------------------------------------------------------------- least loaded

type LeastLoaded struct {
	pool *Pool
	rot  atomic.Uint64
}

func NewLeastLoaded(p *Pool) *LeastLoaded         { return &LeastLoaded{pool: p} }
func (l *LeastLoaded) Name() string               { return "least_loaded" }
func (l *LeastLoaded) Observe(*Backend, []uint64) {}

func (l *LeastLoaded) Pick(_ []uint64, exclude map[int]bool) Decision {
	cands, panicMode := l.pool.candidates(exclude)
	if len(cands) == 0 {
		return Decision{}
	}
	return Decision{Backend: leastLoaded(cands, &l.rot), Reason: withPanic("least_loaded", panicMode)}
}

func leastLoaded(cands []*Backend, rot *atomic.Uint64) *Backend {
	// Rotate the scan start so ties don't always go to backend 0.
	start := int(rot.Add(1)) % len(cands)
	best := cands[start]
	for i := 1; i < len(cands); i++ {
		c := cands[(start+i)%len(cands)]
		if c.Inflight() < best.Inflight() {
			best = c
		}
	}
	return best
}

// -------------------------------------------------------------------- random

// Random sends each request to a uniformly random healthy backend: the
// "one random choice" baseline from balls-into-bins.
type Random struct{ pool *Pool }

func NewRandom(p *Pool) *Random              { return &Random{pool: p} }
func (r *Random) Name() string               { return "random" }
func (r *Random) Observe(*Backend, []uint64) {}

func (r *Random) Pick(_ []uint64, exclude map[int]bool) Decision {
	cands, panicMode := r.pool.candidates(exclude)
	if len(cands) == 0 {
		return Decision{}
	}
	return Decision{Backend: cands[rand.IntN(len(cands))], Reason: withPanic("random", panicMode)}
}

// ---------------------------------------------------------- power of two

// PowerOfTwo samples two distinct healthy backends at random and picks the
// one with fewer in-flight requests (Mitzenmacher's "power of two choices",
// Envoy's LEAST_REQUEST default). With two backends it checks both, so it
// behaves like least_loaded; with more it only ever looks at two.
type PowerOfTwo struct{ pool *Pool }

func NewPowerOfTwo(p *Pool) *PowerOfTwo           { return &PowerOfTwo{pool: p} }
func (p2 *PowerOfTwo) Name() string               { return "p2c" }
func (p2 *PowerOfTwo) Observe(*Backend, []uint64) {}

func (p2 *PowerOfTwo) Pick(_ []uint64, exclude map[int]bool) Decision {
	cands, panicMode := p2.pool.candidates(exclude)
	if len(cands) == 0 {
		return Decision{}
	}
	if len(cands) == 1 {
		return Decision{Backend: cands[0], Reason: withPanic("p2c", panicMode)}
	}
	i := rand.IntN(len(cands))
	j := rand.IntN(len(cands) - 1)
	if j >= i {
		j++ // distinct second sample
	}
	a, b := cands[i], cands[j]
	if b.Inflight() < a.Inflight() {
		a = b
	}
	return Decision{Backend: a, Reason: withPanic("p2c", panicMode)}
}

// -------------------------------------------------------------- prefix aware

// PrefixAware routes to the backend holding the longest cached prefix,
// subject to a bounded-load constraint (consistent hashing with bounded
// loads, Mirrokni et al. 2018): no backend may exceed
//
//	ceil(LoadFactor * avg_inflight) + Slack
//
// When the cache-best backend is over that bound the request spills to the
// next-best backend under it. That deliberately *replicates* hot prefixes
// onto more instances instead of melting one of them.
//
// Cold prefixes (no backend has them) go to a rendezvous-hash owner of the
// first AffinityBlocks blocks, so the first request for a new system prompt
// is placed deterministically and its followers find it.
type PrefixAware struct {
	pool           *Pool
	LoadFactor     float64 // <= 0 disables the bound (pure affinity)
	Slack          int64
	AffinityBlocks int
	rot            atomic.Uint64
}

func NewPrefixAware(p *Pool, loadFactor float64, slack int64, affinityBlocks int) *PrefixAware {
	if affinityBlocks <= 0 {
		affinityBlocks = 8
	}
	return &PrefixAware{pool: p, LoadFactor: loadFactor, Slack: slack, AffinityBlocks: affinityBlocks}
}

func (pa *PrefixAware) Name() string { return "prefix" }

func (pa *PrefixAware) Observe(b *Backend, hashes []uint64) { b.index.Insert(hashes) }

func (pa *PrefixAware) bound(cands []*Backend) int64 {
	if pa.LoadFactor <= 0 {
		return math.MaxInt64
	}
	var total int64
	for _, c := range cands {
		total += c.Inflight()
	}
	avg := float64(total+1) / float64(len(cands)) // +1: the request being placed
	return int64(math.Ceil(avg*pa.LoadFactor)) + pa.Slack
}

func (pa *PrefixAware) Pick(hashes []uint64, exclude map[int]bool) Decision {
	cands, panicMode := pa.pool.candidates(exclude)
	if len(cands) == 0 {
		return Decision{}
	}
	bound := pa.bound(cands)
	under := func(b *Backend) bool { return b.Inflight() < bound }

	// 1. Longest cached prefix, overall and among backends under the bound.
	var bestAll, bestOK *Backend
	bestAllLen, bestOKLen := 0, 0
	better := func(m int, c *Backend, curLen int, cur *Backend) bool {
		return m > curLen || (m == curLen && m > 0 && cur != nil && c.Inflight() < cur.Inflight())
	}
	if len(hashes) > 0 {
		for _, c := range cands {
			m := c.index.MatchLen(hashes)
			if better(m, c, bestAllLen, bestAll) {
				bestAll, bestAllLen = c, m
			}
			if under(c) && better(m, c, bestOKLen, bestOK) {
				bestOK, bestOKLen = c, m
			}
		}
	}
	// A match shorter than the affinity window is probably just the shared
	// chat-template / global system header that *every* backend will end up
	// holding. Treating that as a hit would pile all new tenants onto
	// whichever backend saw the header first, so short matches don't count:
	// placement falls through to rendezvous hashing on the longer key.
	minMatch := min(pa.AffinityBlocks, len(hashes))
	if bestOK != nil && bestOKLen > 0 && bestOKLen >= minMatch {
		reason := "prefix_hit"
		if bestAllLen > bestOKLen {
			reason = "prefix_spill" // cache-best was overloaded; took a shorter match
		}
		return Decision{Backend: bestOK, Reason: withPanic(reason, panicMode), Matched: bestOKLen}
	}

	// 2. No usable cached prefix.
	if len(hashes) == 0 {
		var ok []*Backend
		for _, c := range cands {
			if under(c) {
				ok = append(ok, c)
			}
		}
		if len(ok) == 0 {
			ok = cands
		}
		return Decision{Backend: leastLoaded(ok, &pa.rot), Reason: withPanic("short_prompt", panicMode)}
	}

	// 3. Rendezvous (highest-random-weight) hashing on the affinity key.
	k := pa.AffinityBlocks
	if k > len(hashes) {
		k = len(hashes)
	}
	key := hashes[k-1] // chained, so it commits to the whole first k blocks
	var top, pick *Backend
	var topScore, pickScore uint64
	for _, c := range cands {
		s := prefix.Mix(key ^ (uint64(c.ID+1) * 0x9e3779b97f4a7c15))
		if top == nil || s > topScore {
			top, topScore = c, s
		}
		if under(c) && (pick == nil || s > pickScore) {
			pick, pickScore = c, s
		}
	}
	if pick == nil {
		pick = leastLoaded(cands, &pa.rot)
	}
	reason := "hash_affinity"
	if pick != top || bestAllLen >= minMatch {
		reason = "load_fallback"
	}
	return Decision{Backend: pick, Reason: withPanic(reason, panicMode)}
}

// ------------------------------------------------------------ weighted score

// Weighted is the scoring approach used by the Kubernetes Gateway API
// Inference Extension and llm-d: every backend gets
//
//	PrefixWeight * (matched blocks / prompt blocks)
//	+ LoadWeight * (max_inflight - inflight) / (max_inflight - min_inflight)
//
// with the load term min-max normalized across candidates (1 if all equal),
// and the highest score wins. Unlike PrefixAware there is no hard load cap,
// no minimum match length and no rendezvous placement for cold prompts: the
// cache/load trade-off lives entirely in the two weights. It is here to be
// benchmarked against the bounded-load policy.
type Weighted struct {
	pool                     *Pool
	PrefixWeight, LoadWeight float64
	AffinityBlocks           int // only used to label decisions, not to score
	rot                      atomic.Uint64
}

func NewWeighted(p *Pool, prefixWeight, loadWeight float64, affinityBlocks int) *Weighted {
	if affinityBlocks <= 0 {
		affinityBlocks = 8
	}
	return &Weighted{pool: p, PrefixWeight: prefixWeight, LoadWeight: loadWeight, AffinityBlocks: affinityBlocks}
}

func (w *Weighted) Name() string { return "weighted" }

func (w *Weighted) Observe(b *Backend, hashes []uint64) { b.index.Insert(hashes) }

func (w *Weighted) Pick(hashes []uint64, exclude map[int]bool) Decision {
	cands, panicMode := w.pool.candidates(exclude)
	if len(cands) == 0 {
		return Decision{}
	}
	lo, hi := cands[0].Inflight(), cands[0].Inflight()
	for _, c := range cands[1:] {
		lo, hi = min(lo, c.Inflight()), max(hi, c.Inflight())
	}
	// Rotate the scan start so ties don't always go to backend 0.
	start := int(w.rot.Add(1)) % len(cands)
	var best *Backend
	bestScore, bestLen := math.Inf(-1), 0
	for i := range cands {
		c := cands[(start+i)%len(cands)]
		m, prefixScore := 0, 0.0
		if len(hashes) > 0 {
			m = c.index.MatchLen(hashes)
			prefixScore = float64(m) / float64(len(hashes))
		}
		loadScore := 1.0
		if hi > lo {
			loadScore = float64(hi-c.Inflight()) / float64(hi-lo)
		}
		if s := w.PrefixWeight*prefixScore + w.LoadWeight*loadScore; s > bestScore {
			best, bestScore, bestLen = c, s, m
		}
	}
	reason := "weighted_cold"
	if bestLen > 0 && bestLen >= min(w.AffinityBlocks, len(hashes)) {
		reason = "weighted_hit"
	}
	return Decision{Backend: best, Reason: withPanic(reason, panicMode), Matched: bestLen}
}

func withPanic(reason string, panicMode bool) string {
	if panicMode {
		return reason + "_panic"
	}
	return reason
}
