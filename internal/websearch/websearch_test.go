package websearch

import (
	"strings"
	"testing"
)

func TestNormalizeDDGURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"//duckduckgo.com/l/?uddg=https%3A%2F%2Fwww12.senado.leg.br%2Fnoticias&rut=abc", "https://www12.senado.leg.br/noticias"},
		{"https://example.com/page", "https://example.com/page"},
		{"javascript:void(0)", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := normalizeDDGURL(c.in); got != c.want {
			t.Errorf("normalizeDDGURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestStripTags(t *testing.T) {
	got := stripTags("<b>Senado</b> &amp; C&#226;mara")
	if got != "Senado & Câmara" {
		t.Errorf("stripTags = %q", got)
	}
}

func TestExtractHTMLText(t *testing.T) {
	body := `<html><head><style>.x{color:red}</style></head><body>
		<script>var a={};</script>
		<p>` + strings.Repeat("O Senado Federal aprovou hoje a indicação da nova placa protocolar em sessão deliberativa. ", 3) + `</p>
		</body></html>`
	got := extractHTMLText(body)
	if !strings.Contains(got, "Senado Federal") {
		t.Errorf("extractHTMLText lost body text: %q", got)
	}
	if strings.Contains(got, "var a") || strings.Contains(got, "color:red") {
		t.Errorf("extractHTMLText kept script/style: %q", got)
	}
}

func TestExtractJSONStrings(t *testing.T) {
	body := `{"intro":"Esta é a página oficial do senador fulano de tal no Senado Federal.","other":"x","url":"https://x.gov.br"}`
	got := extractJSONStrings(body, 800)
	if !strings.Contains(got, "senador fulano") {
		t.Errorf("extractJSONStrings missed long string: %q", got)
	}
	if strings.Contains(got, "https://x.gov.br") {
		t.Errorf("extractJSONStrings kept URL-only string: %q", got)
	}
}
