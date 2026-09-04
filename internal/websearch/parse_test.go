package websearch

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestParseBingHits_DecodesRedirect(t *testing.T) {
	// u=a1 encodes the target as base64url of the full URL.
	// "https://www12.senado.leg.br/noticias" pre-encoded:
	encoded := "aHR0cHM6Ly93d3cxMi5zZW5hZG8ubGVnLmJyL25vdGljaWFz"
	// Production bing.com SERPs use scheme-relative ck/a hrefs; with an
	// absolute https:// scheme normalizeDDGURL returns the bing.com URL and
	// the hit is dropped — decode only happens on the scheme-relative form.
	body := `<h2><a href="//www.bing.com/ck/a?!&amp;u=a1` + encoded + `&ntb=1"><strong>Senado</strong> notícias</a></h2>`

	hits := parseSERPLinks(body, 10)
	if len(hits) == 0 {
		t.Fatalf("expected 1 hit, got 0 for body %q", body)
	}
	if hits[0].url != "https://www12.senado.leg.br/noticias" {
		t.Errorf("expected decoded target, got %q", hits[0].url)
	}
	if !strings.Contains(hits[0].title, "Senado") {
		t.Errorf("expected title to keep text, got %q", hits[0].title)
	}
}

func TestParseSERPLinks_DDGANchors(t *testing.T) {
	body := `<a rel="nofollow" class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fwww12.senado.leg.br%2Fnoticias&amp;rut=abc">Senado Notícias</a>` +
		`<a class="result__snippet" href="#">Plenário aprova hoje</a>`
	hits := parseSERPLinks(body, 10)
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if hits[0].url != "https://www12.senado.leg.br/noticias" {
		t.Errorf("expected unwrapped DDG target, got %q", hits[0].url)
	}
	if hits[0].snippet != "Plenário aprova hoje" {
		t.Errorf("expected paired snippet, got %q", hits[0].snippet)
	}
}

func TestParseSERPLinks_GenericFallback(t *testing.T) {
	body := `<a href="https://g1.globo.com/politica/">Política hoje no país</a>` +
		`<a href="https://www.google.com/x">ab</a>` + // 2-rune title, dropped
		`<a href="javascript:void(0)">javascript no-op</a>` // non-http, dropped
	hits := parseSERPLinks(body, 10)
	if len(hits) != 1 {
		t.Fatalf("expected 1 generic hit, got %d", len(hits))
	}
	if hits[0].url != "https://g1.globo.com/politica/" {
		t.Errorf("unexpected hit: %+v", hits[0])
	}
}

func TestParseSERPLinks_Dedupes(t *testing.T) {
	body := `<a href="https://ex.com/a">Primeira matéria do dia</a><a href="https://ex.com/a">Segunda igual</a>`
	hits := parseSERPLinks(body, 10)
	if len(hits) != 1 {
		t.Fatalf("expected dedupe to 1 hit, got %d", len(hits))
	}
}

func TestDecodeToUTF8_CharsetBranches(t *testing.T) {
	latin1 := []byte{'S', 'e', 'n', 'a', 'd', 'o', 0xE7} // "Senadoç" in ISO-8859-1
	got := decodeToUTF8(latin1, "text/html; charset=iso-8859-1")
	if !strings.Contains(got, "Senadoç") {
		t.Errorf("latin-1 decode failed: %q", got)
	}

	utf16be := append([]byte{0xFE, 0xFF}, 0, 'A', 0, 'B') // BOM + "AB"
	got16 := decodeToUTF8(utf16be, "text/html; charset=utf-16")
	if got16 != "AB" {
		t.Errorf("utf-16 decode failed: %q", got16)
	}

	if decodeToUTF8([]byte("plain utf8 text"), "") != "plain utf8 text" {
		t.Error("valid utf-8 must pass through unchanged")
	}
}

func TestLooksLikeText(t *testing.T) {
	if looksLikeText("") {
		t.Error("empty body is not text")
	}
	if looksLikeText(strings.Repeat("\x00", 100)) {
		t.Error("NUL-heavy body must be rejected")
	}
	if !looksLikeText("Texto normal de página em português com acentuação.") {
		t.Error("normal text must pass")
	}
}

// Search-level error paths verified without network: empty query fails fast.
func TestSearchEmptyQuery(t *testing.T) {
	if _, err := Search(context.Background(), "", 6, time.Second); err == nil {
		t.Error("empty query must error")
	}
}
