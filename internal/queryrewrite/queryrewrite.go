// Package queryrewrite turns Clotilde's spoken questions into effective web
// search queries before evidence is fetched. It is fully deterministic so
// answers stay reproducible and tests cover the transformations.
package queryrewrite

import (
	"regexp"
	"strings"
	"time"
)

var (
	noisePhrases = []string{
		"por favor",
		"acabei de ver",
		"acabei de ler",
		"me diz",
		"me fala",
		"quero saber",
		"procure por",
		"pesquise por",
		"pesquisa sobre",
		"me explica",
		"pode me dizer",
		"será que",
		"sera que",
	}

	stopwords = map[string]bool{
		"a": true, "as": true, "o": true, "os": true, "de": true, "da": true,
		"do": true, "das": true, "dos": true, "e": true, "em": true, "no": true,
		"na": true, "nos": true, "nas": true, "um": true, "uma": true,
		"para": true, "com": true, "que": true, "sobre": true, "the": true,
		"of": true, "to": true, "in": true, "is": true, "é": true, "são": true,
		"foi": true, "ser": true,
	}

	interrogativeSet = map[string]bool{
		"qual": true, "quais": true, "quem": true, "quando": true,
		"onde": true, "quanto": true, "quantos": true, "quantas": true, "como": true,
	}

	// domainHints expands a single spoken keyword into the phrasing a search
	// engine actually rewards.
	domainHints = map[string]string{
		"senador":    "senado federal senador",
		"dólar":      "cotação dólar",
		"dolar":      "cotação dólar",
		"placa":      "placa",
		"presidente": "presidente",
		"clima":      "clima tempo",
		"combustível": "preço combustível",
		"pedágio":    "preço pedágio",
	}

	wordRe = regexp.MustCompile(`[\p{L}\p{N}]+`)

	// temporalHints add the current year so results stay fresh; matches are
	// checked against the lowercased question.
	temporalHints = []string{"hoje", "ontem", "amanhã", "recente", "atual", "agora"}
)

// Rewrite converts a spoken question into one or two search queries. The
// first entry is the primary keyword-style query; an optional second entry
// is a truncated literal version of the question for cases where exact
// wording matters.
func Rewrite(question string, now time.Time) []string {
	q := strings.ToLower(strings.TrimSpace(question))
	if q == "" {
		return []string{"notícias hoje"}
	}

	// Strip speech noise.
	cleaned := q
	for _, phrase := range noisePhrases {
		if strings.Contains(cleaned, phrase) {
			cleaned = strings.ReplaceAll(cleaned, phrase, " ")
		}
	}

	// Salient keywords, in order, without stopwords/interrogatives.
	var salient []string
	sawTemporal := ""
	for _, w := range wordRe.FindAllString(cleaned, -1) {
		if stopwords[w] || interrogativeSet[w] {
			continue
		}
		if _, isTemporal := temporalYearSuffix(w, now); isTemporal && sawTemporal == "" {
			sawTemporal = w
		}
		salient = append(salient, w)
	}
	if len(salient) == 0 {
		salient = []string{strings.Join(strings.Fields(cleaned), " ")}
	}

	// Apply at most one domain hint, skipping hints whose first token is
	// already present (avoids "senador senado federal senador" duplication).
	expanded := false
	for i, w := range salient {
		hint, ok := domainHints[w]
		if !ok || hint == w {
			// Identity hints ("placa" -> "placa") are no-ops; keep scanning
			// so a real expansion elsewhere still applies.
			continue
		}
		first := strings.Fields(hint)[0]
		already := false
		for _, other := range salient {
			if other == first {
				already = true
				break
			}
		}
		if !already {
			salient[i] = hint
		}
		expanded = true
		break
	}
	_ = expanded

	// Temporal anchoring: a temporal word gains the current year so engines
	// stop serving stale pages.
	year := now.Format("2006")
	for i, w := range salient {
		if w == sawTemporal {
			salient[i] = w + " " + year
			break
		}
	}

	primary := truncateTokens(strings.Join(salient, " "), 12)

	var out []string
	out = append(out, strings.TrimSpace(primary))

	// Secondary literal query: keep the user's wording when it carries
	// information the keyword query would lose.
	if len(primary) < len(cleaned)-3 && len(out) == 1 {
		secondary := truncateTokens(strings.TrimSpace(cleaned), 12)
		if secondary != "" && secondary != out[0] {
			out = append(out, secondary)
		}
	}
	return out
}

// temporalYearSuffix reports whether w is a temporal hint word (the year
// injection trigger). Extended in one place so callers stay simple.
func temporalYearSuffix(w string, now time.Time) (string, bool) {
	for _, t := range []string{"hoje", "ontem", "amanhã", "recente", "atual"} {
		if w == t {
			return now.Format("2006"), true
		}
	}
	return "", false
}

func truncateTokens(q string, max int) string {
	tokens := strings.Fields(q)
	if len(tokens) <= max {
		return q
	}
	return strings.Join(tokens[:max], " ")
}
