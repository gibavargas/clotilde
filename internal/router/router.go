package router

import (
	"log"
	"regexp"
	"strings"

	"github.com/clotilde/carplay-assistant/internal/admin"
)

// Category represents the routing category
type Category string

const (
	CategoryWebSearch    Category = "web_search"
	CategoryComplex      Category = "complex"
	CategoryFactual      Category = "factual"
	CategoryMathematical Category = "mathematical"
	CategoryCreative     Category = "creative"
	CategorySimple       Category = "simple" // Default fallback
)

// RouteDecision contains routing information for a request
type RouteDecision struct {
	Category        Category
	Model           string
	WebSearch       bool
	ReasoningEffort string // "none", "low", "medium", "high" - empty means no reasoning config
}

// Models that support web search in Responses API
var modelsWithWebSearch = map[string]bool{
	"gpt-4o":            true,
	"gpt-4o-mini":       true,
	"gpt-4o-2024-08-06": true,
	"chatgpt-4o-latest": true,
	"gpt-4-turbo":       true,
	"gpt-4.1":           true,
	"gpt-4.1-mini":      true,
	// gpt-4.1-nano does NOT support web search
	// gpt-5 series needs reasoning >= "low" for web search
	"gpt-5":     true, // with reasoning
	"gpt-5.1":   true, // with reasoning
	"gpt-5-pro": true, // with reasoning
	// gpt-5-mini and gpt-5-nano may not support web search
	"gpt-5-mini": true,
	"o3":         true,
	"o3-mini":    true,
	"o4-mini":    true,
}

// Fallback model for web search when configured model doesn't support it
const webSearchFallbackModel = "gpt-4o-mini"

// Category scoring weights (stronger keywords get higher weight)
var categoryWeights = map[Category]float64{
	CategoryWebSearch:    1.0,
	CategoryComplex:      1.0,
	CategoryFactual:      0.8, // Slightly lower to avoid false positives
	CategoryMathematical: 1.0,
	CategoryCreative:     1.0, // Increased from 0.9 to ensure single strong keywords trigger it
}

// Minimum score threshold for category selection
const minCategoryScore = 1.0

// categoryMatcher holds pre-compiled regexes for a category
type categoryMatcher struct {
	singleWordRegex  *regexp.Regexp
	multiWordPhrases []string
	negativeRegex    *regexp.Regexp // Regex for negative keywords
}

var matchers map[Category]*categoryMatcher

func init() {
	matchers = make(map[Category]*categoryMatcher)

	// Helper to build matcher
	buildMatcher := func(cat Category, keywords []string) {
		var singleWords []string
		var phrases []string

		for _, k := range keywords {
			// Normalize keyword (stemming, accent removal)
			k = Normalize(k)
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			if strings.Contains(k, " ") {
				phrases = append(phrases, k)
			} else {
				singleWords = append(singleWords, regexp.QuoteMeta(k))
			}
		}

		var re *regexp.Regexp
		if len(singleWords) > 0 {
			// \b(word1|word2|...)\b
			pattern := `\b(` + strings.Join(singleWords, "|") + `)\b`
			re = regexp.MustCompile(pattern)
		}

		// Build negative matcher if exists
		var negRe *regexp.Regexp
		if negs, ok := negativeKeywords[string(cat)]; ok && len(negs) > 0 {
			var negWords []string
			for _, k := range negs {
				// Normalize negative keywords too
				k = Normalize(k)
				negWords = append(negWords, regexp.QuoteMeta(k))
			}
			// Match any negative keyword
			negPattern := `\b(` + strings.Join(negWords, "|") + `)\b`
			negRe = regexp.MustCompile(negPattern)
		}

		matchers[cat] = &categoryMatcher{
			singleWordRegex:  re,
			multiWordPhrases: phrases,
			negativeRegex:    negRe,
		}
	}

	buildMatcher(CategoryWebSearch, webSearchKeywords)
	buildMatcher(CategoryComplex, complexKeywords)
	buildMatcher(CategoryFactual, factualKeywords)
	buildMatcher(CategoryMathematical, mathematicalKeywords)
	buildMatcher(CategoryCreative, creativeKeywords)
}

// matchCategory scores a question against a category's keywords using pre-compiled regexes
func matchCategory(question string, cat Category) float64 {
	// Normalize question (stemming, accent removal)
	questionNormalized := Normalize(question)
	score := 0.0

	matcher := matchers[cat]
	if matcher == nil {
		return 0.0
	}

	// Check negative keywords first - fail fast
	if matcher.negativeRegex != nil {
		if matcher.negativeRegex.MatchString(questionNormalized) {
			// Found a negative keyword, return 0 immediately
			return 0.0
		}
	}

	// Check phrases
	for _, phrase := range matcher.multiWordPhrases {
		if strings.Contains(questionNormalized, phrase) {
			score += 1.0
		}
	}

	// Check single words
	if matcher.singleWordRegex != nil {
		// FindAllStringIndex returns all matches
		matches := matcher.singleWordRegex.FindAllStringIndex(questionNormalized, -1)
		score += float64(len(matches))
	}

	return score
}

// Route determines which category, model, and tools to use based on question
func Route(question string) RouteDecision {
	config := admin.GetConfig()
	standardModel := config.StandardModel
	premiumModel := config.PremiumModel

	// Score each category
	scores := map[Category]float64{
		CategoryWebSearch:    matchCategory(question, CategoryWebSearch) * categoryWeights[CategoryWebSearch],
		CategoryComplex:      matchCategory(question, CategoryComplex) * categoryWeights[CategoryComplex],
		CategoryFactual:      matchCategory(question, CategoryFactual) * categoryWeights[CategoryFactual],
		CategoryMathematical: matchCategory(question, CategoryMathematical) * categoryWeights[CategoryMathematical],
		CategoryCreative:     matchCategory(question, CategoryCreative) * categoryWeights[CategoryCreative],
	}

	// Find highest scoring category with deterministic tie-breaking
	var bestCategory Category = CategorySimple
	maxScore := 0.0

	// Priority order for tie-breaking
	priorityOrder := []Category{
		CategoryMathematical, // Specific intent
		CategoryWebSearch,    // Specific intent
		CategoryCreative,     // Specific intent
		CategoryComplex,      // Broad intent
		CategoryFactual,      // Broad intent
	}

	for _, category := range priorityOrder {
		score := scores[category]
		if score > maxScore && score >= minCategoryScore {
			maxScore = score
			bestCategory = category
		}
	}

	// Determine model, web search, and reasoning based on category
	// Check for category-specific model override first
	var model string
	var webSearch bool
	var reasoningEffort string

	categoryKey := string(bestCategory)
	if categoryModel, exists := config.CategoryModels[categoryKey]; exists && categoryModel != "" {
		// Use category-specific model override
		model = categoryModel
		log.Printf("Route: %s (using category model override: %s, score: %.1f)", bestCategory, model, maxScore)
	} else {
		// Use default model for category
		switch bestCategory {
		case CategoryWebSearch:
			model = standardModel
			log.Printf("Route: WEB_SEARCH (category score: %.1f)", maxScore)

		case CategoryComplex:
			model = premiumModel
			log.Printf("Route: COMPLEX (category score: %.1f)", maxScore)

		case CategoryFactual:
			model = standardModel
			log.Printf("Route: FACTUAL (category score: %.1f)", maxScore)

		case CategoryMathematical:
			model = standardModel
			log.Printf("Route: MATHEMATICAL (category score: %.1f)", maxScore)

		case CategoryCreative:
			model = premiumModel
			log.Printf("Route: CREATIVE (category score: %.1f)", maxScore)

		default: // CategorySimple
			model = standardModel
			log.Printf("Route: SIMPLE (no category matched, using default)")
		}
	}

	// Determine web search requirement
	switch bestCategory {
	case CategoryWebSearch:
		webSearch = true
	default:
		webSearch = false
	}

	// If web search is needed, ensure the model supports it
	if webSearch {
		isClaude := strings.HasPrefix(model, "claude-")
		supportsWebSearch := modelsWithWebSearch[model]
		isGPT5 := strings.HasPrefix(model, "gpt-5")

		// Claude models support web search via Perplexity integration (handled in main.go)
		// Only allow if Perplexity is enabled in config
		if isClaude && config.PerplexityEnabled {
			// Claude + Perplexity is supported, no fallback needed
			// Reasoning effort not applicable to Claude
		} else if !supportsWebSearch {
			log.Printf("Model %s does not support web search, using fallback: %s", model, webSearchFallbackModel)
			model = webSearchFallbackModel
			reasoningEffort = ""
		} else if isGPT5 {
			reasoningEffort = "medium"
			log.Printf("gpt-5 with web search: using reasoning='medium' (minimum required)")
		}
	}

	return RouteDecision{
		Category:        bestCategory,
		Model:           model,
		WebSearch:       webSearch,
		ReasoningEffort: reasoningEffort,
	}
}
