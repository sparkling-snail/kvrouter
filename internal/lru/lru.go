// Package lru is a bounded set of block hashes with LRU eviction.
//
// The router uses one per backend as an *approximate* mirror of that
// backend's prefix cache; the mock vLLM server uses one as its actual cache.
package lru

import (
	"container/list"
	"sync"
)

type Set struct {
	mu  sync.Mutex
	cap int
	ll  *list.List // front = most recently used
	m   map[uint64]*list.Element
}

func New(capacity int) *Set {
	if capacity <= 0 {
		capacity = 1
	}
	return &Set{cap: capacity, ll: list.New(), m: make(map[uint64]*list.Element, capacity)}
}

// MatchLen returns how many leading hashes are present (longest cached prefix).
func (s *Set) MatchLen(hashes []uint64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.matchLocked(hashes)
}

func (s *Set) matchLocked(hashes []uint64) int {
	for i, h := range hashes {
		if _, ok := s.m[h]; !ok {
			return i
		}
	}
	return len(hashes)
}

// Insert adds/refreshes all hashes. They are touched deepest-first so the
// root of a chain ends up most-recent; eviction then removes the leaves of
// stale chains before their roots, like vLLM's leaf-first eviction. Evicting
// a root first would orphan every block below it.
func (s *Set) Insert(hashes []uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.insertLocked(hashes)
}

func (s *Set) insertLocked(hashes []uint64) {
	for i := len(hashes) - 1; i >= 0; i-- {
		h := hashes[i]
		if e, ok := s.m[h]; ok {
			s.ll.MoveToFront(e)
			continue
		}
		s.m[h] = s.ll.PushFront(h)
	}
	for s.ll.Len() > s.cap {
		e := s.ll.Back()
		s.ll.Remove(e)
		delete(s.m, e.Value.(uint64))
	}
}

// MatchAndInsert atomically measures the cached prefix and then caches the
// whole chain — what a serving engine does when it schedules a request.
func (s *Set) MatchAndInsert(hashes []uint64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := s.matchLocked(hashes)
	s.insertLocked(hashes)
	return n
}

func (s *Set) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ll.Init()
	s.m = make(map[uint64]*list.Element, s.cap)
}

func (s *Set) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ll.Len()
}
