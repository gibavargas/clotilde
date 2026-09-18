// Package websearch provides a self-contained web search harness in Go.
//
// Pipeline: SERP scraping (Bing first — it keeps serving organic results to
// datacenter IPs; DDG HTML/lite and Mojeek as fallbacks) -> concurrent page
// fetching/extraction -> ranked SearchResults. No internal deadline is
// applied: every call is bounded only by the caller's context, whose sole
// source of cancellation is Apple Shortcuts' client-side timeout.
package websearch

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

// Result is one piece of web evidence returned by the harness.
type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// Search performs a SERP search and enriches the top results with page
// text extracted concurrently. Bounded only by ctx — no budget of its own;
// the request context (canceled when Apple Shortcuts drops the connection)
// is the single deadline.
func Search(ctx context.Context, query string, maxResults int) ([]Result, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("websearch: empty query")
	}
	if maxResults <= 0 {
		maxResults = 6
	}

	started := time.Now()
	hits, err := searchSERP(ctx, query, maxResults+4)
	if err != nil {
		return nil, err
	}
	if len(hits) == 0 {
		return nil, errors.New("websearch: no results")
	}

	type enriched struct {
		idx int
		res Result
	}
	results := make([]Result, len(hits))
	var wg sync.WaitGroup
	for i, hit := range hits {
		results[i] = Result{Title: hit.title, URL: hit.url, Snippet: hit.snippet}
		wg.Add(1)
		go func(i int, url string, base string) {
			defer wg.Done()
			if text := fetchPageText(ctx, url); text != "" {
				results[i].Snippet = truncate(text, 1200)
			}
			_ = base
		}(i, hit.url, hit.base)
	}
	wg.Wait()

	// Drop entries that ended up with no usable text at all.
	out := make([]Result, 0, len(results))
	for _, r := range results {
		if r.Snippet != "" || r.Title != "" {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("websearch: no usable results")
	}
	log.Printf("websearch: %d results in %v", len(out), time.Since(started).Round(time.Millisecond))
	return out, nil
}

type ddgHit struct {
	title   string
	url     string
	snippet string
	base    string
}

var (
	linkRe        = regexp.MustCompile(`<a[^>]+class="[^"]*result__a[^"]*"[^>]+href="([^"]+)"[^>]*>(.*?)</a>`)
	snipRe        = regexp.MustCompile(`class="[^"]*result__snippet[^"]*"[^>]*>(.*?)</a>`)
	genericLinkRe = regexp.MustCompile(`<a[^>]+href="(https?://[^"]+)"[^>]*>(.*?)</a>`)
	bingH2Re      = regexp.MustCompile(`(?s)<h2[^>]*><a[^>]+href="([^"]+)"[^>]*>(.*?)</a>`)
	bingURlRe     = regexp.MustCompile(`[?&]u=a1?([A-Za-z0-9_-]+)`)
	tagRe         = regexp.MustCompile(`<[^>]+>`)
	spaceRe       = regexp.MustCompile(`\s+`)
	uddgRe        = regexp.MustCompile(`[?&]uddg=([^&]+)`)
	metaCharsetRe = regexp.MustCompile(`(?i)<meta[^>]+charset=["']?([a-z0-9_-]+)`)
)

// engines lists the SERP endpoints tried in order. Bing is primary because it
// keeps serving organic results to datacenter IPs; DDG intermittently answers
// 202 (bot challenge) and Mojeek/Ecosia often 403 those same IPs.
var engines = []struct {
	name string
	url  string
}{
	{"bing", "https://www.bing.com/search?q=%s&mkt=pt-BR&count=16"},
	{"ddg-html", "https://html.duckduckgo.com/html/?q=%s"},
	{"ddg-lite", "https://lite.duckduckgo.com/lite/?q=%s"},
	{"mojeek", "https://www.mojeek.com/search?q=%s"},
}

func searchSERP(ctx context.Context, query string, limit int) ([]ddgHit, error) {
	var lastErr error
	for _, engine := range engines {
		// Fresh cookie jar per engine: Bing serves a cookie-setting
		// interstitial ("Please click here...") to headerless clients; the
		// second request with the same jar returns organic results.
		jar, _ := cookiejar.New(nil)
		// No client Timeout: bounded by the request context (Apple Shortcuts
		// client-side timeout via cancellation).
		client := &http.Client{
			Transport: &http.Transport{Proxy: http.ProxyFromEnvironment, DisableCompression: true},
			Jar:       jar,
		}
		for attempt := 0; attempt < 2; attempt++ {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet,
				fmt.Sprintf(engine.url, url.QueryEscape(query)), nil)
			if err != nil {
				return nil, err
			}
			req.Header.Set("User-Agent", userAgent)
			req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
			req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.8")

			body, err := fetchBody(ctx, client, req)
			if err != nil {
				lastErr = err
				if strings.Contains(err.Error(), "status 202") {
					// Bot challenge: short backoff, then try next engine.
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(250 * time.Millisecond):
					}
				}
				break
			}
			hits := parseSERPLinks(body, limit)
			if len(hits) > 0 {
				return hits, nil
			}
			lastErr = errors.New("no results parsed")
		}
	}
	if lastErr == nil {
		lastErr = errors.New("all search engines failed")
	}
	return nil, lastErr
}

// fetchBody performs the request with the given client and decodes the body.
func fetchBody(ctx context.Context, client *http.Client, req *http.Request) (string, error) {
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 600*1024))
	if err != nil {
		return "", err
	}
	return decodeToUTF8(raw, resp.Header.Get("Content-Type")), nil
}

// parseSERPLinks extracts organic result links from any of the supported SERP
// layouts: Bing h2 anchors first, then DDG result__a anchors, otherwise all
// external http(s) anchors with meaningful titles.
func parseSERPLinks(body string, limit int) []ddgHit {
	// Bing: h2>a links are organic results; hrefs are bing.com/ck/a redirects
	// whose target URL is base64url-encoded in the u=a1... parameter.
	if hits := parseBingHits(body, limit); len(hits) > 0 {
		return hits
	}

	if matches := linkRe.FindAllStringSubmatch(body, limit); len(matches) > 0 {
		var hits []ddgHit
		for i, m := range matches {
			final := normalizeDDGURL(html.UnescapeString(m[1]))
			if final == "" {
				continue
			}
			snippet := ""
			if i < len(snippets(body)) {
				snippet = snippets(body)[i]
			}
			hits = append(hits, ddgHit{title: stripTags(m[2]), url: final, snippet: snippet})
		}
		if len(hits) > 0 {
			return hits
		}
	}

	// Generic extraction (DDG Lite / Mojeek / layout drift).
	seen := map[string]bool{}
	var hits []ddgHit
	for _, m := range genericLinkRe.FindAllStringSubmatch(body, 200) {
		raw := html.UnescapeString(m[1])
		final := normalizeDDGURL(raw)
		if final == "" || seen[final] {
			continue
		}
		parsed, err := url.Parse(final)
		if err != nil || strings.Contains(parsed.Host, "duckduckgo.com") || strings.Contains(parsed.Host, "mojeek.com") || strings.Contains(parsed.Host, "bing.com") {
			continue
		}
		title := stripTags(m[2])
		if len([]rune(title)) < 3 {
			continue
		}
		seen[final] = true
		hits = append(hits, ddgHit{title: title, url: final})
		if len(hits) >= limit {
			break
		}
	}
	return hits
}

func snippets(body string) []string {
	var out []string
	for _, m := range snipRe.FindAllStringSubmatch(body, -1) {
		out = append(out, stripTags(m[1]))
	}
	return out
}

// parseBingHits extracts organic results from a Bing SERP, decoding the
// base64url target URLs embedded in bing.com/ck/a redirect links.
func parseBingHits(body string, limit int) []ddgHit {
	matches := bingH2Re.FindAllStringSubmatch(body, limit)
	if len(matches) == 0 {
		return nil
	}
	var hits []ddgHit
	seen := map[string]bool{}
	for _, m := range matches {
		href := html.UnescapeString(m[1])
		final := normalizeDDGURL(href)
		if final == "" {
			if burl := bingURlRe.FindStringSubmatch(href); burl != nil {
				final = decodeBingB64(burl[1])
			}
		}
		if final == "" || seen[final] || strings.Contains(final, "bing.com") {
			continue
		}
		title := stripTags(m[2])
		if len([]rune(title)) < 3 {
			continue
		}
		seen[final] = true
		hits = append(hits, ddgHit{title: title, url: final})
	}
	return hits
}

// decodeBingB64 decodes the base64url payload of a Bing ck/a redirect.
func decodeBingB64(s string) string {
	if pad := len(s) % 4; pad != 0 {
		s += strings.Repeat("=", 4-pad)
	}
	decoded, err := base64.URLEncoding.DecodeString(s)
	if err != nil || !(strings.HasPrefix(string(decoded), "http://") || strings.HasPrefix(string(decoded), "https://")) {
		return ""
	}
	return string(decoded)
}

// normalizeDDGURL unwraps DDG redirect links into the target URL.
func normalizeDDGURL(href string) string {
	if href == "" {
		return ""
	}
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if u.Host != "" && u.Host != "duckduckgo.com" && u.Host != "www.duckduckgo.com" {
		if u.Scheme == "http" || u.Scheme == "https" {
			return u.String()
		}
		return ""
	}
	if m := uddgRe.FindStringSubmatch(href); m != nil {
		if decoded, err := url.QueryUnescape(m[1]); err == nil {
			if ru, err := url.Parse(decoded); err == nil && (ru.Scheme == "http" || ru.Scheme == "https") {
				return ru.String()
			}
		}
	}
	return ""
}

// isInternalTarget reports whether the URL points at loopback, private,
// or link-local infrastructure. Search-result URLs are attacker-influenceable,
// so the page fetcher must never probe internal endpoints.
func isInternalTarget(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	host := u.Hostname()
	if host == "localhost" || host == "" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
	}
	// Hostname endings that only resolve inside private networks.
	for _, suffix := range []string{".local", ".internal", ".lan", ".home", ".corp"} {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}

// fetchPageText downloads a page (HTML with graceful JSON fallback) and
// extracts readable text, best-effort; bounded only by the caller's context.
func fetchPageText(ctx context.Context, pageURL string) string {
	if isInternalTarget(pageURL) {
		return ""
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.8")

	body, err := httpBody(ctx, req)
	if err != nil {
		return ""
	}

	trimmed := strings.TrimSpace(body)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return extractJSONStrings(trimmed, 1200)
	}
	return extractHTMLText(body)
}

func httpBody(ctx context.Context, req *http.Request) (string, error) {
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, DisableCompression: true}
	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 600*1024))
	if err != nil {
		return "", err
	}

	// Some servers ignore DisableCompression or force-compress responses.
	if len(raw) > 2 {
		if c, cerr := gzip.NewReader(bytes.NewReader(raw)); cerr == nil {
			if inflated, ierr := io.ReadAll(io.LimitReader(c, 600*1024)); ierr == nil {
				raw = inflated
			}
			c.Close()
		} else if len(raw) > 4 {
			if inflated, ferr := io.ReadAll(flate.NewReader(bytes.NewReader(raw[2:]))); ferr == nil && len(inflated) > len(raw) {
				raw = inflated
			}
		}
	}

	body := decodeToUTF8(raw, resp.Header.Get("Content-Type"))
	if !looksLikeText(body) {
		return "", errors.New("binary response body")
	}
	return body, nil
}

// decodeToUTF8 converts latin-1/utf-16 bodies to UTF-8 using the HTTP header
// and meta tag hints; utf-8 passes through unchanged.
func decodeToUTF8(raw []byte, contentType string) string {
	if utf8.Valid(raw) {
		return string(raw)
	}
	charsetName := ""
	if _, params, err := mime.ParseMediaType(contentType); err == nil {
		charsetName = params["charset"]
	}
	if charsetName == "" {
		head := string(raw[:min(len(raw), 1024)])
		if m := metaCharsetRe.FindStringSubmatch(head); m != nil {
			charsetName = m[1]
		}
	}
	switch strings.ToLower(strings.TrimSpace(charsetName)) {
	case "iso-8859-1", "latin1", "latin-1", "windows-1252", "cp1252":
		if out, err := charmap.ISO8859_1.NewDecoder().Bytes(raw); err == nil {
			return string(out)
		}
	case "utf-16", "utf-16le", "utf-16be":
		if out, err := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder().Bytes(raw); err == nil {
			return string(out)
		}
	default:
		if out, err := charmap.ISO8859_1.NewDecoder().Bytes(raw); err == nil {
			return string(out)
		}
	}
	return string(raw)
}

// looksLikeText rejects bodies that decode to mostly control characters
// (binaries, encrypted payloads, images).
func looksLikeText(s string) bool {
	if s == "" {
		return false
	}
	if strings.Contains(s[:min(len(s), 512)], "\x00") {
		return false
	}
	probe := s[:min(len(s), 1024)]
	control := 0
	for _, r := range probe {
		if r < 0x20 && r != '\n' && r != '\r' && r != '	' {
			control++
		}
	}
	return control*100 < 512 // ~50% threshold over 1024 runes
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var (
	scriptRe = regexp.MustCompile(`(?is)<(script|style|noscript)[^>]*>.*?</(script|style|noscript)>`)
	blockRe  = regexp.MustCompile(`(?i)</(p|div|li|h[1-6]|tr|br)>`)
	textIsJS = regexp.MustCompile(`^[{}\[\]].*[;{}]$`)
)

// extractHTMLText strips scripts/styles/tags and returns visible text.
func extractHTMLText(body string) string {
	clean := scriptRe.ReplaceAllString(body, " ")
	clean = blockRe.ReplaceAllString(clean, "\n")
	clean = tagRe.ReplaceAllString(clean, " ")
	clean = html.UnescapeString(clean)

	var kept []string
	seen := map[string]bool{}
	for _, line := range strings.Split(clean, "\n") {
		line = strings.TrimSpace(spaceRe.ReplaceAllString(line, " "))
		if len(line) < 40 || len(line) > 2000 {
			continue
		}
		if textIsJS.MatchString(line) {
			continue
		}
		if seen[line] {
			continue
		}
		seen[line] = true
		kept = append(kept, line)
		if len(kept) >= 30 {
			break
		}
	}
	return truncate(strings.Join(kept, "\n"), 1200)
}

// extractJSONStrings pulls readable string values out of a JSON/JSONP body
// (many government/API pages return JSON where HTML scraping fails).
func extractJSONStrings(body string, maxLen int) string {
	var pieces []string
	inStr, esc := false, false
	var cur strings.Builder
	flush := func() {
		s := strings.TrimSpace(cur.String())
		cur.Reset()
		if len(s) >= 40 && !strings.ContainsAny(s, "{}[]<>\"\\") && !isURLOnly(s) {
			pieces = append(pieces, s)
		}
	}
	for _, r := range body {
		if esc {
			cur.WriteRune(r)
			esc = false
			continue
		}
		switch {
		case r == '\\':
			esc = true
		case r == '"':
			if inStr {
				flush()
			}
			inStr = !inStr
		default:
			if inStr {
				cur.WriteRune(r)
			}
		}
	}
	if len(pieces) == 0 {
		return ""
	}
	return truncate(strings.Join(pieces, "\n"), maxLen)
}

func isURLOnly(s string) bool {
	return strings.HasPrefix(s, "http") && !strings.Contains(s, " ")
}

func stripTags(s string) string {
	return strings.TrimSpace(spaceRe.ReplaceAllString(html.UnescapeString(tagRe.ReplaceAllString(s, " ")), " "))
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return strings.TrimSpace(string(r[:max])) + "..."
}

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"
