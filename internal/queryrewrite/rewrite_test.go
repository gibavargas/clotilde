package queryrewrite

import (
	"strings"
	"testing"
)

// The secondary literal query must appear when the cleaned question is much
// longer than the primary (len(primary) < len(cleaned)-3), and be absent
// otherwise.
func TestRewriteSecondaryQueryPresent(t *testing.T) {
	got := Rewrite("acabei de ver a placa 04 de senador, de qual senador ela é?", now)
	if len(got) < 2 {
		t.Fatalf("expected secondary literal query, got %v", got)
	}
	// Primary: compact keywords; secondary: literal leftover wording.
	if !strings.Contains(got[0], "senado federal") {
		t.Errorf("primary missing domain hint: %q", got[0])
	}
	secondary := got[1]
	if len(secondary) <= len(got[0]) {
		t.Errorf("secondary should be the longer literal form; primary=%q secondary=%q", got[0], secondary)
	}
}

func TestRewriteSecondaryQueryAbsent(t *testing.T) {
	// Short input: cleaned ≈ primary, so no secondary.
	got := Rewrite("dólar hoje", now)
	if len(got) != 1 {
		t.Fatalf("expected single query for short input, got %v", got)
	}
}

func TestRewriteSecondaryTruncated(t *testing.T) {
	// Distinct tail so the literal secondary differs from the keyword primary.
	// Cleaned keeps stopwords/interrogatives ("qual", "é") that salient drops,
	// so even after the 12-token cap the literal secondary differs from the
	// keyword primary.
	long := "qual é a cotação dólar " + strings.Repeat("palavra ", 30)
	got := Rewrite(long, now)
	if len(got) != 2 {
		t.Fatalf("expected 2 queries for long input, got %d", len(got))
	}
	// truncateTokens caps both at 12 tokens.
	for i, q := range got {
		if n := len(strings.Fields(q)); n > 12 {
			t.Errorf("query %d exceeded 12 tokens: %d (%q)", i, n, q)
		}
	}
}

func TestRewriteTemporalVariants(t *testing.T) {
	cases := []struct{ word string }{
		{"hoje"}, {"ontem"}, {"recente"}, {"atual"},
	}
	for _, tc := range cases {
		got := Rewrite("a votação do senado "+tc.word, now)
		if !strings.Contains(got[0], "2026") {
			t.Errorf("temporal word %q did not gain the year: %q", tc.word, got[0])
		}
	}
}

func TestRewriteDollarHint(t *testing.T) {
	got := Rewrite("Qual a cotação do dólar hoje?", now)
	if !strings.Contains(got[0], "cotação dólar") {
		t.Errorf("domain hint missing: %q", got[0])
	}
	if !strings.Contains(got[0], "2026") {
		t.Errorf("recency year missing: %q", got[0])
	}
}

func TestRewriteSalientEmptyFallback(t *testing.T) {
	// A question made only of stopwords/interrogatives must still produce a
	// non-empty query (salient fallback = remaining words).
	got := Rewrite("quem? qual? exclusivamente", now)
	if len(got) == 0 || got[0] == "" {
		t.Fatalf("salient-empty fallback produced empty query: %v", got)
	}
}
