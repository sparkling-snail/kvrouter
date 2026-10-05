// Package prefix turns an inference request into a chain of block hashes.
//
// vLLM's automatic prefix caching splits the token sequence into fixed-size
// blocks and keys each block by hash(parent_hash, block_tokens). Two requests
// can reuse block i only if they share every token up to the end of block i.
// We reproduce the same *chained* structure on characters instead of tokens:
// the router does not need to tokenize, it only needs "same prefix -> same
// hash chain" so it can send the request where that prefix already lives.
package prefix

import (
	"encoding/json"
	"strings"
)

const (
	fnvOffset = 14695981039346656037
	fnvPrime  = 1099511628211
)

// BlockHashes returns one chained hash per *full* block of blockSize bytes.
// h[i] commits to text[0 : (i+1)*blockSize]. A trailing partial block is
// ignored, matching vLLM, which only caches full blocks.
func BlockHashes(text string, blockSize int) []uint64 {
	if blockSize <= 0 {
		blockSize = 128
	}
	n := len(text) / blockSize
	out := make([]uint64, n)
	prev := uint64(fnvOffset)
	for i := 0; i < n; i++ {
		h := prev
		blk := text[i*blockSize : (i+1)*blockSize]
		for j := 0; j < len(blk); j++ {
			h ^= uint64(blk[j])
			h *= fnvPrime
		}
		h = Mix(h)
		out[i] = h
		prev = h
	}
	return out
}

// Mix is the splitmix64 finalizer; used for avalanche and rendezvous scores.
func Mix(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	return x
}

type message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type request struct {
	Prompt   json.RawMessage `json:"prompt"`
	Messages []message       `json:"messages"`
}

// ExtractPrompt pulls the cacheable prompt text out of an OpenAI-compatible
// /v1/completions or /v1/chat/completions body. Chat messages are flattened
// in order, so a multi-turn conversation's earlier turns form a prefix of
// its later turns, exactly as they do after the chat template is applied.
func ExtractPrompt(body []byte) string {
	var r request
	if err := json.Unmarshal(body, &r); err != nil {
		return ""
	}
	if len(r.Messages) > 0 {
		var b strings.Builder
		for _, m := range r.Messages {
			b.WriteString("<|")
			b.WriteString(m.Role)
			b.WriteString("|>\n")
			b.WriteString(contentText(m.Content))
			b.WriteString("\n")
		}
		return b.String()
	}
	if len(r.Prompt) > 0 {
		var s string
		if json.Unmarshal(r.Prompt, &s) == nil {
			return s
		}
		var arr []string
		if json.Unmarshal(r.Prompt, &arr) == nil && len(arr) > 0 {
			return arr[0]
		}
	}
	return ""
}

func contentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	// Multimodal-style content: [{"type":"text","text":"..."}]
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		var b strings.Builder
		for _, p := range parts {
			if p.Type == "text" {
				b.WriteString(p.Text)
			}
		}
		return b.String()
	}
	return ""
}
