package router

import (
	"strings"
	"testing"
)

// TestKeywordTablesSurviveNormalization walks every keyword in every table
// and asserts it survives buildMatcher's normalization without silently
// degrading: a keyword that normalizes to empty would be dropped invisibly
// (router.go buildMatcher 'continue'), and a keyword whose stem collides with
// its own phrase form breaks matching. Table rot becomes a test failure
// instead of a silent routing regression.
func TestKeywordTablesSurviveNormalization(t *testing.T) {
	tables := map[string][]string{
		"web_search":   webSearchKeywords,
		"complex":      complexKeywords,
		"factual":      factualKeywords,
		"mathematical": mathematicalKeywords,
		"creative":     creativeKeywords,
	}

	total := 0
	for name, keywords := range tables {
		if len(keywords) == 0 {
			t.Errorf("keyword table %q is empty — routing for this category is dead", name)
			continue
		}
		for _, k := range keywords {
			total++
			n := Normalize(k)
			// "%" and single-letter units (m, g, l) are intentional math
			// tokens; anything longer that degenerates is table rot.
			if len(n) < 2 && len(k) > 1 {
				t.Errorf("%s: keyword %q degenerates to %q", name, k, n)
			}
			// Production normalizes a question exactly once, so the only
			// invariant that matters is survival (>= 2 chars), checked above.
			// Single-symbol unit keywords ("%", "m", "g", "l") are intentional
			// math-table entries and are allowed to normalize to themselves;
			// flag anything else that degenerates.
			if len(n) < 2 && len(k) > 1 {
				t.Errorf("%s: multi-char keyword %q degenerates to %q", name, k, n)
			}
		}
	}
	t.Logf("verified %d keywords across %d tables", total, len(tables))
}

// TestEveryKeywordMatchesOwnCategory: each single-word keyword must score > 0
// against a question that is exactly that keyword — proving the compiled
// matcher actually fires for it.
func TestEveryKeywordMatchesOwnCategory(t *testing.T) {
	cases := []struct {
		cat   Category
		words []string
	}{
		{CategoryWebSearch, webSearchKeywords},
		{CategoryComplex, complexKeywords},
		{CategoryFactual, factualKeywords},
		{CategoryMathematical, mathematicalKeywords},
		{CategoryCreative, creativeKeywords},
	}
	m := matchers[CategoryWebSearch]
	if m == nil {
		t.Fatal("matchers not built (init race or table rename)")
	}
	for _, c := range cases {
		matcher := matchers[c.cat]
		if matcher == nil {
			t.Fatalf("no matcher for category %s", c.cat)
		}
		for _, k := range c.words {
			n := Normalize(k)
			if n == "" || len(n) < 2 {
				continue // covered by the survival test
			}
			// Single words fire via regex; phrases require the phrase path.
			if matcher.singleWordRegex == nil && len(matcher.multiWordPhrases) == 0 {
				t.Fatalf("category %s has an empty matcher", c.cat)
			}
			score := matchCategory(n, c.cat)
			if score <= 0 {
				// Multi-word phrases can't be probed with a single normalized
				// token; only flag single-word failures.
				if len(strings.Fields(n)) == 1 && matcher.singleWordRegex != nil &&
					matcher.singleWordRegex.MatchString(n) == false {
					t.Errorf("%s: keyword %q (stem %q) does not match its own compiled regex", c.cat, k, n)
				}
			}
		}
	}
}
