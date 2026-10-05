package prefix

import (
	"strings"
	"testing"
)

func TestBlockHashesChainProperty(t *testing.T) {
	a := strings.Repeat("x", 1000) + "AAAA" + strings.Repeat("y", 500)
	b := strings.Repeat("x", 1000) + "BBBB" + strings.Repeat("y", 500)
	ha, hb := BlockHashes(a, 128), BlockHashes(b, 128)
	if len(ha) != 11 || len(hb) != 11 { // 1504/128 = 11 full blocks
		t.Fatalf("want 11 full blocks, got %d/%d", len(ha), len(hb))
	}
	// Strings diverge at byte 1000 -> inside block 7 (896..1023).
	for i := 0; i < 7; i++ {
		if ha[i] != hb[i] {
			t.Fatalf("block %d should match", i)
		}
	}
	// Chaining: every block from the divergence on differs, even though the
	// trailing "y" blocks are byte-identical.
	for i := 7; i < 11; i++ {
		if ha[i] == hb[i] {
			t.Fatalf("block %d should differ (chain must carry divergence)", i)
		}
	}
}

func TestExtractPromptChatIsPrefixAcrossTurns(t *testing.T) {
	t1 := ExtractPrompt([]byte(`{"messages":[{"role":"system","content":"S"},{"role":"user","content":"u1"}]}`))
	t2 := ExtractPrompt([]byte(`{"messages":[{"role":"system","content":"S"},{"role":"user","content":"u1"},{"role":"assistant","content":"a1"},{"role":"user","content":"u2"}]}`))
	if !strings.HasPrefix(t2, t1) {
		t.Fatalf("turn 2 prompt must extend turn 1:\n%q\n%q", t1, t2)
	}
	parts := ExtractPrompt([]byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"hi"},{"type":"image_url","image_url":{}}]}]}`))
	if !strings.Contains(parts, "hi") {
		t.Fatalf("content parts not flattened: %q", parts)
	}
	if got := ExtractPrompt([]byte(`{"prompt":"hello"}`)); got != "hello" {
		t.Fatalf("completions prompt: %q", got)
	}
}
