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

func TestIsInternalTarget(t *testing.T) {
	internal := []string{
		"http://localhost/x",
		"http://LOCALHOST/x",
		"http://localhost./x",
		"http://sub.localhost/x",
		"http://127.0.0.1/x",
		"http://10.0.0.1/x",
		"http://192.168.1.1/x",
		"http://172.16.0.1/x",
		"http://100.64.0.1/x",
		"http://169.254.169.254/metadata", // cloud metadata
		"http://[::1]/x",
		"http://[::ffff:127.0.0.1]/x",
		"http://[::ffff:169.254.169.254]/latest/meta-data/",
		"http://service.internal/x",
		"http://METADATA.GOOGLE.INTERNAL./computeMetadata/v1/",
		"http://printer.local/x",
		"file:///etc/passwd",
		"not a url",
	}
	for _, u := range internal {
		if !isInternalTarget(u) {
			t.Errorf("internal target must be blocked: %s", u)
		}
	}
	external := []string{
		"https://www12.senado.leg.br/noticias",
		"https://g1.globo.com/politica/",
		"https://8.8.8.8/dns", // public IP is fine
	}
	for _, u := range external {
		if isInternalTarget(u) {
			t.Errorf("external target must be allowed: %s", u)
		}
	}
}
