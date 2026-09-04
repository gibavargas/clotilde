package router

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Normalize prepares text for matching by:
// 1. Lowercasing
// 2. Removing accents (diacritics)
// 3. Simple stemming (removing common suffixes)
func Normalize(text string) string {
	// 1. Lowercase
	text = strings.ToLower(text)

	// 2. Remove accents
	text = removeAccents(text)

	// 3. Remove punctuation (keep only letters and numbers)
	// Replace non-alphanumeric with space
	reg := regexp.MustCompile(`[^a-z0-9\s]+`)
	text = reg.ReplaceAllString(text, " ")

	// 4. Simple Stemming (RSLP-lite)
	// We apply a few simple rules to reduce words to their root.
	// This is not a full RSLP implementation but covers 80% of cases.
	words := strings.Fields(text)
	for i, word := range words {
		words[i] = stem(word)
	}

	return strings.Join(words, " ")
}

func removeAccents(s string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	result, _, _ := transform.String(t, s)
	return result
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}

// minStemLen is the shortest stem a suffix strip may produce. Stripping
// below this (e.g. "vinho" -> "v") destroys matching power and invites
// false positives on unrelated short tokens.
const minStemLen = 3

// stem applies simple Portuguese suffix removal rules (RSLP-lite). Every
// strip is guarded so the remaining stem keeps at least minStemLen runes.
func stem(word string) string {
	// Ignore short words
	if len(word) < 4 {
		return word
	}

	stemOK := func(stemmed string) bool {
		return len([]rune(stemmed)) >= minStemLen
	}

	// Plurals (s)
	if strings.HasSuffix(word, "s") && stemOK(word[:len(word)-1]) {
		word = word[:len(word)-1]
	}

	// Adverbs (mente)
	if strings.HasSuffix(word, "mente") {
		if s := word[:len(word)-len("mente")]; stemOK(s) {
			return s
		}
		return word
	}

	// Gerund (ando, endo, indo)
	if strings.HasSuffix(word, "ndo") {
		if s := word[:len(word)-3]; stemOK(s) {
			return s
		}
		return word
	}

	// Verbs/Nouns (ar, er, ir, or)
	if strings.HasSuffix(word, "ar") || strings.HasSuffix(word, "er") ||
		strings.HasSuffix(word, "ir") || strings.HasSuffix(word, "or") {
		if s := word[:len(word)-2]; stemOK(s) {
			return s
		}
		return word
	}

	// Augmentative/Diminutive (inho, inha)
	if strings.HasSuffix(word, "inho") || strings.HasSuffix(word, "inha") {
		if s := word[:len(word)-4]; stemOK(s) {
			return s
		}
		return word
	}
	if strings.HasSuffix(word, "oes") { // after accent removal: ões -> oes
		if s := word[:len(word)-3]; stemOK(s) {
			return s + "ao"
		}
		return word
	}

	return word
}
