package router

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var punctuationRegexp = regexp.MustCompile(`[^a-z0-9\s]+`)

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
	text = punctuationRegexp.ReplaceAllString(text, " ")

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

// stem applies simple Portuguese suffix removal rules
func stem(word string) string {
	// Ignore short words
	if len(word) < 4 {
		return word
	}

	// Plurals (s)
	if strings.HasSuffix(word, "s") {
		word = strings.TrimSuffix(word, "s")
	}

	// Adverbs (mente)
	if strings.HasSuffix(word, "mente") {
		return strings.TrimSuffix(word, "mente")
	}

	// Gerund (ando, endo, indo)
	if strings.HasSuffix(word, "ndo") {
		word = strings.TrimSuffix(word, "ndo")
		return word
	}

	// Verbs/Nouns (ar, er, ir, or)
	if strings.HasSuffix(word, "ar") || strings.HasSuffix(word, "er") || strings.HasSuffix(word, "ir") || strings.HasSuffix(word, "or") {
		return word[:len(word)-2]
	}

	// Augmentative/Diminutive (inho, inha, ão, ões)
	if strings.HasSuffix(word, "inho") || strings.HasSuffix(word, "inha") {
		return word[:len(word)-4]
	}
	if strings.HasSuffix(word, "oes") { // after accent removal: ões -> oes
		return word[:len(word)-3] + "ao"
	}

	return word
}
