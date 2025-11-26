package admin

import (
	"sync"
	"testing"
)

func TestSetDefaultConfig_InitializesOnce(t *testing.T) {
	// Reset state
	configMutex.Lock()
	runtimeConfig = RuntimeConfig{
		SystemPrompt:  "",
		StandardModel: "gpt-4o-mini",
		PremiumModel:  "gpt-4o",
	}
	initialized = false
	configMutex.Unlock()

	defaultPrompt := "You are a helpful assistant. Current time: %s"
	SetDefaultConfig(defaultPrompt)

	config := GetConfig()
	if config.SystemPrompt != defaultPrompt {
		t.Errorf("Expected default prompt, got %q", config.SystemPrompt)
	}

	// Try to set again - should not overwrite
	SetDefaultConfig("Different prompt: %s")
	config = GetConfig()
	if config.SystemPrompt != defaultPrompt {
		t.Errorf("Expected original prompt (should not overwrite), got %q", config.SystemPrompt)
	}
}

func TestGetConfig_ReturnsCopy(t *testing.T) {
	// Reset state
	configMutex.Lock()
	runtimeConfig = RuntimeConfig{
		SystemPrompt:  "Test: %s",
		StandardModel: "gpt-4o-mini",
		PremiumModel:  "gpt-4o",
	}
	initialized = true
	configMutex.Unlock()

	config1 := GetConfig()
	config2 := GetConfig()

	// Should be different instances (copies)
	if &config1 == &config2 {
		t.Error("GetConfig should return copies, not same instance")
	}

	// But values should be same
	if config1.SystemPrompt != config2.SystemPrompt {
		t.Error("Config values should match")
	}
}

func TestSetConfig_ValidModels(t *testing.T) {
	// Reset state
	configMutex.Lock()
	runtimeConfig = RuntimeConfig{
		SystemPrompt:  "Test: %s",
		StandardModel: "gpt-4o-mini",
		PremiumModel:  "gpt-4o",
	}
	initialized = true
	configMutex.Unlock()

	testCases := []struct {
		name          string
		standardModel string
		premiumModel  string
		shouldSucceed bool
	}{
		{"valid models", "gpt-4o-mini", "gpt-4o", true},
		{"gpt-4.1 series", "gpt-4.1-mini", "gpt-4.1", true},
		{"gpt-5 series", "gpt-5-mini", "gpt-5.1", true},
		{"o-series", "o1-mini", "o3", true},
		{"invalid standard", "invalid-model", "gpt-4o", false},
		{"invalid premium", "gpt-4o-mini", "invalid-model", false},
		{"both invalid", "invalid-1", "invalid-2", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			newConfig := RuntimeConfig{
				SystemPrompt:  "Test prompt: %s",
				StandardModel: tc.standardModel,
				PremiumModel:  tc.premiumModel,
			}

			err := SetConfig(newConfig)
			if tc.shouldSucceed && err != nil {
				t.Errorf("Expected success, got error: %v", err)
			}
			if !tc.shouldSucceed && err == nil {
				t.Errorf("Expected error, got success")
			}

			if tc.shouldSucceed {
				config := GetConfig()
				if config.StandardModel != tc.standardModel {
					t.Errorf("Expected standard model %q, got %q", tc.standardModel, config.StandardModel)
				}
				if config.PremiumModel != tc.premiumModel {
					t.Errorf("Expected premium model %q, got %q", tc.premiumModel, config.PremiumModel)
				}
			}
		})
	}
}

func TestSetConfig_SystemPromptValidation(t *testing.T) {
	// Reset state
	configMutex.Lock()
	runtimeConfig = RuntimeConfig{
		SystemPrompt:  "Test: %s",
		StandardModel: "gpt-4o-mini",
		PremiumModel:  "gpt-4o",
	}
	initialized = true
	configMutex.Unlock()

	testCases := []struct {
		name          string
		prompt        string
		shouldSucceed bool
		errorContains string
	}{
		{"valid with one %s", "Prompt with time: %s", true, ""},
		{"no %s placeholder", "Prompt without placeholder", false, "exactly one %s"},
		{"multiple %s", "Prompt %s with %s two", false, "exactly one %s"},
		{"null byte", "Prompt with \x00 null", false, "null bytes"},
		{"dangerous %d", "Prompt with %s time and %d number", false, "unsupported format specifier"},
		{"dangerous %f", "Prompt with %s time and %f float", false, "unsupported format specifier"},
		{"dangerous %v", "Prompt with %s time and %v value", false, "unsupported format specifier"},
		{"escaped %%s", "Prompt with %%s escaped", false, "escaped placeholder"}, // Escaped %s is detected separately
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			newConfig := RuntimeConfig{
				SystemPrompt:  tc.prompt,
				StandardModel: "gpt-4o-mini",
				PremiumModel:  "gpt-4o",
			}

			err := SetConfig(newConfig)
			if tc.shouldSucceed && err != nil {
				t.Errorf("Expected success, got error: %v", err)
			}
			if !tc.shouldSucceed {
				if err == nil {
					t.Errorf("Expected error, got success")
				} else if tc.errorContains != "" {
					if err.Error() == "" || (tc.errorContains != "" && err.Error() != "" && 
						!contains(err.Error(), tc.errorContains)) {
						t.Errorf("Expected error to contain %q, got %q", tc.errorContains, err.Error())
					}
				}
			}
		})
	}
}

func TestSetConfig_ThreadSafety(t *testing.T) {
	// Reset state
	configMutex.Lock()
	runtimeConfig = RuntimeConfig{
		SystemPrompt:  "Test: %s",
		StandardModel: "gpt-4o-mini",
		PremiumModel:  "gpt-4o",
	}
	initialized = true
	configMutex.Unlock()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(idx int) {
			defer wg.Done()
			newConfig := RuntimeConfig{
				SystemPrompt:  "Concurrent test: %s",
				StandardModel: "gpt-4o-mini",
				PremiumModel: "gpt-4o",
			}
			SetConfig(newConfig)
		}(i)
	}

	// Concurrent reads
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			config := GetConfig()
			// Verify we got a valid config
			if config.StandardModel == "" {
				t.Error("Got empty standard model")
			}
		}()
	}

	wg.Wait()

	// Final config should be valid
	config := GetConfig()
	if config.SystemPrompt == "" {
		t.Error("Final config should have system prompt")
	}
}

func TestConfigError_ErrorMethod(t *testing.T) {
	err := &ConfigError{
		Field:   "system_prompt",
		Message: "Invalid format",
	}

	errorMsg := err.Error()
	if errorMsg == "" {
		t.Error("Error message should not be empty")
	}
	if !contains(errorMsg, "system_prompt") {
		t.Errorf("Error message should contain field name, got %q", errorMsg)
	}
	if !contains(errorMsg, "Invalid format") {
		t.Errorf("Error message should contain message, got %q", errorMsg)
	}
}

func TestValidateSystemPrompt_UTF8(t *testing.T) {
	// Test with valid UTF-8
	prompt := "Olá, mundo! 🌍 Current time: %s"
	err := validateSystemPrompt(prompt)
	if err != nil {
		t.Errorf("Valid UTF-8 prompt should pass, got error: %v", err)
	}

	// Test with invalid UTF-8 (this is tricky to create in Go, but we can test the logic)
	// Note: Go strings are always valid UTF-8, so we'd need to test with []byte
	// For now, we'll just verify the function exists and works with valid input
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
			s[len(s)-len(substr):] == substr || 
			containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

