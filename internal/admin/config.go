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
	BaseSystemPrompt string            `json:"base_system_prompt"`    // Core principles (shared by all)
	CategoryPrompts  map[string]string `json:"category_prompts"`      // category -> prompt override (optional)
	StandardModel    string            `json:"standard_model"`        // Fast/cheap model (e.g., gpt-4o-mini)
	PremiumModel     string            `json:"premium_model"`         // Powerful model (e.g., gpt-4.1)
	CategoryModels   map[string]string `json:"category_models"`       // category -> model override (optional)
	
	// Legacy field for backward compatibility
	SystemPrompt string `json:"system_prompt,omitempty"`
}

var (
	configMutex sync.RWMutex
	runtimeConfig = RuntimeConfig{
		BaseSystemPrompt: "", // Will be initialized with default from main.go
		CategoryPrompts:  make(map[string]string),
		CategoryModels:   make(map[string]string),
		StandardModel:    "gpt-4o-mini",
		PremiumModel:     "gpt-4o-mini", // Default to gpt-4o-mini for cost efficiency
	}
	initialized = false
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

// GetConfig returns a copy of the current runtime configuration
func GetConfig() RuntimeConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	
	// Deep copy maps
	categoryPrompts := make(map[string]string)
	for k, v := range runtimeConfig.CategoryPrompts {
		categoryPrompts[k] = v
	}
	
	categoryModels := make(map[string]string)
	for k, v := range runtimeConfig.CategoryModels {
		categoryModels[k] = v
	}
	
	return RuntimeConfig{
		BaseSystemPrompt: runtimeConfig.BaseSystemPrompt,
		CategoryPrompts:  categoryPrompts,
		CategoryModels:   categoryModels,
		StandardModel:    runtimeConfig.StandardModel,
		PremiumModel:     runtimeConfig.PremiumModel,
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
	// All models that support Responses API - can be used in either list
	validModels := map[string]bool{
		// GPT-4o series (confirmed working)
		"gpt-4o":             true,
		"gpt-4o-mini":        true,
		"gpt-4o-2024-08-06":  true,
		"chatgpt-4o-latest":  true,
		// GPT-4 series
		"gpt-4-turbo":        true,
		"gpt-3.5-turbo":      true,
		// GPT-4.1 series (may require specific API access)
		"gpt-4.1":            true,
		"gpt-4.1-mini":       true,
		"gpt-4.1-nano":       true,
		// GPT-5 series (may require specific API access)
		"gpt-5":              true,
		"gpt-5.1":            true,
		"gpt-5-mini":         true,
		"gpt-5-nano":         true,
		"gpt-5-pro":          true,
		// O-series reasoning models
		"o1":                 true,
		"o1-mini":            true,
		"o1-pro":             true,
		"o3":                 true,
		"o3-mini":            true,
		"o4-mini":            true,
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
	if newConfig.CategoryPrompts != nil {
		runtimeConfig.CategoryPrompts = make(map[string]string)
		for k, v := range newConfig.CategoryPrompts {
			runtimeConfig.CategoryPrompts[k] = v
		}
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

