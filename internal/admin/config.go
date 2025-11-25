package admin

import (
	"sync"
)

// RuntimeConfig holds runtime configuration that can be changed via admin UI
type RuntimeConfig struct {
	SystemPrompt  string `json:"system_prompt"`
	StandardModel string `json:"standard_model"` // Fast/cheap model (e.g., gpt-4o-mini)
	PremiumModel  string `json:"premium_model"`  // Powerful model (e.g., gpt-4.1)
}

var (
	configMutex sync.RWMutex
	runtimeConfig = RuntimeConfig{
		SystemPrompt:  "", // Will be initialized with default from main.go
		StandardModel: "gpt-4o-mini",
		PremiumModel:  "gpt-4.1",
	}
	initialized = false
)

// SetDefaultConfig initializes the config with default values from main.go
// This should be called once at startup with the default system prompt template
func SetDefaultConfig(defaultSystemPrompt string) {
	configMutex.Lock()
	defer configMutex.Unlock()
	
	if !initialized {
		runtimeConfig.SystemPrompt = defaultSystemPrompt
		initialized = true
	}
}

// GetConfig returns a copy of the current runtime configuration
func GetConfig() RuntimeConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	
	return RuntimeConfig{
		SystemPrompt:  runtimeConfig.SystemPrompt,
		StandardModel: runtimeConfig.StandardModel,
		PremiumModel:  runtimeConfig.PremiumModel,
	}
}

// SetConfig updates the runtime configuration
// Returns error if validation fails
func SetConfig(newConfig RuntimeConfig) error {
	// Validate model names - all models that support Responses API
	validStandardModels := map[string]bool{
		// Fast/cheap models
		"gpt-4.1-nano":  true,
		"gpt-4.1-mini":  true,
		"gpt-4o-mini":   true,
		"gpt-3.5-turbo": true,
		"gpt-5-nano":    true,
		"gpt-5-mini":    true,
	}
	
	validPremiumModels := map[string]bool{
		// Powerful/complex models
		"gpt-4.1":            true,
		"gpt-4o":             true,
		"gpt-4o-2024-08-06":  true,
		"chatgpt-4o-latest":  true,
		"gpt-4-turbo":        true,
		"o4-mini":            true,
		"o3":                 true,
		"o3-mini":            true,
		"gpt-5":              true,
		"gpt-5.1":            true,
		"gpt-5-pro":          true,
	}
	
	if !validStandardModels[newConfig.StandardModel] {
		return &ConfigError{Field: "standard_model", Message: "Invalid standard model"}
	}
	
	if !validPremiumModels[newConfig.PremiumModel] {
		return &ConfigError{Field: "premium_model", Message: "Invalid premium model"}
	}
	
	configMutex.Lock()
	defer configMutex.Unlock()
	
	runtimeConfig.SystemPrompt = newConfig.SystemPrompt
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

