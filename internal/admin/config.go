package admin

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"
)

// RuntimeConfig holds runtime configuration that can be changed via admin UI
type RuntimeConfig struct {
	BaseSystemPrompt  string            `json:"base_system_prompt"` // Core principles (shared by all)
	CategoryPrompts   map[string]string `json:"category_prompts"`   // category -> prompt override (optional)
	StandardModel     string            `json:"standard_model"`     // Fast/cheap model (e.g., gpt-4.1-mini)
	PremiumModel      string            `json:"premium_model"`      // Powerful model (default: gpt-4.1-mini, also supports gpt-4.1, o3)
	CategoryModels    map[string]string `json:"category_models"`    // category -> model override (optional)
	PerplexityEnabled bool              `json:"perplexity_enabled"` // Enable Perplexity Search API for web search (default: true)

	// Legacy field for backward compatibility
	SystemPrompt string `json:"system_prompt,omitempty"`
}

var (
	configMutex   sync.RWMutex
	runtimeConfig = RuntimeConfig{
		BaseSystemPrompt:  "", // Will be initialized with default from main.go
		CategoryPrompts:   nil,
		CategoryModels:    make(map[string]string),
		StandardModel:     "claude-haiku-4-5-20251001", // Claude Haiku 4.5 - fastest, ideal for CarPlay
		PremiumModel:      "claude-haiku-4-5-20251001", // Same fast model for all queries to avoid timeouts
		PerplexityEnabled: true,                      // Default: enabled
	}
	defaultCategoryPrompts = make(map[string]string) // Store default category prompts
	initialized            = false
)

// SetDefaultConfig initializes the config with default values from main.go
// This should be called once at startup with the default system prompt template
func SetDefaultConfig(defaultSystemPrompt string) {
	configMutex.Lock()
	defer configMutex.Unlock()

	if !initialized {
		runtimeConfig.BaseSystemPrompt = defaultSystemPrompt
		// Legacy support
		runtimeConfig.SystemPrompt = defaultSystemPrompt
		initialized = true
	}
}

// SetDefaultCategoryPrompts stores the default category prompts for UI display
// This should be called once at startup with the default category prompt templates
func SetDefaultCategoryPrompts(prompts map[string]string) {
	configMutex.Lock()
	defer configMutex.Unlock()

	for k, v := range prompts {
		defaultCategoryPrompts[k] = v
	}
}

// GetConfig returns a copy of the current runtime configuration
// If category prompts are not set, returns default category prompts for UI display
func GetConfig() RuntimeConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()

	// Deep copy maps
	categoryPrompts := make(map[string]string)

	// If category prompts are set in runtime config, use those
	// Otherwise, use defaults for UI display
	// Check if map is nil (never set) vs empty (explicitly cleared)
	if runtimeConfig.CategoryPrompts != nil {
		// Map exists - use it (even if empty, user explicitly cleared it)
		for k, v := range runtimeConfig.CategoryPrompts {
			categoryPrompts[k] = v
		}
	} else {
		// Map is nil - never been set, return defaults so UI can display them
		for k, v := range defaultCategoryPrompts {
			categoryPrompts[k] = v
		}
	}

	categoryModels := make(map[string]string)
	for k, v := range runtimeConfig.CategoryModels {
		categoryModels[k] = v
	}

	return RuntimeConfig{
		BaseSystemPrompt:  runtimeConfig.BaseSystemPrompt,
		CategoryPrompts:   categoryPrompts,
		CategoryModels:    categoryModels,
		StandardModel:     runtimeConfig.StandardModel,
		PremiumModel:      runtimeConfig.PremiumModel,
		PerplexityEnabled: runtimeConfig.PerplexityEnabled,
		// Legacy support
		SystemPrompt: runtimeConfig.BaseSystemPrompt,
	}
}

// validateSystemPrompt validates the system prompt content
func validateSystemPrompt(prompt string) error {
	// Check for null bytes (security risk)
	if bytes.Contains([]byte(prompt), []byte{0}) {
		return &ConfigError{Field: "system_prompt", Message: "System prompt contains null bytes"}
	}

	// Check if valid UTF-8
	if !utf8.ValidString(prompt) {
		return &ConfigError{Field: "system_prompt", Message: "System prompt contains invalid UTF-8"}
	}

	// Validate format string placeholder - must contain exactly one %s
	placeholderCount := strings.Count(prompt, "%s")
	if placeholderCount == 0 {
		return &ConfigError{Field: "system_prompt", Message: "System prompt must contain exactly one %s placeholder for date/time"}
	}
	if placeholderCount > 1 {
		return &ConfigError{Field: "system_prompt", Message: fmt.Sprintf("System prompt must contain exactly one %%s placeholder (found %d)", placeholderCount)}
	}

	// Check for potentially dangerous format strings (but allow %s)
	// Count % that aren't part of %s
	dangerousPatterns := []string{"%d", "%f", "%v", "%+v", "%#v", "%x", "%X", "%p"}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(prompt, pattern) {
			return &ConfigError{Field: "system_prompt", Message: fmt.Sprintf("System prompt contains unsupported format specifier: %s", pattern)}
		}
	}

	// Check for escaped %s (%%s) - this would break the template
	if strings.Contains(prompt, "%%s") {
		return &ConfigError{Field: "system_prompt", Message: "System prompt contains escaped placeholder (%%s)"}
	}

	return nil
}

// SetConfig updates the runtime configuration
// Returns error if validation fails
func SetConfig(newConfig RuntimeConfig) error {
	// All models that can be used - OpenAI and Claude (Anthropic)
	validModels := map[string]bool{
		// GPT-4o series (confirmed working)
		"gpt-4o":            true,
		"gpt-4o-mini":       true,
		"gpt-4o-2024-08-06": true,
		"chatgpt-4o-latest": true,
		// GPT-4 series
		"gpt-4-turbo":   true,
		"gpt-3.5-turbo": true,
		// GPT-4.1 series (may require specific API access)
		"gpt-4.1":      true,
		"gpt-4.1-mini": true,
		"gpt-4.1-nano": true,
		// GPT-5 series (may require specific API access)
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
		// Claude models (Anthropic) - FAST, ideal for CarPlay
		// Claude Haiku 4.5 is extremely fast (~1-3s), best for CarPlay
		"claude-haiku-4-5-20251001": true, // Latest Haiku 4.5 - fastest, best for CarPlay
		// Older Claude models (for backward compatibility)
		"claude-3-5-haiku-20241022":  true,
		"claude-3-5-haiku-latest":    true,
		"claude-3-5-sonnet-20241022": true,
		"claude-3-5-sonnet-latest":   true,
		"claude-sonnet-4-20250514":   true, // Claude Sonnet 4
		"claude-3-opus-20240229":     true, // Most capable but slower
	}

	if !validModels[newConfig.StandardModel] {
		return &ConfigError{Field: "standard_model", Message: "Invalid standard model"}
	}

	if !validModels[newConfig.PremiumModel] {
		return &ConfigError{Field: "premium_model", Message: "Invalid premium model"}
	}

	// Determine which prompt to validate (prefer BaseSystemPrompt, fallback to SystemPrompt for legacy)
	promptToValidate := newConfig.BaseSystemPrompt
	if promptToValidate == "" {
		promptToValidate = newConfig.SystemPrompt
	}

	// Validate base system prompt content (must have %s placeholder)
	if promptToValidate != "" {
		if err := validateSystemPrompt(promptToValidate); err != nil {
			return err
		}
	} else {
		// Base prompt is required
		return &ConfigError{Field: "base_system_prompt", Message: "Base system prompt is required"}
	}

	// Validate category prompts if provided
	for category, prompt := range newConfig.CategoryPrompts {
		if prompt != "" {
			// Category prompts don't need %s placeholder (they're appended to base)
			// But still check for null bytes and UTF-8
			if bytes.Contains([]byte(prompt), []byte{0}) {
				return &ConfigError{Field: "category_prompts." + category, Message: "Category prompt contains null bytes"}
			}
			if !utf8.ValidString(prompt) {
				return &ConfigError{Field: "category_prompts." + category, Message: "Category prompt contains invalid UTF-8"}
			}
		}
	}

	// Validate category models if provided
	for category, model := range newConfig.CategoryModels {
		if !validModels[model] {
			return &ConfigError{Field: "category_models." + category, Message: "Invalid model for category"}
		}
	}

	configMutex.Lock()
	defer configMutex.Unlock()

	// Update base prompt (prefer BaseSystemPrompt, fallback to SystemPrompt for legacy)
	if newConfig.BaseSystemPrompt != "" {
		runtimeConfig.BaseSystemPrompt = newConfig.BaseSystemPrompt
		runtimeConfig.SystemPrompt = newConfig.BaseSystemPrompt // Legacy support
	} else if newConfig.SystemPrompt != "" {
		runtimeConfig.BaseSystemPrompt = newConfig.SystemPrompt
		runtimeConfig.SystemPrompt = newConfig.SystemPrompt
	}

	// Update category prompts
	if newConfig.CategoryPrompts != nil && len(newConfig.CategoryPrompts) > 0 {
		runtimeConfig.CategoryPrompts = make(map[string]string)
		for k, v := range newConfig.CategoryPrompts {
			runtimeConfig.CategoryPrompts[k] = v
		}
	} else {
		// If nil or empty, set to nil to use defaults
		runtimeConfig.CategoryPrompts = nil
	}

	// Update category models
	if newConfig.CategoryModels != nil {
		runtimeConfig.CategoryModels = make(map[string]string)
		for k, v := range newConfig.CategoryModels {
			runtimeConfig.CategoryModels[k] = v
		}
	}

	runtimeConfig.StandardModel = newConfig.StandardModel
	runtimeConfig.PremiumModel = newConfig.PremiumModel

	// Update PerplexityEnabled (always update if provided, default to true on first init)
	if initialized {
		runtimeConfig.PerplexityEnabled = newConfig.PerplexityEnabled
	} else {
		// On first initialization, use provided value or default to true
		if newConfig.PerplexityEnabled {
			runtimeConfig.PerplexityEnabled = true
		} else {
			runtimeConfig.PerplexityEnabled = false
		}
	}

	return nil
}

// ConfigError represents a configuration validation error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message + " (field: " + e.Field + ")"
}
