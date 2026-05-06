package admin

import "strings"

// validModels is the runtime allowlist for user-selectable providers.
// Keep this centralized so admin UI validation, API validation, and tests do not
// drift as provider model names change.
var validModels = map[string]bool{
	// GPT-4o series
	"gpt-4o":            true,
	"gpt-4o-mini":       true,
	"gpt-4o-2024-08-06": true,
	"chatgpt-4o-latest": true,

	// GPT-4 series
	"gpt-4-turbo":   true,
	"gpt-3.5-turbo": true,

	// GPT-4.1 series
	"gpt-4.1":      true,
	"gpt-4.1-mini": true,
	"gpt-4.1-nano": true,

	// GPT-5 series
	"gpt-5":      true,
	"gpt-5.1":    true,
	"gpt-5-mini": true,
	"gpt-5-nano": true,
	"gpt-5-pro":  true,

	// O-series reasoning models
	"o1":      true,
	"o1-mini": true,
	"o1-pro":  true,
	"o3":      true,
	"o3-mini": true,
	"o4-mini": true,

	// Claude models
	"claude-haiku-4-5-20251001":  true,
	"claude-3-5-haiku-20241022":  true,
	"claude-3-5-haiku-latest":    true,
	"claude-3-5-sonnet-20241022": true,
	"claude-3-5-sonnet-latest":   true,
	"claude-sonnet-4-20250514":   true,
	"claude-3-opus-20240229":     true,

	// OpenRouter curated shortcuts. Additional OpenRouter slugs are accepted
	// when prefixed with openrouter/ and shaped as provider/model.
	"openrouter/anthropic/claude-haiku-4.5": true,
}

func IsValidModel(model string) bool {
	if validModels[model] {
		return true
	}
	return isValidOpenRouterModel(model)
}

func isValidOpenRouterModel(model string) bool {
	slug, ok := strings.CutPrefix(model, "openrouter/")
	if !ok {
		return false
	}
	if strings.TrimSpace(slug) != slug || strings.ContainsAny(slug, "\x00\r\n\t ") {
		return false
	}
	provider, name, ok := strings.Cut(slug, "/")
	return ok && provider != "" && name != ""
}
